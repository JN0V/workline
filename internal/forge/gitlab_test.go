package forge

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// GitLab is reached through its REST API alone, with the token and the
// project CI gives: no CLI to install, on gitlab.com or an instance of one's
// own.
func TestGitLabAPI(t *testing.T) {
	var got []string // what the forge was asked to write, "METHOD path field=value"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("PRIVATE-TOKEN") != "secret" {
			http.Error(w, `{"message":"401 Unauthorized"}`, http.StatusUnauthorized)
			return
		}
		p := strings.TrimPrefix(r.URL.EscapedPath(), "/api/v4/projects/group%2Fproj")
		switch {
		case r.Method == "GET" && p == "/merge_requests/7/notes" && r.URL.Query().Get("page") != "2":
			w.Header().Set("X-Next-Page", "2")
			json.NewEncoder(w).Encode([]map[string]any{{"id": 1, "body": "someone's"}})
		case r.Method == "GET" && p == "/merge_requests/7/notes":
			json.NewEncoder(w).Encode([]map[string]any{{"id": 2, "body": "old <!-- workline:k -->"}})
		case r.Method == "GET" && p == "/merge_requests/7":
			json.NewEncoder(w).Encode(map[string]any{"source_branch": "feat", "source_project_id": 3, "target_project_id": 3})
		case r.Method == "GET" && p == "/releases/v1.0.0":
			http.Error(w, `{"message":"404 Not Found"}`, http.StatusNotFound)
		case r.Method == "PUT" || r.Method == "POST":
			body, _ := io.ReadAll(r.Body)
			form, _ := url.ParseQuery(string(body))
			var fields []string
			for k := range form {
				fields = append(fields, k+"="+form.Get(k))
			}
			got = append(got, r.Method+" "+p+" "+strings.Join(fields, " "))
			w.Write([]byte(`{"iid": 9}`))
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

	// A sticky note on the second page is edited, not added again.
	if err := g.Sticky(Target{Kind: "merge-request", ID: 7}, "new", "<!-- workline:k -->", true); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "PUT /merge_requests/7/notes/2 body=new\n\n<!-- workline:k -->" {
		t.Fatalf("sticky: %q", got)
	}
	// A release the forge does not have yet is created.
	got = nil
	if err := g.Release("v1.0.0", "notes"); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !strings.HasPrefix(got[0], "POST /releases ") {
		t.Fatalf("release: %q", got)
	}
	// A merge request's branch, and whether it lives in this project.
	branch, here, err := g.MergeRequestBranch(7)
	if err != nil || branch != "feat" || !here {
		t.Fatalf("branch: %q %v %v", branch, here, err)
	}
	// A refused token is the forge out of reach, not the role's fault.
	t.Setenv("GITLAB_TOKEN", "wrong")
	if _, err := (&gitlab{repo: t.TempDir()}).Issue(1); err == nil || !strings.Contains(err.Error(), ErrUnreachable.Error()) {
		t.Fatalf("a refused token: %v", err)
	}
}

// Outside CI, the instance and the project come from the repository's remote.
func TestGitLabFromRemote(t *testing.T) {
	for _, c := range []struct{ remote, api, project string }{
		{"git@gitlab.example.com:group/sub/proj.git", "https://gitlab.example.com/api/v4", "group%2Fsub%2Fproj"},
		{"https://gitlab.com/JN0V/workline-sandbox.git", "https://gitlab.com/api/v4", "JN0V%2Fworkline-sandbox"},
		{"https://oauth2:tok@git.acme.local:8443/a/b", "https://git.acme.local:8443/api/v4", "a%2Fb"},
		{"ssh://git@gitlab.acme.io:2222/team/app.git", "https://gitlab.acme.io/api/v4", "team%2Fapp"},
	} {
		api, project, ok := fromRemote(c.remote)
		if !ok || api != c.api || project != c.project {
			t.Errorf("%s: %q %q %v; want %q %q", c.remote, api, project, ok, c.api, c.project)
		}
	}
}
