package engine

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/verdict"
)

// queueGroup gives a merge queue's group commit the yes its base requires
// with the merge policy (ADR-0050, section 4): GitHub tests the group's
// commit against the base's required checks, workline/accepted from the
// App among them, which no other run sets there; the queue would wait
// until its timeout. The yes is the one `follow` keeps on the head of the
// pull request the group ends with, a person's (ADR-0042): the queue's run
// never gives one. The pull requests ahead of it have groups of their own.
func queueGroup(f forge.Forge, o FollowOptions, res *Result) *Result {
	fail := func(err error) *Result {
		res.Status = verdict.Block
		if errors.Is(err, forge.ErrUnreachable) && !forge.Refused(err) {
			res.Status = verdict.BlockedExternal
		}
		res.Findings = append(res.Findings, verdict.Finding{Rule: "follow-failed", Message: err.Error()})
		return res
	}
	n, ok := groupOf(o.Group)
	switch {
	case !ok:
		return fail(fmt.Errorf("the merge queue's group %q names no pull request (gh-readonly-queue/<base>/pr-<n>-<commit>)", o.Group))
	case o.Merge == "":
		res.Summary = "a merge queue's group: the project sets no merge policy (`merge:`), no yes to set on its commit"
		return res
	}
	m, merges := f.(forge.Merger)
	g, keeps := f.(forge.QueueGuard)
	if !merges || !keeps || forge.KeepsBranches(f) {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "merge-unsupported", Level: "warn",
			Message: "this forge keeps no yes on a merge queue's group: nothing set on its commit"})
		res.Summary = "a merge queue's group: the forge keeps no yes there"
		return res
	}
	// The head queued is the pull request's own: GitHub takes a pull
	// request pushed to out of the queue, its group with it. The commit
	// the group's branch names is the one it was built on, never the head.
	st, err := m.MergeState(forge.MergeRequest{ID: n})
	if err != nil {
		return fail(fmt.Errorf("#%d, queued: its head could not be read: %w", n, err))
	}
	if st.Head == "" {
		return fail(fmt.Errorf("#%d, queued: the forge names no head", n))
	}
	yes, why, err := g.AcceptedOn(st.Head)
	if err != nil {
		return fail(fmt.Errorf("#%d's head %s, queued: its yes could not be read: %w", n, short(st.Head), err))
	}
	said, rule := fmt.Sprintf("#%d's head %s, queued, holds the yes", n, short(st.Head)), "queue-accepted"
	if !yes {
		said, rule = fmt.Sprintf("#%d's head %s, queued, holds no yes", n, short(st.Head)), "queue-not-accepted"
		if why != "" {
			said += " (" + why + ")"
		}
	}
	if err := m.Accept(o.GroupHead, yes, said); err != nil {
		return fail(fmt.Errorf("the merge queue's group %s: %w", short(o.GroupHead), err))
	}
	res.Findings = append(res.Findings, verdict.Finding{Rule: rule, Where: fmt.Sprintf("#%d", n),
		Message: fmt.Sprintf("%s: set on the group's commit, %s", said, short(o.GroupHead))})
	res.Summary = "a merge queue's group: " + said + ", set on its commit"
	return res
}

// groupOf reads a merge queue group's branch as GitHub names it,
// gh-readonly-queue/<base>/pr-<n>-<commit>: the pull request the group
// ends with. The commit is the one the group was built on, the base's tip
// or the group ahead of it.
func groupOf(ref string) (int, bool) {
	at := strings.LastIndex(ref, "/pr-")
	if !strings.HasPrefix(ref, "gh-readonly-queue/") || at < 0 {
		return 0, false
	}
	num, built, ok := strings.Cut(ref[at+len("/pr-"):], "-")
	n, err := strconv.Atoi(num)
	if !ok || err != nil || n < 1 || built == "" {
		return 0, false
	}
	return n, true
}
