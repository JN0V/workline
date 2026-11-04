package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Inside a repository
//
// Given a git repository,
// When I run `workline hooks install --repo` in it,
// Then it writes `.githooks/commit-msg` and exits 0.
func TestHooksInstallRepoInsideARepository(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	dir := t.TempDir()
	if out, err := exec.Command("git", "-C", dir, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	t.Chdir(dir)

	if code := dispatch([]string{"hooks", "install", "--repo"}); code != 0 {
		t.Fatalf("hooks install --repo: exit %d, want 0", code)
	}
	if _, err := os.Stat(filepath.Join(dir, ".githooks", "commit-msg")); err != nil {
		t.Errorf(".githooks/commit-msg not written: %v", err)
	}
}
