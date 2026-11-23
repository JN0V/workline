package forge

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// GitHub squashes only when merge commits and rebase merging are both
// off: then the branch's commits never reach the base.
func TestGitHubSquashOnly(t *testing.T) {
	for _, c := range []struct {
		name                  string
		squash, merge, rebase bool
		on, only              bool
	}{
		{"squash only", true, false, false, true, true},
		{"squash or merge", true, true, false, true, false},
		{"squash or rebase", true, false, true, true, false},
		{"no squash", false, true, true, false, false},
	} {
		bin := t.TempDir()
		answer, _ := json.Marshal(map[string]any{"data": map[string]any{"repository": map[string]any{
			"squashMergeAllowed": c.squash, "squashMergeCommitTitle": "PR_TITLE",
			"mergeCommitAllowed": c.merge, "rebaseMergeAllowed": c.rebase}}})
		script := "#!/bin/sh\ncat <<'EOF'\n" + string(answer) + "\nEOF\n"
		if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
		t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
		got, err := (&github{repo: t.TempDir()}).Squash(7)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got.On != c.on || got.Only != c.only || got.Suffix != " (#7)" {
			t.Errorf("%s: %+v, want on %v only %v", c.name, got, c.on, c.only)
		}
	}
}

// GitLab squashes only when its squash option is "always".
func TestGitLabSquashOnly(t *testing.T) {
	var option string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"squash_option": option})
	}))
	defer srv.Close()
	t.Setenv("CI_API_V4_URL", srv.URL+"/api/v4")
	t.Setenv("CI_PROJECT_ID", "")
	t.Setenv("CI_PROJECT_PATH", "group/proj")
	t.Setenv("GITLAB_TOKEN", "secret")
	g := &gitlab{repo: t.TempDir()}
	for _, c := range []struct {
		option   string
		on, only bool
	}{
		{"always", true, true},
		{"default_on", true, false},
		{"default_off", true, false},
		{"never", false, false},
	} {
		option = c.option
		got, err := g.Squash(3)
		if err != nil {
			t.Fatalf("%s: %v", c.option, err)
		}
		if got.On != c.on || got.Only != c.only {
			t.Errorf("%s: %+v, want on %v only %v", c.option, got, c.on, c.only)
		}
	}
}
