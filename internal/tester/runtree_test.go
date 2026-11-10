package tester

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/agent"
)

// A run folder under .git has its agent's tree out of it, its path in a
// file of the run, removed with that file (#1304); one elsewhere has it in.
func TestRunTreeOutOfGitDir(t *testing.T) {
	repo, _ := gitRepo(t, map[string]string{"lib.sh": "add() { :; }\n"})
	t.Setenv("TMPDIR", t.TempDir())
	for _, c := range []struct {
		runDir string
		linked bool
	}{
		{filepath.Join(repo, ".git", "workline", "runs", "r1"), true},
		{filepath.Join(t.TempDir(), "r2"), false},
	} {
		if err := os.MkdirAll(c.runDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := RunTree(repo, "HEAD", c.runDir, agent.WorktreeDir); err != nil {
			t.Fatal(err)
		}
		tree := agent.Worktree(c.runDir)
		if tree == "" {
			t.Fatalf("%s: no worktree", c.runDir)
		}
		if _, err := os.Stat(filepath.Join(tree, "lib.sh")); err != nil {
			t.Errorf("%s: lib.sh not checked out: %v", c.runDir, err)
		}
		if under := strings.Contains(filepath.ToSlash(tree)+"/", "/.git/"); under {
			t.Errorf("%s: its worktree %s lies under .git", c.runDir, tree)
		}
		_, err := os.Stat(agent.TreeAt(c.runDir, agent.WorktreeDir))
		if got := err == nil; got != c.linked {
			t.Errorf("%s: its path in the run %v, want %v", c.runDir, got, c.linked)
		}
		if got := tree != filepath.Join(c.runDir, agent.WorktreeDir); got != c.linked {
			t.Errorf("%s: out of the run %v, want %v", c.runDir, got, c.linked)
		}
		RemoveRunTree(repo, c.runDir, agent.WorktreeDir, tree)
		if _, err := os.Lstat(tree); err == nil {
			t.Errorf("%s: its tree left", c.runDir)
		}
		if _, err := os.Lstat(agent.TreeAt(c.runDir, agent.WorktreeDir)); err == nil {
			t.Errorf("%s: its path left in the run", c.runDir)
		}
	}
}
