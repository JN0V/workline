package line

// A step that is a review round (#1013): an issue in the developer's
// state whose own pull request, from the developer's branch, is open is
// not a new claim. The developer would pass it over, its pull request
// closing it; what is due is the round on that pull request, which the
// product owner's widened Scope, or a person's answer, sent back. The
// job that moved the issue starts that round (Result.Reviews), not the
// issue's step; the reconciler's own list of rounds is the net.

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/JN0V/workline/internal/developer"
	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/verdict"
)

// roundOn is the developer's open merge request for issue id, from its
// own branch, when the step of state s on it is the developer's; nil
// when none. Read once a run (tally).
func roundOn(b forge.Backlog, t *tally, s routing.State, id int) (*forge.MergeRequest, error) {
	if s.Role != developerRole {
		return nil, nil
	}
	if err := t.mergeRequests(b, "whether the developer's pull request is open on it"); err != nil {
		return nil, err
	}
	for _, mr := range t.mrs {
		if mr.Branch == developer.Branch+strconv.Itoa(id) {
			if t.rounds == nil {
				t.rounds = map[int]forge.MergeRequest{}
			}
			t.rounds[id] = mr
			return &mr, nil
		}
	}
	return nil, nil
}

// roundDue says whether a review round is due on mr, the developer's:
// what it could not read added to res, a warning, and not due.
func roundDue(res *Result, f forge.Forge, repo string, mr forge.MergeRequest) bool {
	where := fmt.Sprintf("merge request %d", mr.ID)
	ok, err := developer.RoundDue(f, repo, mr)
	switch {
	case errors.Is(err, developer.ErrRecordUnread):
		res.Findings = append(res.Findings, verdict.Finding{Rule: "round-unread", Level: "warn", Where: where,
			Message: "whether a review round is due is not known: " + err.Error()})
		return false
	case err != nil:
		res.Findings = append(res.Findings, verdict.Finding{Rule: "round-unread", Level: "warn", Where: where,
			Message: "whether a review round is due was not read: " + forge.Said(err)})
		return false
	}
	return ok
}

// roundSaid is a round due on mr, said with its command, for a line on
// a person's machine: a round works on its branch, checked out.
func roundSaid(res *Result, mr forge.MergeRequest) {
	res.Findings = append(res.Findings, verdict.Finding{Rule: "review-round-due", Level: "info", Where: fmt.Sprintf("merge request %d", mr.ID),
		Message: fmt.Sprintf("a review round is due: on its branch, %s, run `workline develop --merge-request %d`", mr.Branch, mr.ID)})
}

// byRound ends the issue line on issue id, whose step is the review
// round on mr: listed for the job that moved it to start (list), said
// with its command, or said not due. No agent. A round due on an issue
// at its backstop is not: the issue is blocked, the pull request kept
// open (ADR-0064, §5).
func byRound(res *Result, cfg *routing.Config, f forge.Forge, spec string, o engine.Options, event string, id int, mr forge.MergeRequest) {
	where := fmt.Sprintf("#%d's step is the review round on merge request %d", id, mr.ID)
	if !roundDue(res, f, o.Repo, mr) {
		res.Summary = fmt.Sprintf("%s: %s, not due: nothing to start", event, where)
		return
	}
	stop, w := roundStopped(res, f, cfg, o.Repo, mr)
	switch {
	case stop && w != nil && o.Inputs[ReconcileInput] == "list":
		res.Due = []int{id}
		res.Summary = fmt.Sprintf("%s: %s, at its backstop: to be blocked, not started (--input %s=list)", event, where, ReconcileInput)
	case stop && w != nil:
		keepState(res, cfg, f, spec, o, *w, "")
		res.Summary = fmt.Sprintf("%s: %s, not run: #%d blocked at its backstop", event, where, id)
	case stop:
		res.Summary = fmt.Sprintf("%s: %s, not run: #%d at its backstop, blocked already", event, where, id)
	case o.Inputs[ReconcileInput] == "list":
		res.Reviews = []Review{{ID: mr.ID, Branch: mr.Branch}}
		res.Summary = fmt.Sprintf("%s: %s, due: not started (--input %s=list)", event, where, ReconcileInput)
	default:
		roundSaid(res, mr)
		res.Summary = fmt.Sprintf("%s: %s, due", event, where)
	}
}
