// Package forgejo tests the sample forge command against a mock of the
// Forgejo and Gitea API: what the script asks of it, request by request.
package forgejo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"testing"
)

// mock answers the API calls the label and all-issues operations make,
// and records them.
type mock struct {
	refuse bool // the permission lookup refused: a token that may not ask
	mu     sync.Mutex
	calls  []string // "METHOD path body"
}

func (m *mock) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	m.mu.Lock()
	m.calls = append(m.calls, strings.TrimSpace(r.Method+" "+r.URL.EscapedPath()+" "+string(body)))
	m.mu.Unlock()
	const repo = "/api/v1/repos/owner/name"
	switch {
	case r.Method == "GET" && r.URL.Path == repo+"/labels":
		if r.URL.Query().Get("page") == "1" {
			fmt.Fprint(w, `[{"id": 1, "name": "needs review"}]`)
		} else {
			fmt.Fprint(w, `[]`)
		}
	case r.Method == "POST" && r.URL.Path == repo+"/labels":
		fmt.Fprint(w, `{"id": 2}`)
	case r.Method == "POST" && r.URL.Path == repo+"/issues/3/labels":
		fmt.Fprint(w, `[]`)
	case r.Method == "GET" && r.URL.Path == repo+"/issues/3/labels":
		fmt.Fprint(w, `[{"id": 5, "name": "workline: step 0"}, {"id": 6, "name": "workline:"}]`)
	case r.Method == "DELETE" && r.URL.Path == repo+"/issues/3/labels/5":
		w.WriteHeader(http.StatusNoContent)
	case r.Method == "GET" && r.URL.Path == repo+"/issues" &&
		r.URL.Query().Get("state") == "all" && r.URL.Query().Get("type") == "issues":
		switch r.URL.Query().Get("page") {
		case "1":
			fmt.Fprint(w, `[{"number": 4, "title": "Closed", "body": "b <!-- workline:k -->", "state": "closed", "labels": [{"id": 1, "name": "needs review"}]},
				{"number": 2, "title": "Open", "body": null, "state": "open", "labels": []}]`)
		case "2":
			fmt.Fprint(w, `[{"number": 3, "title": "A pull request", "state": "open", "labels": [], "pull_request": {"merged": false}}]`)
		default:
			fmt.Fprint(w, `[]`)
		}
	case r.Method == "GET" && r.URL.Path == repo+"/issues/3/comments":
		if r.URL.Query().Get("page") == "1" {
			fmt.Fprint(w, `[{"body": "Rows lost.", "user": {"id": 7, "login": "zed"}, "created_at": "2026-10-01T09:00:00+02:00"}, {"body": "agreed", "user": {"id": 8, "login": "dev"}},
				{"body": "agreed", "user": {"id": -2, "login": "forgejo-actions"}}, {"body": "agreed", "user": {"id": 9, "login": "renovate-bot"}},
				{"body": "old", "user": {"id": 10, "login": "gone"}}, {"body": "built", "user": {"id": 11, "login": "ci[bot]"}}]`)
		} else {
			fmt.Fprint(w, `[]`)
		}
	case r.Method == "GET" && r.URL.Path == repo+"/pulls" && r.URL.Query().Get("state") == "open":
		if r.URL.Query().Get("page") == "1" {
			here, main := `"repo": {"full_name": "owner/name"}`, `"base": {"ref": "main", "repo": {"full_name": "owner/name"}}`
			fmt.Fprint(w, `[{"number": 6, "title": "docs: suspect", "head": {"ref": "workline/documentalist/suspect", `+here+`}, `+main+`, "user": {"id": 3, "login": "workline-bot"}},
				{"number": 7, "title": "Two hours", "head": {"ref": "feat/other", `+here+`}, `+main+`, "user": {"id": 8, "login": "dev"}},
				{"number": 9, "title": "Typo", "head": {"ref": "patch-1", "repo": {"full_name": "zed/name"}}, `+main+`, "user": {"id": 7, "login": "zed"}},
				{"number": 10, "head": {"ref": "renovate/go", `+here+`}, `+main+`, "user": {"id": 9, "login": "renovate-bot"}},
				{"number": 11, "title": "Backport", "head": {"ref": "fix/x", `+here+`}, "base": {"ref": "release/1.x", `+here+`}, "user": {"id": 8, "login": "dev"}}]`)
		} else {
			fmt.Fprint(w, `[]`)
		}
	case r.Method == "GET" && r.URL.Path == "/api/v1/user":
		fmt.Fprint(w, `{"id": 11, "login": "ci[bot]"}`)
	case r.Method == "GET" && r.URL.Path == repo+"/issues/5/comments":
		fmt.Fprint(w, `[{"id": 50, "body": "old <!-- workline:k -->", "user": {"id": 11, "login": "ci[bot]"}},
			{"id": 51, "body": "another's <!-- workline:k -->", "user": {"id": 7, "login": "zed"}}]`)
	case r.Method == "PATCH" && strings.HasPrefix(r.URL.Path, repo+"/issues/comments/"), r.Method == "POST" && r.URL.Path == repo+"/issues/5/comments":
		fmt.Fprint(w, `{}`)
	case r.Method == "GET" && r.URL.Path == repo+"/issues/6/comments":
		fmt.Fprint(w, `[]`)
	case r.Method == "POST" && r.URL.Path == repo+"/issues/6/comments":
		fmt.Fprint(w, `{"id": 60}`)
	case r.Method == "PATCH" && r.URL.Path == repo+"/pulls/6":
		fmt.Fprint(w, `{"number": 6, "state": "closed"}`)
	case strings.HasPrefix(r.URL.Path, repo+"/collaborators/") && m.refuse:
		http.Error(w, `{"message": "Only admins can query all permissions"}`, http.StatusForbidden)
	case r.Method == "GET" && r.URL.Path == repo+"/collaborators/dev/permission":
		fmt.Fprint(w, `{"permission": "write"}`)
	case r.Method == "GET" && r.URL.Path == repo+"/collaborators/zed/permission",
		r.URL.Path == repo+"/collaborators/forgejo-actions/permission", r.URL.Path == repo+"/collaborators/renovate-bot/permission",
		r.URL.Path == repo+"/collaborators/ci[bot]/permission":
		fmt.Fprint(w, `{"permission": "read"}`)
	case r.Method == "GET" && r.URL.Path == repo+"/collaborators/gone/permission":
		http.Error(w, `{"message": "user does not exist"}`, http.StatusNotFound)
	default:
		http.Error(w, "not mocked", http.StatusNotFound)
	}
}

