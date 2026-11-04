package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Scenario "Staged file, no commit":
// Given a new repository with `README.md` staged and no commit,
// When I run `workline init`,
// Then it exits 0 and prints no `engine-error`.
func TestStagedFileNoCommit(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	t.Setenv("GIT_AUTHOR_NAME", "Test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")

	dir := t.TempDir()
	for _, args := range [][]string{{"init", "-q"}, {"add", "README.md"}} {
		if args[0] == "add" {
			if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# New\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}

	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(old); err != nil {
			t.Error(err)
		}
	})

	var code int
	var stdout string
	stderr, _ := captureStderr(t, func() int {
		stdout, code = captureStdout(t, func() int { return dispatch([]string{"init"}) })
		return code
	})
	all := stdout + stderr
	if code != 0 {
		t.Errorf("workline init: exit %d, want 0; output %q", code, all)
	}
	if strings.Contains(all, "engine-error") {
		t.Errorf("output holds engine-error: %q", all)
	}
	if !strings.Contains(all, "pass — docs are up to date") {
		t.Errorf("output lacks %q: %q", "pass — docs are up to date", all)
	}
}
