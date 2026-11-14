package steer

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// commit writes files in a new repository, commits them, and gives the
// repository and its HEAD.
func commit(t *testing.T, files map[string]string) (string, string) {
	t.Helper()
	repo := t.TempDir()
	run := func(args ...string) string {
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1",
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.invalid", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.invalid")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return string(out)
	}
	run("init", "-q", "-b", "main")
	for p, s := range files {
		if err := os.MkdirAll(filepath.Join(repo, filepath.Dir(p)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repo, p), []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run("add", ".")
	run("commit", "-q", "-m", "files")
	return repo, "HEAD"
}

// CODEOWNERS is the one list (ADR-0068 §4): a path it gives an owner is
// held, and a CODEOWNERS file; a page AGENTS.md links to, a file named a
// prompt, another role's prompt are not, nor an agent file it leaves out.
func TestFilesReadsCodeowners(t *testing.T) {
	repo, tip := commit(t, map[string]string{
		".github/CODEOWNERS": "# oversight\n/AGENTS.md @owner\n/docs/PRINCIPLES.md @owner\n/.github/workflows/ @owner\n" +
			"/roles/reviewer/instruction.md @owner\n/roles/reviewer/lenses/ @owner\n/roles/auditor/kinds @owner\n",
		"AGENTS.md":          "Read [the guide](docs/guide.md).\n",
		"docs/guide.md":      "A guide.\n",
		"docs/PRINCIPLES.md": "1. People.\n",
	})
	held, err := Files(repo, tip)
	if err != nil {
		t.Fatal(err)
	}
	for p, want := range map[string]string{
		"AGENTS.md": ListedWhy, "docs/PRINCIPLES.md": ListedWhy, ".github/workflows/ci.yml": ListedWhy,
		"roles/reviewer/instruction.md": ListedWhy, "roles/reviewer/lenses/tests.md": ListedWhy,
		".github/CODEOWNERS": ListedWhy, "CODEOWNERS": ListedWhy, // the list itself, wherever the forges read it
		"docs/guide.md": "", "roles/tester/instruction.md": "", "docs/review-prompt.md": "",
		"roles/reviewer/README.md": "", "sub/CLAUDE.md": "",
		"roles/auditor/kinds/split.md": ListedWhy, // a folder named with no trailing `/`: all of it
		"roles/auditor/kindsx.md":      "",
	} {
		if got := held(p); got != want {
			t.Errorf("%s: %q, want %q", p, got, want)
		}
	}
}

// With no CODEOWNERS, what steers agents: an agent file and the pages it
// links; a file named a prompt no longer.
func TestFilesWithNoCodeowners(t *testing.T) {
	repo, tip := commit(t, map[string]string{
		"AGENTS.md":             "Read [the guide](docs/guide.md).\n",
		"docs/guide.md":         "A guide.\n",
		"docs/review-prompt.md": "A prompt.\n",
	})
	held, err := Files(repo, tip)
	if err != nil {
		t.Fatal(err)
	}
	for p, want := range map[string]string{"AGENTS.md": AgentsWhy, "docs/guide.md": AgentsWhy, "sub/claude.md": AgentsWhy,
		"CODEOWNERS": ListedWhy, ".github/CODEOWNERS": ListedWhy, // a list added, or one deleted where the rules are read at the head
		"docs/review-prompt.md": "", "roles/tester/instruction.md": ""} {
		if got := held(p); got != want {
			t.Errorf("%s: %q, want %q", p, got, want)
		}
	}
}

// CODEOWNERS read as the forges read it: the last rule that matches
// decides in its section; a rule naming no owner unlists; a section's
// default owners stand for a rule naming none; an optional section owns,
// but requires no approval; an exclusion is never taken to unlist.
func TestCodeownersOwns(t *testing.T) {
	co := ParseCodeowners("CODEOWNERS", []byte("* @all\n/vendor/\nREADME.md @docs\n[CI] @ops\n/.gitlab-ci.yml\n^[Docs] @docs\n/notes/\n!/ci/\n/ci/ @ops\n"))
	if co.Unread != "excludes paths (`!`)" {
		t.Errorf("unread %q", co.Unread)
	}
	for f, want := range map[string][2]bool{
		"src/a.go":        {true, true},
		"vendor/x/y.go":   {false, false},
		"sub/README.md":   {true, true},
		".gitlab-ci.yml":  {true, true},
		"notes/a.md":      {true, true}, // the first section's `*` requires it
		"ci/a.yml":        {true, true},
		"vendor/notes.md": {false, false},
	} {
		if owned, required := co.Owns(f); owned != want[0] || required != want[1] {
			t.Errorf("%s: owned %v required %v, want %v", f, owned, required, want)
		}
	}
	opt := ParseCodeowners("CODEOWNERS", []byte("^[Docs] @docs\n/notes/\n"))
	if owned, required := opt.Owns("notes/a.md"); !owned || required {
		t.Errorf("optional section: owned %v required %v, want owned, not required", owned, required)
	}
}
