package forge

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A comment's reactions, each with who gave it and whether they may
// write: a bot's never asked.
func TestGitHubReactions(t *testing.T) {
	bin := t.TempDir()
	script := "#!/bin/sh\n" + `case "$*" in
*comments/77/reactions*) printf '%s\n' '{"emoji":"+1","author":"jn0v","bot":false}' '{"emoji":"-1","author":"zed","bot":false}' '{"emoji":"eyes","author":"workline[bot]","bot":true}' ;;
*collaborators/jn0v/permission*) echo admin ;;
*collaborators/zed/permission*) echo read ;;
*) echo "asked $*" >&2; exit 1 ;;
esac
`
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	got, err := (&github{repo: t.TempDir()}).Reactions(155, "77")
	if err != nil {
		t.Fatal(err)
	}
	want := "[{+1 jn0v true false} {-1 zed false false} {eyes workline[bot] false true}]"
	if fmt.Sprint(got) != want {
		t.Errorf("reactions: %v, want %s", got, want)
	}
}

// GitLab's awarded emoji read as GitHub names them; who may write is a
// Developer or above.
func TestGitLabReactions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.EscapedPath(), "/api/v4/projects/group%2Fproj")
		switch p {
		case "/issues/155/notes/77/award_emoji":
			json.NewEncoder(w).Encode([]map[string]any{
				{"name": "thumbsup", "user": map[string]any{"username": "lead"}},
				{"name": "thumbsdown", "user": map[string]any{"username": "guest"}},
			})
		case "/members/all":
			json.NewEncoder(w).Encode([]map[string]any{
				{"username": "lead", "access_level": 40, "state": "active"},
				{"username": "guest", "access_level": 10, "state": "active"},
			})
		default:
			http.Error(w, `{"message":"404 Not Found"}`, http.StatusNotFound)
		}
	}))
	defer srv.Close()
	t.Setenv("CI_API_V4_URL", srv.URL+"/api/v4")
	t.Setenv("CI_PROJECT_ID", "")
	t.Setenv("CI_PROJECT_PATH", "group/proj")
	t.Setenv("GITLAB_TOKEN", "secret")
	got, err := (&gitlab{repo: t.TempDir()}).Reactions(155, "77")
	if err != nil {
		t.Fatal(err)
	}
	want := "[{+1 lead true false} {-1 guest false false}]"
	if fmt.Sprint(got) != want {
		t.Errorf("reactions: %v, want %s", got, want)
	}
}
