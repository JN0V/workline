package engine

// A line pull request whose head GitHub never ran its checks on (#1719):
// the `opened` event was lost, and no push follows. The reconciler brings
// the branch up to date with its base as the App, a merge commit pushed
// with no force, which GitHub reads as a push and runs the checks on.

import (
	"fmt"
	"strings"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/kit/devmark"
	"github.com/JN0V/workline/internal/kit/worktree"
)

// BringChecksBack pushes mr's branch merged with its base, so that its
// head gets a new run of the checks. pushed: it did; said, what was done
// or why nothing was, for the finding; "" when there is nothing to say.
// A branch already holding its base, or in conflict with it, is left as
// it is: a push of it would change nothing, or ask a person's hand.
func BringChecksBack(repo string, mr forge.MergeRequest) (pushed bool, said string, err error) {
	base := mr.Base
	if !mr.Here || base == "" || !strings.HasPrefix(mr.Branch, devmark.Branch) {
		return false, "", nil
	}
	baseTip, tip, err := tips(repo, base, mr.Branch, false)
	if err != nil {
		return false, "", err
	}
	if tip == "" {
		return false, "", nil
	}
	if isAncestor(repo, baseTip, tip) {
		return false, fmt.Sprintf("its head has no check of the project's, but its branch holds %s already: no push brings them back; re-run its checks by hand", base), nil
	}
	// Its own merge already pushed, and still no check: the project's CI
	// does not run on it (no workflow, a filter, a token's push). A second
	// merge would be noise, pushed again and again.
	if msg, err := git(repo, nil, "log", "-1", "--format=%B", tip); err == nil &&
		strings.Contains(msg, OwnTrailer) && strings.HasPrefix(msg, fmt.Sprintf("Merge branch '%s' into", base)) {
		return false, "its head has no check of the project's, though the line merged " + base + " into it already: no second push; the project's CI may not run on it", nil
	}
	tree, files, err := worktree.MergeTree(repo, tip, baseTip)
	if err != nil {
		return false, "", fmt.Errorf("whether it merges with %s could not be read: %v", base, err)
	}
	if len(files) > 0 {
		return false, fmt.Sprintf("its head has no check of the project's, and it conflicts with %s in %s: the developer's merge round settles it", base, strings.Join(files, ", ")), nil
	}
	commit, err := ownMergeCommit(repo, tree, tip, baseTip, base, mr.Branch)
	if err != nil {
		return false, "", fmt.Errorf("the merge of %s into it could not be made: %v", base, err)
	}
	if _, err := git(repo, nil, "push", "-q", "origin", commit+":refs/heads/"+mr.Branch); err != nil {
		return false, "", fmt.Errorf("the push of that merge was refused: %v", err)
	}
	return true, fmt.Sprintf("its head has no check of the project's: %s merged into %s, a merge commit pushed, nothing rewritten; its checks run on the new head", base, mr.Branch), nil
}