// run runs the script on one operation against the mock, and returns its answer.
func run(t *testing.T, m *mock, op string, req any) string {
	t.Helper()
	out, err := try(t, m, op, req)
	if err != nil {
		t.Fatalf("%s: %v\n%s\ncalls: %q", op, err, out, m.calls)
	}
	return out
}

// try runs the script on one operation, its exit said.
func try(t *testing.T, m *mock, op string, req any) (string, error) {
	t.Helper()
	for _, tool := range []string{"sh", "curl", "jq"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skip(tool + " is not installed")
		}
	}
	srv := httptest.NewServer(m)
	defer srv.Close()
	in, _ := json.Marshal(req)
	cmd := exec.Command("sh", "workline-forge.sh")
	cmd.Stdin = strings.NewReader(string(in))
	cmd.Env = []string{"PATH=" + lookPath(), "WORKLINE_FORGE_OPERATION=" + op,
		"FORGEJO_URL=" + srv.URL, "FORGEJO_TOKEN=t", "FORGEJO_REPO=owner/name"}
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// all-issues lists the issues open and closed, every page, pull requests
// left out; Forgejo keeps no close reason, so none is answered.
func TestAllIssues(t *testing.T) {
	out := run(t, &mock{}, "all-issues", map[string]any{})
	var got struct {
		Issues []map[string]any `json:"issues"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("answer %q: %v", out, err)
	}
	want := `[{"body":"b \u003c!-- workline:k --\u003e","closed":true,"id":4,"labels":["needs review"],"title":"Closed"},` +
		`{"body":"","closed":false,"id":2,"labels":[],"title":"Open"}]`
	if b, _ := json.Marshal(got.Issues); string(b) != want {
		t.Errorf("issues = %s\nwant     %s", b, want)
	}
}

// comments answers each comment with its author, whether they may write
// to the repository, whether the token's user wrote it, and when it was
// written, when Forgejo says.
func TestComments(t *testing.T) {
	m := &mock{}
	out := run(t, m, "comments", map[string]any{"target": map[string]any{"kind": "issue", "id": 3}})
	if !slices.Contains(m.calls, "GET /api/v1/repos/owner/name/collaborators/ci%5Bbot%5D/permission") {
		t.Errorf("a login not escaped in the URL; calls: %q", m.calls)
	}
	want := `{"comments":[{"body":"Rows lost.","author":"zed","insider":false,"bot":false,"own":false,"created":"2026-10-01T09:00:00+02:00"},` +
		`{"body":"agreed","author":"dev","insider":true,"bot":false,"own":false},` +
		`{"body":"agreed","author":"forgejo-actions","insider":false,"bot":true,"own":false},` +
		`{"body":"agreed","author":"renovate-bot","insider":false,"bot":true,"own":false},` +
		`{"body":"old","author":"gone","insider":false,"bot":false,"own":false},` +
		`{"body":"built","author":"ci[bot]","insider":false,"bot":true,"own":true}]}`
	if out != want {
		t.Errorf("answer = %s\nwant     %s", out, want)
	}
	// A lookup refused is said, never read as "outside the project".
	if out, err := try(t, &mock{refuse: true}, "comments", map[string]any{"target": map[string]any{"kind": "issue", "id": 3}}); err == nil {
		t.Errorf("a refused permission lookup passed: %s", out)
	}
}

// sticky edits the comment holding the marker; asked `own`, only the
// one the token's user wrote, another's left, never edited.
func TestStickyOwn(t *testing.T) {
	m := &mock{}
	req := map[string]any{"target": map[string]any{"kind": "merge-request", "id": 5}, "body": "new", "marker": "<!-- workline:k -->", "create": false}
	run(t, m, "sticky", req)
	if !slices.ContainsFunc(m.calls, func(c string) bool { return strings.HasPrefix(c, "PATCH /api/v1/repos/owner/name/issues/comments/50 ") }) {
		t.Errorf("the comment holding the marker not edited; calls: %q", m.calls)
	}
	m = &mock{}
	req["own"] = true
	run(t, m, "sticky", req)
	if !slices.ContainsFunc(m.calls, func(c string) bool { return strings.HasPrefix(c, "PATCH /api/v1/repos/owner/name/issues/comments/50 ") }) ||
		slices.ContainsFunc(m.calls, func(c string) bool { return strings.Contains(c, "comments/51") }) {
		t.Errorf("not the token's user's edited; calls: %q", m.calls)
	}
}

// Label names holding spaces are kept whole: added, created when missing,
// and removed by their full name.
func TestLabelsWithSpaces(t *testing.T) {
	m := &mock{}
	out := run(t, m, "label", map[string]any{"target": map[string]any{"kind": "issue", "id": 3},
		"add": []string{"needs review", "to refine"}, "remove": []string{"workline: step 0"}})
	if out != "{}" {
		t.Errorf("answer = %q, want {}", out)
	}
	for _, want := range []string{
		`POST /api/v1/repos/owner/name/labels {"name":"to refine","color":"#ededed"}`,
		`POST /api/v1/repos/owner/name/issues/3/labels {"labels":[1,2]}`,
		`DELETE /api/v1/repos/owner/name/issues/3/labels/5`,
	} {
		if !slices.Contains(m.calls, want) {
			t.Errorf("no call %q; calls: %q", want, m.calls)
		}
	}
	if slices.Contains(m.calls, `DELETE /api/v1/repos/owner/name/issues/3/labels/6`) {
		t.Errorf("a label named by a part of another's name was removed; calls: %q", m.calls)
	}
}

// close-merge-request closes the pull request open from the branch, after
// one comment saying why; another pull request is left alone.
func TestCloseMergeRequest(t *testing.T) {
	m := &mock{}
	out := run(t, m, "close-merge-request", map[string]any{"branch": "workline/documentalist/suspect",
		"body": "Closed by workline's documentalist: main holds its change already.", "marker": "<!-- workline:closed -->"})
	if out != "{}" {
		t.Errorf("answer = %q, want {}", out)
	}
	for _, want := range []string{
		`POST /api/v1/repos/owner/name/issues/6/comments {"body":"Closed by workline's documentalist: main holds its change already.\n\n<!-- workline:closed -->"}`,
		`PATCH /api/v1/repos/owner/name/pulls/6 {"state": "closed"}`,
	} {
		if !slices.Contains(m.calls, want) {
			t.Errorf("no call %q; calls: %q", want, m.calls)
		}
	}
	if slices.ContainsFunc(m.calls, func(c string) bool { return strings.Contains(c, "/pulls/7") }) {
		t.Errorf("another pull request was touched; calls: %q", m.calls)
	}
}

// merge-requests lists the open pull requests, each with where it comes
// from and goes, and who opened it (ADR-0041): a fork's not here, a bot
// told as for comments, every head where Forgejo keeps it.
func TestMergeRequests(t *testing.T) {
	out := run(t, &mock{}, "merge-requests", map[string]any{})
	want := `{"merge-requests":[` +
		`{"id":6,"branch":"workline/documentalist/suspect","base":"main","here":true,"title":"docs: suspect","author":"workline-bot","bot":true,"head":"refs/pull/6/head"},` +
		`{"id":7,"branch":"feat/other","base":"main","here":true,"title":"Two hours","author":"dev","bot":false,"head":"refs/pull/7/head"},` +
		`{"id":9,"branch":"patch-1","base":"main","here":false,"title":"Typo","author":"zed","bot":false,"head":"refs/pull/9/head"},` +
		`{"id":10,"branch":"renovate/go","base":"main","here":true,"title":"","author":"renovate-bot","bot":true,"head":"refs/pull/10/head"},` +
		`{"id":11,"branch":"fix/x","base":"release/1.x","here":true,"title":"Backport","author":"dev","bot":false,"head":"refs/pull/11/head"}]}`
	if out != want {
		t.Errorf("answer = %s\nwant     %s", out, want)
	}
}

func lookPath() string {
	var dirs []string
	for _, tool := range []string{"sh", "curl", "jq", "git", "cat", "sed"} {
		if p, err := exec.LookPath(tool); err == nil {
			dirs = append(dirs, p[:strings.LastIndex(p, "/")])
		}
	}
	return strings.Join(slices.Compact(slices.Sorted(slices.Values(dirs))), ":")
}
