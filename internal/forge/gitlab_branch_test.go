package forge

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// GitLab: given a GitLab project, when the line closes a merge request
// without merging, then its source branch is deleted: the branch of
// another merge request is not.
func TestGitLabCloseMergeRequestDeletesTheBranch(t *testing.T) {
	var deleted, closed []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.EscapedPath(), "/api/v4/projects/group%2Fproj")
		switch {
		case r.Method == http.MethodGet && p == "/merge_requests":
			all := []map[string]any{
				{"iid": 6, "source_branch": "workline/tester/42"},
				{"iid": 7, "source_branch": "feat/other"},
			}
			var list []map[string]any
			for _, m := range all {
				if b := r.URL.Query().Get("source_branch"); b == "" || b == m["source_branch"] {
					list = append(list, m)
				}
			}
			_ = json.NewEncoder(w).Encode(list)
		case r.Method == http.MethodGet && strings.HasSuffix(p, "/notes"):
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodPost && strings.HasSuffix(p, "/notes"):
			_, _ = w.Write([]byte(`{}`))
		case r.Method == http.MethodPut && strings.HasPrefix(p, "/merge_requests/"):
			closed = append(closed, p)
			_, _ = w.Write([]byte(`{}`))
		case r.Method == http.MethodDelete:
			if u, err := url.PathUnescape(p); err == nil {
				p = u
			}
			deleted = append(deleted, p)
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, `{"message":"404 Not Found"}`, http.StatusNotFound)
		}
	}))
	defer srv.Close()
	t.Setenv("CI_API_V4_URL", srv.URL+"/api/v4")
	t.Setenv("CI_PROJECT_ID", "")
	t.Setenv("CI_PROJECT_PATH", "group/proj")
	t.Setenv("GITLAB_TOKEN", "secret")
	g := &gitlab{repo: t.TempDir()}
	if err := g.CloseMergeRequest("workline/tester/42", "Closed by workline: superseded.", "<!-- workline:closed -->"); err != nil {
		t.Fatal(err)
	}
	if len(closed) != 1 || closed[0] != "/merge_requests/6" {
		t.Errorf("closed: %q, want the merge request 6 alone", closed)
	}
	if len(deleted) != 1 || deleted[0] != "/repository/branches/workline/tester/42" {
		t.Errorf("deleted: %q, want the branch workline/tester/42 alone", deleted)
	}
}
