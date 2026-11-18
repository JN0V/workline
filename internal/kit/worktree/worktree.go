// Package worktree checks a revision out apart, in a git worktree of the
// repository, its hooks off: where the tester and the developer lay and
// try a change, and where a run's agent edits (ADR-0056).
package worktree

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/agent"
)

// Worktree checks rev out in tree, a git worktree of repo apart, detached,
// its hooks off: where a change is laid and tried, and where the
// developer's agent edits (ADR-0056). Removed when it cannot be made.
func Worktree(repo, rev, tree string) error {
	if _, err := git(repo, "-c", "core.hooksPath=/dev/null", "worktree", "add", "--detach", "--quiet", tree, rev); err != nil {
		RemoveWorktree(repo, tree)
		return fmt.Errorf("%s could not be checked out: %v", short(rev), err)
	}
	return nil
}

// RunTree checks rev out for a run's agent, at runDir/name (an
// agent.WorktreeDir or agent.ReadTreeDir). A run folder under a .git
// folder, the default (no WORKLINE_RUNS_DIR), has its tree in a folder of
// its own outside it, its path in the run's agent.TreeAt file: Claude
// Code edits no file under .git, taking it for a sensitive one (#1304).
func RunTree(repo, rev, runDir, name string) error {
	if !underGitDir(runDir) {
		return Worktree(repo, rev, filepath.Join(runDir, name))
	}
	tree, err := os.MkdirTemp("", "workline-"+name+"-")
	if err != nil {
		return err
	}
	if err := Worktree(repo, rev, tree); err != nil {
		return err
	}
	if err := os.WriteFile(agent.TreeAt(runDir, name), []byte(tree+"\n"), 0o644); err != nil {
		RemoveWorktree(repo, tree)
		return err
	}
	return nil
}

// RemoveRunTree removes a tree RunTree made, and the file naming it.
func RemoveRunTree(repo, runDir, name, tree string) {
	RemoveWorktree(repo, tree)
	_ = os.Remove(agent.TreeAt(runDir, name))
}

// underGitDir says dir lies under a folder named .git.
func underGitDir(dir string) bool {
	abs, err := filepath.Abs(dir)
	if err != nil {
		abs = dir
	}
	return slices.Contains(strings.Split(filepath.ToSlash(abs), "/"), ".git")
}

// RemoveWorktree removes a worktree Worktree made, whatever it holds.
func RemoveWorktree(repo, tree string) {
	if _, err := git(repo, "worktree", "remove", "--force", tree); err != nil {
		_ = os.RemoveAll(tree)
		_, _ = git(repo, "worktree", "prune")
	}
}

func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return string(out), nil
}

func short(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
