package engine

// A developer's request behind its base (ADR-0067, #1353): the policy
// would merge it, the forge does not. Lazy: acted on only then, never at
// each push to the base. git's merge of the base, in memory, says which:
// clean, the engine merges the base in alone, no agent, a merge commit
// pushed with no force; in conflict, said on the request, its record the
// developer's next round reads: that round settles the markers.

import (
	"fmt"
	"strings"

	"github.com/JN0V/workline/internal/developer"
	"github.com/JN0V/workline/internal/forge"
)

// upToDate acts on mr, at tip, behind baseTip: in conflict, said on it,
// for the developer's merge round; merging clean, merged in by the engine
// when clean is asked — the forge holding it, not only its merge to come.
// acted: the request is not to be merged now.
func upToDate(f forge.Forge, o FollowOptions, local bool, mr forge.MergeRequest, baseTip, tip string, say func(rule, level, msg string), clean bool) (bool, error) {
	if !mr.Here || !strings.HasPrefix(mr.Branch, developer.Branch) || isAncestor(o.Repo, baseTip, tip) {
		return false, nil
	}
	tree, files, err := developer.MergeTree(o.Repo, tip, baseTip)
	if err != nil {
		say("merge-unread", "warn", fmt.Sprintf("whether it merges with %s could not be read: %v", o.Base, err))
		return false, nil
	}
	if len(files) > 0 {
		t := forge.Target{Kind: "merge-request", ID: mr.ID}
		marker := forge.Marker("sticky=" + developer.ConflictKey)
		body := developer.ConflictSaid(o.Base, mr.Branch, tip, baseTip, files)
		kept, err := forge.ReadKept(f, t)
		if err != nil {
			return true, err
		}
		if now, ok := kept.Last(marker); !ok || !strings.Contains(now, strings.TrimSpace(body)) {
			if err := forge.KeepOwn(f, t, body, marker, true); err != nil {
				return true, err
			}
		}
		say("merge-conflict", "warn", fmt.Sprintf("it conflicts with %s in %s: not merged; the developer's next round merges %[1]s in and settles it", o.Base, strings.Join(files, ", ")))
		return true, nil
	}
	if !clean || local {
		return false, nil
	}
	msg := fmt.Sprintf("Merge branch '%s' into %s\n\n%s: developer\n", o.Base, mr.Branch, OwnTrailer)
	args := []string{"commit-tree", tree, "-p", tip, "-p", baseTip, "-F", "-"}
	if signed, _ := git(o.Repo, nil, "config", "--bool", "--get", "commit.gpgsign"); strings.TrimSpace(signed) == "true" {
		args = append(args, "-S")
	}
	commit, err := git(o.Repo, strings.NewReader(msg), args...)
	if err != nil {
		say("merge-refused", "warn", fmt.Sprintf("behind %s, which merges in clean, but its merge could not be made: %v", o.Base, err))
		return true, nil
	}
	if _, err := git(o.Repo, nil, "push", "-q", "origin", commit+":refs/heads/"+mr.Branch); err != nil {
		say("merge-refused", "warn", fmt.Sprintf("behind %s, which merges in clean, but the push of that merge was refused: %v", o.Base, err))
		return true, nil
	}
	say("merged-from-base", "", fmt.Sprintf("behind %s and held by the forge, it merges clean: %[1]s merged into %s, a merge commit pushed, nothing rewritten; its checks run on the new head", o.Base, mr.Branch))
	return true, nil
}
