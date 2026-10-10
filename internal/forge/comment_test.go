package forge

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A comment's address, which the product owner's comment links to: the
// issue's page, then the forge's anchor of the comment — GitHub's
// `#issuecomment-`, GitLab's `#note_`; none on a forge without one, nor
// for a comment the forge gave no id.
func TestCommentLink(t *testing.T) {
	bin := t.TempDir()
	script := "#!/bin/sh\necho https://github.com/a/b\n" // what `--jq .html_url` prints
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.TrimPrefix(r.URL.EscapedPath(), "/api/v4/projects/group%2Fproj") != "" {
			http.Error(w, `{"message":"404 Not Found"}`, http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"web_url": "https://gitlab.example/group/proj"})
	}))
	defer srv.Close()
	t.Setenv("CI_API_V4_URL", srv.URL+"/api/v4")
	t.Setenv("CI_PROJECT_ID", "")
	t.Setenv("CI_PROJECT_PATH", "group/proj")
	t.Setenv("GITLAB_TOKEN", "secret")

	for _, c := range []struct {
		name string
		f    any
		id   string
		want string
	}{
		{"github", &github{repo: t.TempDir()}, "2077", "https://github.com/a/b/issues/92#issuecomment-2077"},
		{"gitlab", &gitlab{repo: t.TempDir()}, "31", "https://gitlab.example/group/proj/-/issues/92#note_31"},
		{"the simulated forge", &Fake{}, "2", "https://forge.example/issues/92#comment-2"},
		{"no id", &Fake{}, "", ""},
		{"a forge without anchors", &Local{}, "2", ""},
	} {
		if got := CommentLink(c.f, 92, c.id); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}
