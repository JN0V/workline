package forge

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

// glab's web login holds an OAuth token: it goes as a Bearer token; a
// personal access token in GITLAB_TOKEN as PRIVATE-TOKEN (#733).
func TestGitLabGlabOAuthToken(t *testing.T) {
	var sent []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sent = append(sent, "PRIVATE-TOKEN="+r.Header.Get("PRIVATE-TOKEN")+" Authorization="+r.Header.Get("Authorization"))
		_ = json.NewEncoder(w).Encode(map[string]any{"iid": 1, "title": "t", "state": "opened"})
	}))
	defer srv.Close()
	bin := t.TempDir()
	// FAKE_OAUTH: what glab says of its login; a personal token's says false.
	glab := "#!/bin/sh\ncase \"$3\" in token) echo glab-tok ;; is_oauth2) echo \"$FAKE_OAUTH\" ;; esac\n"
	t.Setenv("FAKE_OAUTH", "true")
	if err := os.WriteFile(filepath.Join(bin, "glab"), []byte(glab), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("CI_API_V4_URL", srv.URL+"/api/v4")
	t.Setenv("CI_PROJECT_ID", "")
	t.Setenv("CI_PROJECT_PATH", "group/proj")
	t.Setenv("CI_JOB_TOKEN", "")
	t.Setenv("GITLAB_TOKEN", "")
	if _, err := (&gitlab{repo: t.TempDir()}).api("projects/:id"); err != nil {
		t.Fatal(err)
	}
	// glab logged in with a personal access token: PRIVATE-TOKEN.
	t.Setenv("FAKE_OAUTH", "false")
	if _, err := (&gitlab{repo: t.TempDir()}).api("projects/:id"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITLAB_TOKEN", "glpat-x")
	t.Setenv("FAKE_OAUTH", "true") // GITLAB_TOKEN first, whatever glab holds
	if _, err := (&gitlab{repo: t.TempDir()}).api("projects/:id"); err != nil {
		t.Fatal(err)
	}
	want := []string{"PRIVATE-TOKEN= Authorization=Bearer glab-tok", "PRIVATE-TOKEN=glab-tok Authorization=", "PRIVATE-TOKEN=glpat-x Authorization="}
	if !slices.Equal(sent, want) {
		t.Fatalf("sent %q, want %q", sent, want)
	}
}

// Out of CI, the boards are the first call: the project's path comes
// from the remote all the same, never asked empty.
func TestGitLabBoardsFromRemote(t *testing.T) {
	var asked []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var q struct {
			Variables map[string]string `json:"variables"`
		}
		_ = json.NewDecoder(r.Body).Decode(&q)
		asked = append(asked, q.Variables["p"])
		_, _ = w.Write([]byte(`{"data":{"project":{"group":null,"boards":{"nodes":[{"id":"gid://gitlab/Board/7","name":"Development","lists":{"nodes":[{"listType":"label","label":{"title":"To do"}}]}}]}}}}`))
	}))
	defer srv.Close()
	repo := t.TempDir()
	if out, err := exec.Command("git", "-C", repo, "init", "-q").CombinedOutput(); err != nil {
		t.Fatal(string(out))
	}
	if out, err := exec.Command("git", "-C", repo, "remote", "add", "origin", "https://gitlab.example/group/proj.git").CombinedOutput(); err != nil {
		t.Fatal(string(out))
	}
	t.Setenv("CI_API_V4_URL", "")
	t.Setenv("CI_PROJECT_ID", "")
	t.Setenv("CI_PROJECT_PATH", "")
	t.Setenv("GITLAB_HOST", srv.URL)
	t.Setenv("GITLAB_TOKEN", "glpat-x")
	boards, err := (&gitlab{repo: repo}).boards()
	if err != nil || len(boards) != 1 || !slices.Equal(asked, []string{"group/proj"}) {
		t.Fatalf("boards %v, %v; asked %q", boards, err, asked)
	}
	// No remote naming a project: the forge's error, never a query with
	// an empty path.
	t.Setenv("GITLAB_HOST", "")
	if _, err := (&gitlab{repo: t.TempDir()}).boards(); err == nil || len(asked) != 1 {
		t.Fatalf("no project: %v; asked %q", err, asked)
	}
}
