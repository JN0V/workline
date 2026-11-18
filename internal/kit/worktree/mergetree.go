package worktree

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strings"
)

// MergeTree merges base into head as git does, in memory (`git
// merge-tree`, 2.38): the tree, its conflicts' markers left in it, and
// the files in conflict; none when it merges clean.
func MergeTree(repo, head, base string) (tree string, files []string, err error) {
	cmd := exec.Command("git", "-C", repo, "merge-tree", "--write-tree", "-z", "--name-only", "--no-messages", head, base)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	var ee *exec.ExitError
	if err != nil && (!errors.As(err, &ee) || ee.ExitCode() != 1) {
		return "", nil, fmt.Errorf("git merge-tree (2.38 or later): %v: %s", err, strings.TrimSpace(stderr.String()))
	}
	parts := strings.Split(string(out), "\x00")
	tree = strings.TrimSpace(parts[0])
	for _, p := range parts[1:] {
		if p != "" && !slices.Contains(files, p) {
			files = append(files, p)
		}
	}
	if err != nil && len(files) == 0 {
		return "", nil, errors.New("git merge-tree met a conflict it named no file of")
	}
	return tree, files, nil
}
