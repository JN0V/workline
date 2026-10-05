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
	mu    sync.Mutex
	calls []string // "METHOD path body"
}

func (m *mock) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	m.mu.Lock()
	m.calls = append(m.calls, strings.TrimSpace(r.Method+" "+r.URL.Path+" "+string(body)))
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
			fmt.Fprint(w, `[{"body": "Rows lost.", "user": {"login": "zed"}}, {"body": "agreed", "user": {"login": "dev"}}]`)
		} else {
			fmt.Fprint(w, `[]`)
		}
	case r.Method == "GET" && r.URL.Path == repo+"/collaborators/dev/permission":
		fmt.Fprint(w, `{"permission": "write"}`)
	case r.Method == "GET" && r.URL.Path == repo+"/collaborators/zed/permission":
		http.Error(w, `{"message": "not a collaborator"}`, http.StatusForbidden)
	default:
		http.Error(w, "not mocked", http.StatusNotFound)
	}
}

// run runs the script on one operation against the mock, and returns its answer.
func run(t *testing.T, m *mock, op string, req any) string {
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
	if err != nil {
		t.Fatalf("%s: %v\n%s\ncalls: %q", op, err, out, m.calls)
	}
	return strings.TrimSpace(string(out))
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

// comments answers each comment with its author, and whether they may
// write to the repository.
func TestComments(t *testing.T) {
	out := run(t, &mock{}, "comments", map[string]any{"target": map[string]any{"kind": "issue", "id": 3}})
	want := `{"comments":[{"body":"Rows lost.","author":"zed","insider":false},{"body":"agreed","author":"dev","insider":true}]}`
	if out != want {
		t.Errorf("answer = %s\nwant     %s", out, want)
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

func lookPath() string {
	var dirs []string
	for _, tool := range []string{"sh", "curl", "jq", "git", "cat", "sed"} {
		if p, err := exec.LookPath(tool); err == nil {
			dirs = append(dirs, p[:strings.LastIndex(p, "/")])
		}
	}
	return strings.Join(slices.Compact(slices.Sorted(slices.Values(dirs))), ":")
}
