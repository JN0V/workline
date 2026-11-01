package line

// Room in the next state (ADR-0059, decision 9): a step that moves an
// issue into a state at its WIP limit does not start. It waits, said
// once on the issue in the line's comment, and the next event or
// reconcile starts it when room appears: the line pulls, it never pushes
// work onto a full state.

import (
	"fmt"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/verdict"
)

// waits says why the step of state s waits: its next state is at its
// limit; "" when it has room or no limit. open is the open issues, listed
// here at first need and kept for the next call.
func waits(cfg *routing.Config, b forge.Backlog, s routing.State, open *[]forge.Issue) (string, error) {
	limit := cfg.Limit(s.Next)
	if s.Next == "" || limit == 0 {
		return "", nil
	}
	if *open == nil {
		issues, err := b.Issues()
		if err != nil {
			return "", err
		}
		*open = issues
	}
	labels := make([][]string, 0, len(*open))
	for _, is := range *open {
		labels = append(labels, is.Labels)
	}
	n := cfg.Holds(s.Next, labels)
	if n < limit {
		return "", nil
	}
	return fmt.Sprintf("%s holds %d open %s, its limit %d (routing.wip): the %s starts when it has room",
		s.Next, n, plural(n, "issue", "issues"), limit, s.Role), nil
}

// waiting is what stateOf returns for a step that waits: the line's
// comment to write, its record unchanged but for the state it waits on;
// nil when the comment says it already, once is enough.
func waiting(res *Result, s routing.State, why string, rec Record, where string, id int) *StateWrite {
	res.Findings = append(res.Findings, verdict.Finding{Rule: "state-waits", Level: "info", Where: where,
		Message: fmt.Sprintf("its state, %s, waits to wake the %s: %s", s.Name, s.Role, why)})
	if rec.Waits == s.Next {
		return nil
	}
	rec.Waits = s.Next
	return &StateWrite{Issue: id, Record: rec, Waits: why}
}

// anticipate counts an issue whose step started in its next state, so
// the steps after it in the same run see it there.
func anticipate(cfg *routing.Config, is *forge.Issue, next string) {
	if s, ok := cfg.State(next); ok && next != "" {
		is.Labels = []string{s.Label}
	}
}
