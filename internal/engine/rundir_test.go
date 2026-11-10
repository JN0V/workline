package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/tester"
)

// An interrupted run's tree, outside its run folder (#1304), goes with
// that folder when the run is rotated out, its git worktree too.
func TestNewRunDirDropsAnOldRunsTree(t *testing.T) {
	repo, run, write := gitRepo(t)
	write("lib.sh", "add() { :; }\n")
	run("add", "lib.sh")
	run("commit", "-q", "-m", "a")
	t.Setenv("TMPDIR", t.TempDir())
	t.Setenv("WORKLINE_RUNS_DIR", "")
	first, err := newRunDir(repo, "developer")
	if err != nil {
		t.Fatal(err)
	}
	if err := tester.RunTree(repo, "HEAD", first, agent.WorktreeDir); err != nil {
		t.Fatal(err)
	}
	tree := agent.Worktree(first)
	if tree == "" || tree == filepath.Join(first, agent.WorktreeDir) {
		t.Fatalf("the tree %q is not out of the run", tree)
	}
	for range keepRuns { // the interrupted run never dropped its tree
		if _, err := newRunDir(repo, "developer"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(first); err == nil {
		t.Fatalf("the run %s is kept", first)
	}
	if _, err := os.Stat(tree); err == nil {
		t.Errorf("its tree %s is left", tree)
	}
	if got := run("worktree", "list", "--porcelain"); strings.Contains(got, tree) {
		t.Errorf("git still lists its tree:\n%s", got)
	}
}
