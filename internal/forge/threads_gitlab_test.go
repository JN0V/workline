package forge

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// gitlabLineThreads reads merge request 5's threads from a simulated
// GitLab whose project has an owner (50) and a guest (10) as members, and
// whose discussions on the diff hold one line comment each: by the owner,
// by the guest, and by a stranger who is no member.
func gitlabLineThreads(t *testing.T) []Thread {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.EscapedPath(), "/api/v4/projects/group%2Fproj")
		note := func(id int, body, user string) map[string]any {
			return map[string]any{"id": id, "body": body, "created_at": "2026-01-01T00:00:00Z",
				"author":   map[string]any{"username": user},
				"position": map[string]any{"new_path": "a_test.go", "new_line": id}}
		}
		switch p {
		case "/members/all":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"username": "owner", "access_level": 50, "state": "active"},
				{"username": "guest", "access_level": 10, "state": "active"},
			})
		case "/merge_requests/5":
			_ = json.NewEncoder(w).Encode(map[string]any{"web_url": "https://gitlab.example/group/proj/-/merge_requests/5"})
		case "/merge_requests/5/discussions":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"id": "d1", "notes": []map[string]any{note(1, "a test is missing here", "owner")}},
				{"id": "d2", "notes": []map[string]any{note(2, "what about this?", "guest")}},
				{"id": "d3", "notes": []map[string]any{note(3, "and this?", "stranger")}},
			})
		default:
			http.Error(w, `{"message":"404 Not Found"}`, http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	t.Setenv("CI_API_V4_URL", srv.URL+"/api/v4")
	t.Setenv("CI_PROJECT_ID", "")
	t.Setenv("CI_PROJECT_PATH", "group/proj")
	t.Setenv("GITLAB_TOKEN", "secret")
	threads, err := (&gitlab{repo: t.TempDir()}).Threads(5)
	if err != nil {
		t.Fatalf("Threads: %v", err)
	}
	if len(threads) != 3 {
		t.Fatalf("Threads: %d threads, want 3: %+v", len(threads), threads)
	}
	return threads
}

// Person's line comment: Given a GitLab merge request with a person's
// comment on a line, When the threads are read, Then that comment is
// marked as from an insider, and is no bot's, so the merge is held by it.
func TestGitLabThreadsPersonsLineComment(t *testing.T) {
	threads := gitlabLineThreads(t)
	c := threads[0].Comments[0]
	if c.Author != "owner" || !c.Insider || c.Bot {
		t.Errorf("the owner's line comment: %+v, want an insider's, no bot's", c)
	}
}

// Outsider's comment: Given a comment on a line by someone with no access
// to the project, When the threads are read, Then it is not marked as an
// insider's: neither a guest's (below Planner) nor a stranger's, while the
// member's comment beside them is.
func TestGitLabThreadsOutsidersComment(t *testing.T) {
	threads := gitlabLineThreads(t)
	if !threads[0].Comments[0].Insider {
		t.Errorf("the member's comment is not marked an insider's, so the outsiders' not being so tells nothing: %+v", threads[0].Comments[0])
	}
	for _, th := range threads[1:] {
		if c := th.Comments[0]; c.Insider {
			t.Errorf("%s's comment is marked an insider's: %+v", c.Author, c)
		}
	}
}
