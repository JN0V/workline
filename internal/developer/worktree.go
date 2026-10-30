package developer

// The worktree and its diff (ADR-0056, step 1): pre checks the commit its
// code starts from out in the run's worktree, where a cmd: agent edits in
// place; post takes the worktree's diff against that commit, untracked
// files in, whatever the agent committed, one patch a file, for the bounds
// to hold like any patch, a file deleted a deletion patch. A link or a
// file that is not text is refused here, a link out of the tree said so.
// The engine removes the worktree once its agent is asked no more: only
// what passes the bounds leaves the run.

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/tester"
)

// openWorktree checks at out in the run's worktree, for its agent.
func openWorktree(runDir, repo, at string) error {
	return tester.Worktree(repo, at, filepath.Join(runDir, agent.WorktreeDir))
}

// worktreeCode is the run's worktree's diff against base, one patch a
// file, a deletion too, and the files refused before the bounds. No
// worktree: nothing.
func worktreeCode(runDir, base string) (patches []intent.Intention, refused []Refusal, err error) {
	tree := agent.Worktree(runDir)
	if tree == "" {
		return nil, nil, nil
	}
	// Modes are not the developer's (a role writes text files): an
	// executable bit the agent set is not read.
	if _, err := treeGit(tree, "add", "-A"); err != nil {
		return nil, nil, fmt.Errorf("its agent's worktree: %w", err)
	}
	names, err := treeGit(tree, "diff", "--cached", "--no-renames", "--name-only", "-z", base)
	if err != nil {
		return nil, nil, fmt.Errorf("its agent's worktree: %w", err)
	}
	for _, name := range strings.Split(names, "\x00") {
		if name == "" {
			continue
		}
		diff, err := treeGit(tree, "diff", "--cached", "--no-renames", "--no-ext-diff", "--no-textconv", "--no-color", base, "--", ":(literal)"+name)
		if err != nil {
			return nil, nil, fmt.Errorf("its agent's worktree: %w", err)
		}
		if rule := notTextFile(tree, name, diff); rule != "" {
			refused = append(refused, Refusal{File: name, Rule: rule})
			continue
		}
		patches = append(patches, intent.Intention{Kind: "patch", Value: diff})
	}
	return patches, refused, nil
}

// notTextFile is the rule refusing a file of the worktree that is not a
// text file: symlink-out for a link leading out of the tree, not-text for
// another link, a binary, a submodule; "" for a text file.
func notTextFile(tree, name, diff string) string {
	if st, err := os.Lstat(filepath.Join(tree, filepath.FromSlash(name))); err == nil && st.Mode()&os.ModeSymlink != 0 {
		to, err := os.Readlink(filepath.Join(tree, filepath.FromSlash(name)))
		if err != nil || path.IsAbs(filepath.ToSlash(to)) || filepath.IsAbs(to) {
			return "symlink-out"
		}
		if p := path.Join(path.Dir(name), filepath.ToSlash(to)); p == ".." || strings.HasPrefix(p, "../") {
			return "symlink-out"
		}
		return "not-text"
	}
	for _, l := range strings.Split(diff, "\n") {
		if strings.HasPrefix(l, "@@") {
			break
		}
		if strings.HasPrefix(l, "Binary files ") || strings.HasPrefix(l, "GIT binary patch") || linkOrModule(l) {
			return "not-text"
		}
	}
	return ""
}

// linkOrModule says a header line of a diff gives a link's or a
// submodule's mode: its mode lines and its index line, never its names.
func linkOrModule(l string) bool {
	for _, p := range []string{"new file mode ", "deleted file mode ", "old mode ", "new mode ", "index "} {
		if rest, ok := strings.CutPrefix(l, p); ok {
			f := strings.Fields(rest)
			return len(f) > 0 && (f[len(f)-1] == "120000" || f[len(f)-1] == "160000")
		}
	}
	return false
}

// treeGit runs git in the agent's worktree, its hooks off, the files'
// modes not read.
func treeGit(tree string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", tree, "-c", "core.hooksPath=/dev/null", "-c", "core.fileMode=false"}, args...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return string(out), nil
}
