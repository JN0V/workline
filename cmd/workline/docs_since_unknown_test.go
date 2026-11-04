package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// gitIn runs git in dir, isolated from the machine's settings.
func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// TestUnknownRevision
//
// Scenario: Unknown revision
// Given a repository with no tag `v9.9`,
// When I run `workline docs --since v9.9 --ai none`,
// Then it says `v9.9` names no commit, and exits non-zero.
func TestUnknownRevision(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	t.Setenv("GIT_AUTHOR_NAME", "Tester")
	t.Setenv("GIT_AUTHOR_EMAIL", "tester@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Tester")
	t.Setenv("GIT_COMMITTER_EMAIL", "tester@example.com")

	bin := filepath.Join(t.TempDir(), "workline")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}

	repo := t.TempDir()
	gitIn(t, repo, "init", "-q")
	if err := os.MkdirAll(filepath.Join(repo, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "docs", "a.md"), []byte("# A\n\nSome words.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, repo, "add", ".")
	gitIn(t, repo, "-c", "commit.gpgsign=false", "commit", "-q", "-m", "first")

	cmd := exec.Command(bin, "docs", "--since", "v9.9", "--ai", "none")
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	text := string(out)

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() == 0 {
		t.Errorf("want a non-zero exit, got err=%v\n%s", err, text)
	}

	said := 0
	for _, l := range strings.Split(text, "\n") {
		if strings.Contains(l, "v9.9") && strings.Contains(l, "names no commit") {
			said++
		}
	}
	if said != 1 {
		t.Errorf("want one line saying v9.9 names no commit, got %d\n%s", said, text)
	}

	if regexp.MustCompile(`(?m)block\s+—\s*$`).MatchString(text) {
		t.Errorf("a block line with an empty reason:\n%s", text)
	}
	if strings.Contains(strings.ToLower(text), "workline docs` again") {
		t.Errorf("advice to run workline docs again:\n%s", text)
	}
}
