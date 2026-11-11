package engine

// A developer's request behind its base (ADR-0067, #1353): the policy
// would merge it, the forge does not. Lazy: acted on only then, never at
// each push to the base. git's merge of the base, in memory, says which:
// clean, the engine merges the base in alone, no agent, a merge commit
// pushed with no force; in conflict, said on the request, its record the
// developer's next round reads: that round settles the markers. On the
// local forge, which holds no branch behind, a clean merge is proven
// first when the base changed the request's files (#1370).

import (
	"fmt"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/developer"
	"github.com/JN0V/workline/internal/forge"
)

// upToDate acts on mr, at tip, behind baseTip: in conflict, said on it,
// for the developer's merge round; merging clean, merged in by the engine
// when clean is asked — the forge holding it, not only its merge to come —
// or, on the local forge, proven on the tests first (provenUpToDate).
// acted: the request is not to be merged now.
// due says why its merge is due, as the refused merge's comment says it.
func upToDate(f forge.Forge, o FollowOptions, local bool, mr forge.MergeRequest, baseTip, tip string, say func(rule, level, msg string), clean bool, due string) (bool, error) {
	if !mr.Here || !strings.HasPrefix(mr.Branch, developer.Branch) || isAncestor(o.Repo, baseTip, tip) {
		return false, nil
	}
	tree, files, err := developer.MergeTree(o.Repo, tip, baseTip)
	if err != nil {
		say("merge-unread", "warn", fmt.Sprintf("whether it merges with %s could not be read: %v", o.Base, err))
		return false, nil
	}
	if len(files) > 0 {
		// Said in the refused merge's one comment (#1354), its record the
		// merge round reads; once that round failed here, a person's.
		failed, err := developer.ConflictFailed(f, mr, tip)
		if err != nil {
			return true, err
		}
		c := &forge.Conflict{Base: o.Base, Files: files, Commit: baseTip, Failed: failed}
		sayRefused(f, mr, tip, due, c, say)
		next := "the developer's next round merges " + o.Base + " in and settles it"
		if failed {
			next = "the developer's round failed at this head: a person settles it"
		}
		say("merge-conflict", "warn", fmt.Sprintf("it conflicts with %s in %s: not merged; %s", o.Base, strings.Join(files, ", "), next))
		return true, nil
	}
	if local {
		return provenUpToDate(f, o, mr, tree, baseTip, tip, say, due)
	}
	if !clean {
		return false, nil
	}
	commit, err := ownMergeCommit(o.Repo, tree, tip, baseTip, o.Base, mr.Branch)
	if err != nil {
		say("merge-refused", "warn", fmt.Sprintf("behind %s, which merges in clean, but its merge could not be made: %v", o.Base, err))
		return true, nil
	}
	if _, err := git(o.Repo, nil, "push", "-q", "origin", commit+":refs/heads/"+mr.Branch); err != nil {
		say("merge-refused", "warn", fmt.Sprintf("behind %s, which merges in clean, but the push of that merge was refused: %v", o.Base, err))
		return true, nil
	}
	say("merged-from-base", "", fmt.Sprintf("behind %s and held by the forge, it merges clean: %[1]s merged into %s, a merge commit pushed, nothing rewritten; its checks run on the new head", o.Base, mr.Branch))
	sayUpdated(f, mr, o.Base, commit, say)
	return true, nil
}

// ownMergeCommit is the engine's merge of base, at baseTip, into branch,
// at tip, with tree, git's clean merge of the two: two parents, the
// branch's head first, signed when the clone signs.
func ownMergeCommit(repo, tree, tip, baseTip, base, branch string) (string, error) {
	msg := fmt.Sprintf("Merge branch '%s' into %s\n\n%s: developer\n", base, branch, OwnTrailer)
	args := []string{"commit-tree", tree, "-p", tip, "-p", baseTip, "-F", "-"}
	if signed, _ := git(repo, nil, "config", "--bool", "--get", "commit.gpgsign"); strings.TrimSpace(signed) == "true" {
		args = append(args, "-S")
	}
	return git(repo, strings.NewReader(msg), args...)
}

// redMerged is a request behind its base, merging clean, whose merge
// with it turns the tests red (#1370): said on it as its merge refused.
type redMerged struct {
	Base, BaseTip string
	Files         []string // changed by both since the branch left the base
	Proof         string
}

func (r *redMerged) Error() string {
	return fmt.Sprintf("its tests are red with %s merged in: %s", r.Base, r.Proof)
}

// redMark is kept in the refused merge's comment: the tests were red on
// the merge of base into head, not run again while neither moves.
func redMark(head, base string) string {
	return forge.Marker(fmt.Sprintf("merge-red head=%s base=%s", head, base))
}

// changedByBoth are the files both tip and baseTip changed since they
// parted, sorted.
func changedByBoth(repo, tip, baseTip string) ([]string, error) {
	from, err := git(repo, nil, "merge-base", tip, baseTip)
	if err != nil {
		return nil, err
	}
	names := func(to string) ([]string, error) {
		out, err := git(repo, nil, "diff", "--name-only", "--no-renames", from, to)
		return strings.Fields(out), err
	}
	ours, err := names(tip)
	if err != nil {
		return nil, err
	}
	theirs, err := names(baseTip)
	if err != nil {
		return nil, err
	}
	var both []string
	for _, p := range ours {
		if slices.Contains(theirs, p) {
			both = append(both, p)
		}
	}
	return both, nil
}

