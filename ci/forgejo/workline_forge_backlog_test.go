package forgejo

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"sync"
	"testing"
)

// serveScript runs the script on one operation against a handler, and
// returns its answer and each call it made ("METHOD path?query body").
func serveScript(t *testing.T, h http.HandlerFunc, op string, req any) (string, []string) {
	t.Helper()
	for _, tool := range []string{"sh", "curl", "jq"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skip(tool + " is not installed")
		}
	}
	var mu sync.Mutex
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		calls = append(calls, strings.TrimSpace(r.Method+" "+r.URL.RequestURI()+" "+string(body)))
		mu.Unlock()
		r.Body = io.NopCloser(strings.NewReader(string(body)))
		h(w, r)
	}))
	defer srv.Close()
	in, _ := json.Marshal(req)
	cmd := exec.Command("sh", "workline-forge.sh")
	cmd.Stdin = strings.NewReader(string(in))
	cmd.Env = []string{"PATH=" + lookPath(), "WORKLINE_FORGE_OPERATION=" + op,
		"FORGEJO_URL=" + srv.URL, "FORGEJO_TOKEN=t", "FORGEJO_REPO=owner/name"}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s: %v\n%s\ncalls: %q", op, err, out, calls)
	}
	return strings.TrimSpace(string(out)), calls
}

// Backlog read: Given a Forgejo or Gitea repository with open issues, When
// the product owner runs `issues` against the sample command, Then it lists
// the open issues, over every page, a pull request left out, and no
// unknown operation error appears.
func TestBacklogRead(t *testing.T) {
	const repo = "/api/v1/repos/owner/name"
	out, calls := serveScript(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != repo+"/issues" || r.URL.Query().Get("state") != "open" {
			http.Error(w, "not mocked", http.StatusNotFound)
			return
		}
		switch r.URL.Query().Get("page") {
		case "1":
			_, _ = w.Write([]byte(`[{"number": 1, "title": "One", "body": "b1", "state": "open", "labels": [{"name": "bug"}]},
				{"number": 2, "title": "A pull request", "state": "open", "labels": [], "pull_request": {"merged": false}}]`))
		case "2":
			_, _ = w.Write([]byte(`[{"number": 3, "title": "Three", "body": null, "state": "open", "labels": []}]`))
		case "3":
			_, _ = w.Write([]byte(`[{"number": 4, "title": "Four", "body": "b4", "state": "open", "labels": []}]`))
		default:
			_, _ = w.Write([]byte(`[]`))
		}
	}, "issues", map[string]any{})
	var got struct {
		Issues []struct {
			ID     int      `json:"id"`
			Title  string   `json:"title"`
			Body   string   `json:"body"`
			Labels []string `json:"labels"`
		} `json:"issues"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil || strings.Contains(out, "unknown operation") {
		t.Fatalf("answer %q: %v; calls: %q", out, err, calls)
	}
	ids := ""
	for _, is := range got.Issues {
		ids += string(rune('0' + is.ID))
	}
	if ids != "134" {
		t.Fatalf("issue ids = %q, want 134 (every page, no pull request); answer %s", ids, out)
	}
	if got.Issues[0].Title != "One" || got.Issues[0].Body != "b1" || len(got.Issues[0].Labels) != 1 || got.Issues[0].Labels[0] != "bug" {
		t.Errorf("first issue = %+v", got.Issues[0])
	}
	if got.Issues[1].Body != "" {
		t.Errorf("a null body = %q, want empty", got.Issues[1].Body)
	}
}

// Close and milestone: Given an open issue, When workline closes it or
// sets a milestone through the sample, Then the issue is closed with
// PATCH {"state": "closed"}, or put in the open milestone found by its
// title, its number sent in PATCH /issues/<id>.
func TestCloseAndMilestone(t *testing.T) {
	const repo = "/api/v1/repos/owner/name"
	handler := func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && (r.URL.Path == repo+"/issues/7" || r.URL.Path == repo+"/issues/8"):
			_, _ = w.Write([]byte(`{"number": 7, "title": "T", "body": "", "state": "open", "labels": [], "milestone": null}`))
		case r.Method == "GET" && r.URL.Path == repo+"/milestones":
			if r.URL.Query().Get("page") == "1" || r.URL.Query().Get("page") == "" {
				_, _ = w.Write([]byte(`[{"id": 3, "title": "v1", "due_on": null}, {"id": 4, "title": "v2", "due_on": null}]`))
			} else {
				_, _ = w.Write([]byte(`[]`))
			}
		case r.Method == "PATCH":
			_, _ = w.Write([]byte(`{}`))
		default:
			http.Error(w, "not mocked", http.StatusNotFound)
		}
	}
	patched := func(calls []string, path string) map[string]any {
		for _, c := range calls {
			if rest, ok := strings.CutPrefix(c, "PATCH "+repo+path+" "); ok {
				var v map[string]any
				if err := json.Unmarshal([]byte(rest), &v); err != nil {
					t.Fatalf("PATCH body %q: %v", rest, err)
				}
				return v
			}
		}
		return nil
	}
	out, calls := serveScript(t, handler, "close", map[string]any{"id": 7})
	if out != "{}" {
		t.Errorf("close answer = %q, want {}", out)
	}
	if v := patched(calls, "/issues/7"); v["state"] != "closed" {
		t.Errorf("close sent %v; calls: %q", v, calls)
	}
	out, calls = serveScript(t, handler, "set-milestone", map[string]any{"id": 8, "milestone": "v2"})
	if out != "{}" {
		t.Errorf("set-milestone answer = %q, want {}", out)
	}
	if v := patched(calls, "/issues/8"); v["milestone"] != float64(4) {
		t.Errorf("set-milestone sent %v, want milestone 4; calls: %q", v, calls)
	}
}

// Closed state: Given a closed issue, When workline reads it with `issue`,
// Then the answer says it is closed.
func TestClosedState(t *testing.T) {
	out, calls := serveScript(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/api/v1/repos/owner/name/issues/4" {
			_, _ = w.Write([]byte(`{"number": 4, "title": "Done", "body": "b", "state": "closed", "labels": []}`))
			return
		}
		http.Error(w, "not mocked", http.StatusNotFound)
	}, "issue", map[string]any{"id": 4})
	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("answer %q: %v", out, err)
	}
	if got["closed"] != true {
		t.Errorf("closed = %v, want true; answer %s; calls: %q", got["closed"], out, calls)
	}
}

// Merge request base: Given an open pull request, When workline reads its
// branch, Then the answer names the base branch.
func TestMergeRequestBase(t *testing.T) {
	out, calls := serveScript(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/api/v1/repos/owner/name/pulls/6" {
			_, _ = w.Write([]byte(`{"number": 6, "title": "T", "body": "b", "head": {"ref": "feat/x", "repo": {"full_name": "owner/name"}},
				"base": {"ref": "release/1.x", "repo": {"full_name": "owner/name"}}}`))
			return
		}
		http.Error(w, "not mocked", http.StatusNotFound)
	}, "merge-request-branch", map[string]any{"id": 6})
	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("answer %q: %v", out, err)
	}
	if got["base"] != "release/1.x" {
		t.Errorf("base = %v, want release/1.x; answer %s; calls: %q", got["base"], out, calls)
	}
}
