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
// off, and the squash's body is not the commits' messages: then the
// branch's commits never reach the base.
func TestGitHubSquashOnly(t *testing.T) {
	for _, c := range []struct {
		name                  string
		squash, merge, rebase bool
		body                  string
		on, only              bool
	}{
		{"squash only", true, false, false, "BLANK", true, true},
		{"squash only, the PR's body", true, false, false, "PR_BODY", true, true},
		{"squash only, the commits' messages in its body", true, false, false, "COMMIT_MESSAGES", true, false},
		{"squash or merge", true, true, false, "BLANK", true, false},
		{"squash or rebase", true, false, true, "BLANK", true, false},
		{"no squash", false, true, true, "BLANK", false, false},
	} {
		bin := t.TempDir()
		answer, _ := json.Marshal(map[string]any{"data": map[string]any{"repository": map[string]any{
			"squashMergeAllowed": c.squash, "squashMergeCommitTitle": "PR_TITLE",
			"mergeCommitAllowed": c.merge, "rebaseMergeAllowed": c.rebase, "squashMergeCommitMessage": c.body}}})
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

// GitLab squashes only when its squash option is "always", its template
// quoting no commit.
func TestGitLabSquashOnly(t *testing.T) {
	var option, template string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"squash_option": option, "squash_commit_template": template})
	}))
	defer srv.Close()
	t.Setenv("CI_API_V4_URL", srv.URL+"/api/v4")
	t.Setenv("CI_PROJECT_ID", "")
	t.Setenv("CI_PROJECT_PATH", "group/proj")
	t.Setenv("GITLAB_TOKEN", "secret")
	g := &gitlab{repo: t.TempDir()}
	for _, c := range []struct {
		option, template string
		on, only         bool
	}{
		{"always", "", true, true},
		{"always", "%{title}", true, true},
		{"always", "%{title}\n\n%{all_commits}", true, false},
		{"always", "%{first_multiline_commit}", true, false},
		{"default_on", "", true, false},
		{"default_off", "", true, false},
		{"never", "", false, false},
	} {
		option, template = c.option, c.template
		got, err := g.Squash(3)
		if err != nil {
			t.Fatalf("%s %q: %v", c.option, c.template, err)
		}
		if got.On != c.on || got.Only != c.only {
			t.Errorf("%s %q: %+v, want on %v only %v", c.option, c.template, got, c.on, c.only)
		}
	}
}