// provenUpToDate is a role's request of the local forge behind its base,
// merging clean (#1370). No forge's checks run on a push here, and no
// rule holds a branch behind: a merge as it is would land a tree no test
// ran on. When the base changed none of its files since they parted, it
// merges as it is. Else the engine merges the base in, a commit apart,
// and runs the test command whole on it: green, its branch moves to that
// merge, which the next reconcile's line reviews as any new head, then
// the policy merges it; red, nothing moves, said on it and its issue,
// not run again until its head or the base moves. acted: not merged now.
func provenUpToDate(f forge.Forge, o FollowOptions, mr forge.MergeRequest, tree, baseTip, tip string, say func(rule, level, msg string), due string) (bool, error) {
	both, err := changedByBoth(o.Repo, tip, baseTip)
	if err != nil {
		say("merge-unread", "warn", fmt.Sprintf("which files %s changed since its branch left it could not be read: %v; merged as it is", o.Base, err))
		return false, nil
	}
	if len(both) == 0 {
		return false, nil
	}
	files := strings.Join(both, ", ")
	if b, ok := f.(forge.Backlog); ok {
		comments, err := b.Comments(forge.Target{Kind: "merge-request", ID: mr.ID})
		if err != nil {
			return true, err
		}
		if strings.Contains(strings.Join(comments, "\n"), redMark(tip, baseTip)) {
			say("merge-red", "warn", fmt.Sprintf("behind %[1]s, which changed %[2]s since its branch left it: its tests were red with %[1]s merged in, at this head and this %[1]s; not merged, said on it", o.Base, files))
			return true, nil
		}
	}
	if at, _ := checkedOutAt(o.Repo, mr.Branch); at != "" {
		say("merge-refused", "warn", fmt.Sprintf("behind %[1]s, which changed %[2]s since its branch left it: %[1]s is to be merged into it first, but %[3]s is checked out in %[4]s; check another branch out there, and the next `workline follow` merges %[1]s in", o.Base, files, mr.Branch, at))
		return true, nil
	}
	commit, err := ownMergeCommit(o.Repo, tree, tip, baseTip, o.Base, mr.Branch)
	if err != nil {
		say("merge-refused", "warn", fmt.Sprintf("behind %[1]s, which changed %[2]s since its branch left it, but the merge of %[1]s into it could not be made: %[3]v", o.Base, files, err))
		return true, nil
	}
	p := developer.ProveMerge(o.RolesDir, o.Repo, commit)
	if p.Result != developer.Green {
		red := &redMerged{Base: o.Base, BaseTip: baseTip, Files: both, Proof: p.Said()}
		sayRefused(f, mr, tip, due, red, say)
		say("merge-red", "warn", fmt.Sprintf("behind %[1]s, which changed %[2]s since its branch left it: with %[1]s merged in, %[3]s; not merged, its branch left as it is, said on it", o.Base, files, p.Said()))
		return true, nil
	}
	if _, err := git(o.Repo, nil, "update-ref", "refs/heads/"+mr.Branch, commit, tip); err != nil {
		say("merge-refused", "warn", fmt.Sprintf("behind %[1]s, its tests green with %[1]s merged in, but %[2]s moved meanwhile: %[3]v", o.Base, mr.Branch, err))
		return true, nil
	}
	say("merged-from-base", "", fmt.Sprintf("behind %[1]s, which changed %[2]s since its branch left it: %[1]s merged into %[3]s, a merge commit, nothing rewritten; the tests green on it (%[4]s); its line runs on the new head at the next reconcile, then the policy weighs it again", o.Base, files, mr.Branch, p.Said()))
	sayUpdated(f, mr, o.Base, commit, say)
	return true, nil
}

// checkedOutAt is the worktree of repo where branch is checked out, ""
// none: git moves no branch checked out there.
func checkedOutAt(repo, branch string) (string, error) {
	out, err := git(repo, nil, "worktree", "list", "--porcelain")
	if err != nil {
		return "", err
	}
	dir := ""
	for _, l := range strings.Split(out, "\n") {
		if d, ok := strings.CutPrefix(l, "worktree "); ok {
			dir = d
		}
		if l == "branch refs/heads/"+branch {
			return dir, nil
		}
	}
	return "", nil
}

// sayUpdated edits the refused merge's comments on mr and its issues,
// when it holds one: brought up to date with base at head, its checks to
// run on it.
func sayUpdated(f forge.Forge, mr forge.MergeRequest, base, head string, say func(rule, level, msg string)) {
	t := forge.Target{Kind: "merge-request", ID: mr.ID}
	if b, ok := f.(forge.Backlog); ok {
		comments, err := b.Comments(t)
		if err != nil || !strings.Contains(strings.Join(comments, "\n"), refusedSticky(0)) {
			return
		}
	}
	body := fmt.Sprintf("**Brought up to date**: %s merged into its branch by the engine, a merge commit at %s, nothing rewritten; its checks run on it, then the merge policy weighs it again.", base, short(head))
	if err := f.Sticky(t, body, refusedSticky(0), false); err != nil {
		say("refusal-unsaid", "warn", "the merge refused before could not be said brought up to date: "+forge.Said(err))
	}
}
