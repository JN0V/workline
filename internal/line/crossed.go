package line

// A run too big for its role (ADR-0064, step 3): its prompt over the
// role's context budget, its token cap reached, or what it read cut
// before it blocked. The same input crosses the same, so it is never run
// again on it: it ends needs-input(product-owner: size), the
// needs-input(R2) of ADR-0063 §5. The issue waits on the product owner,
// Blocked on the board with the size said, and the product owner reads it
// at the line's next run (revisit.go), told the size: it narrows the
// issue's Scope and sends it back, or leaves it to a person. Once a state
// (Record.Sized): a second crossing from the same state, the product
// owner's own, or a project where it reads no blocked issue, waits for a
// person at once. The work is kept: no label but the state's changes, no
// branch nor pull request is touched.

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/verdict"
)

// crossed blocks st's issue for the size out says: for the product
// owner, or for a person. here: the issue is still in the state its step
// ran in; one its role moved keeps that move, the size said by its run.
func crossed(st *stateStep, out verdict.Outcome, blockNow func(rule, why string), here bool, cfg *routing.Config) {
	if !here {
		return
	}
	who := "the " + roleSaid(st.role)
	size := fmt.Sprintf("%s's run on it is too big for its context: %s", who, oneLine(out.Said))
	ok, why := revisits(cfg)
	switch {
	case st.role == backlog.Keeper:
		blockNow("size-crossed-own", size+"; no role is left to ask: a person narrows or splits it, or raises the setting named")
	case slices.Contains(st.write.Record.Sized, st.from.Name):
		blockNow("size-crossed-again", fmt.Sprintf("%s; the product owner was asked on its size from %s already: a person narrows or splits it, or raises the setting named", size, st.from.Name))
	case !ok:
		blockNow("size-crossed-again", fmt.Sprintf("%s; no product owner reads it here (%s): a person narrows or splits it, or raises the setting named", size, why))
	default:
		from := st.from.Name
		blockNow("size-crossed", size+"; the product owner reads it at once")
		// The reason kept is the size alone: the comment's next says the
		// product owner reads it, until it did (blockedBody).
		st.write.Record.Why, st.write.Blocked = size, size
		st.write.Record.Sized = append(slices.Clone(st.write.Record.Sized), from)
		st.write.Record.SizeOf = st.role
	}
}

// sizeAsked is the step waking the product owner at once on a blocked
// issue whose record holds a size question it has not read (crossed):
// told which role's run crossed, from which state; nil for none.
func sizeAsked(res *Result, rec Record, issue int) *stateStep {
	if rec.SizeOf == "" {
		return nil
	}
	said := fmt.Sprintf("the %s's run on it crossed its size", roleSaid(rec.SizeOf))
	res.Findings = append(res.Findings, verdict.Finding{Rule: "state-revisit", Level: "info", Where: fmt.Sprintf("#%d", issue),
		Message: fmt.Sprintf("blocked, read again by the %s: %s", backlog.Keeper, said)})
	next := rec
	next.SizeOf, next.Revisited = "", clock().UTC().Format(time.RFC3339)
	return &stateStep{role: backlog.Keeper, write: StateWrite{Issue: issue, Record: next},
		inputs: map[string]string{engine.RevisitInput: said, engine.RevisitFromInput: rec.From, engine.SizeInput: rec.SizeOf}}
}

// roleSaid is a role's name as a person reads it.
func roleSaid(role string) string { return strings.ReplaceAll(role, "-", " ") }
