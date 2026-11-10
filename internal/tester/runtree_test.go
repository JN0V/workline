package tester

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/agent"
)

// A run folder under .git has its agent's tree out of it, linked from
// the run, and removed with its link (#1304); one elsewhere has it in.
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
		st, err := os.Lstat(filepath.Join(c.runDir, agent.WorktreeDir))
		if err != nil {
			t.Fatal(err)
		}
		if got := st.Mode()&os.ModeSymlink != 0; got != c.linked {
			t.Errorf("%s: linked out %v, want %v", c.runDir, got, c.linked)
		}
		RemoveRunTree(repo, c.runDir, agent.WorktreeDir, tree)
		if _, err := os.Lstat(tree); err == nil {
			t.Errorf("%s: its tree left", c.runDir)
		}
		if _, err := os.Lstat(filepath.Join(c.runDir, agent.WorktreeDir)); err == nil {
			t.Errorf("%s: its link left", c.runDir)
		}
	}
}
