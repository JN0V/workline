package engine

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/tester"
)

// The worktree a role's pre made for its agent goes when the engine drops
// it, whatever post did or did not do; a run with none is left as it is.
func TestDropWorktree(t *testing.T) {
	repo := t.TempDir()
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"-c", "user.name=x", "-c", "user.email=x@example.invalid", "commit", "-q", "--allow-empty", "-m", "start"},
	} {
		if out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	run := t.TempDir()
	dropWorktree(repo, run) // none: nothing
	tree := filepath.Join(run, agent.WorktreeDir)
	if err := tester.Worktree(repo, "HEAD", tree); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tree, "edit.sh"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dropWorktree(repo, run)
	if _, err := os.Stat(tree); !os.IsNotExist(err) {
		t.Errorf("the worktree is still there: %v", err)
	}
	out, err := exec.Command("git", "-C", repo, "worktree", "list", "--porcelain").Output()
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(out), "worktree "); n != 1 {
		t.Errorf("%d worktrees listed, want the repository's alone:\n%s", n, out)
	}
}
