package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Scenario "Staged file, no commit" (neighbour): a history that exists but
// cannot be read is a block with git's own text, never an empty reason.
func TestInitUnreadableHistoryBlocksWithReason(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	run("init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# New\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "README.md")
	// the branch names a commit that is not in the object store
	ref := filepath.Join(dir, ".git", "refs", "heads", "main")
	if err := os.WriteFile(ref, []byte(strings.Repeat("a", 40)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })

	var code int
	var stdout string
	stderr, _ := captureStderr(t, func() int {
		stdout, code = captureStdout(t, func() int { return dispatch([]string{"init"}) })
		return code
	})
	all := stdout + stderr
	if code == 0 {
		t.Errorf("exit 0, want a block; output %q", all)
	}
	if strings.Contains(all, "pre exited with code 99") {
		t.Errorf("engine-error with no reason: %q", all)
	}
	i := strings.Index(all, "block —")
	if i < 0 {
		t.Fatalf("no block line: %q", all)
	}
	if rest := strings.TrimSpace(all[i+len("block —"):]); !strings.Contains(rest, "bad object") {
		t.Errorf("block reason empty or without git's text: %q", all)
	}
}
