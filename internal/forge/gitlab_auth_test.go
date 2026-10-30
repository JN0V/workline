package forge

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
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
	glab := "#!/bin/sh\ncase \"$3\" in token) echo oauth-tok ;; is_oauth2) echo true ;; esac\n"
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
	t.Setenv("GITLAB_TOKEN", "glpat-x")
	if _, err := (&gitlab{repo: t.TempDir()}).api("projects/:id"); err != nil {
		t.Fatal(err)
	}
	want := []string{"PRIVATE-TOKEN= Authorization=Bearer oauth-tok", "PRIVATE-TOKEN=glpat-x Authorization="}
	if !slices.Equal(sent, want) {
		t.Fatalf("sent %q, want %q", sent, want)
	}
}
