package forge

import (
	"encoding/json"
	"fmt"
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
			json.NewEncoder(w).Encode(map[string]any{"source_branch": "feat", "target_branch": "main", "source_project_id": 3, "target_project_id": 3})
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
	// A merge request's branch, and whether it lives in this project.
	mr, err := g.MergeRequest(7)
	if err != nil || mr.Branch != "feat" || mr.Base != "main" || !mr.Here {
		t.Fatalf("branch: %+v %v", mr, err)
	}
	// A refused token is the forge out of reach, not the role's fault.
	t.Setenv("GITLAB_TOKEN", "wrong")
	if _, err := (&gitlab{repo: t.TempDir()}).Issue(1); err == nil || !strings.Contains(err.Error(), ErrUnreachable.Error()) {
		t.Fatalf("a refused token: %v", err)
	}
}

// Who is of the project comes from its members' access level: Planner and
// above; a project access token's user is a bot. A note another user wrote
// cannot be edited (403): the sticky one is written anew after it.
func TestGitLabWho(t *testing.T) {
	var got []string
	members := true
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.EscapedPath(), "/api/v4/projects/group%2Fproj")
		author := func(u string) map[string]any { return map[string]any{"username": u} }
		switch {
		case p == "/members/all" && !members:
			http.Error(w, `{"message":"403 Forbidden"}`, http.StatusForbidden)
		case p == "/members/all":
			json.NewEncoder(w).Encode([]map[string]any{
				{"username": "owner", "access_level": 50, "state": "active"},
				{"username": "plan", "access_level": 15, "state": "active"},
				{"username": "guest", "access_level": 10, "state": "active"},
				{"username": "gone", "access_level": 30, "state": "blocked"},
				{"username": "project_1_bot_0a1b", "access_level": 15, "state": "active"},
			})
		case r.Method == "GET" && p == "/issues/3":
			json.NewEncoder(w).Encode(map[string]any{"iid": 3, "title": "t", "author": author("guest"), "state": "opened"})
		case r.Method == "GET" && p == "/issues/3/notes":
			json.NewEncoder(w).Encode([]map[string]any{
				{"id": 1, "body": "state <!-- workline:k -->", "author": author("owner")},
				{"id": 2, "body": "changed the description", "system": true, "author": author("plan")},
				{"id": 3, "body": "agreed", "author": author("guest")},
				{"id": 4, "body": "ok", "author": author("plan")},
				{"id": 5, "body": "done", "author": author("project_1_bot_0a1b")},
				{"id": 6, "body": "me too", "author": author("gone")},
			})
		case r.Method == "PUT" && p == "/issues/3/notes/1":
			http.Error(w, `{"message":"403 Forbidden"}`, http.StatusForbidden)
		case r.Method == "POST":
			body, _ := io.ReadAll(r.Body)
			form, _ := url.ParseQuery(string(body))
			got = append(got, r.Method+" "+p+" "+form.Get("body"))
			w.Write([]byte(`{"id": 7}`))
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

	is, err := g.Issue(3)
	if err != nil || is.Author != "guest" || is.Insider {
		t.Fatalf("a guest's issue: %+v %v", is, err)
	}
	notes, err := g.Notes(Target{Kind: "issue", ID: 3})
	if err != nil {
		t.Fatal(err)
	}
	want := []Note{
		{Body: "state <!-- workline:k -->", Author: "owner", Insider: true},
		{Body: "agreed", Author: "guest"},
		{Body: "ok", Author: "plan", Insider: true},
		{Body: "done", Author: "project_1_bot_0a1b", Insider: true, Bot: true},
		{Body: "me too", Author: "gone"},
	}
	if fmt.Sprint(notes) != fmt.Sprint(want) {
		t.Fatalf("notes:\n%+v\nwant\n%+v", notes, want)
	}
	if err := g.Sticky(Target{Kind: "issue", ID: 3}, "new", "<!-- workline:k -->", false); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "POST /issues/3/notes new\n\n<!-- workline:k -->" {
		t.Fatalf("a sticky note another user wrote: %q", got)
	}
	// Members not readable: said, never every reporter taken for an outsider.
	members = false
	if _, err := (&gitlab{repo: t.TempDir()}).Issue(3); err == nil || !strings.Contains(err.Error(), "members") {
		t.Fatalf("members refused: %v", err)
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
