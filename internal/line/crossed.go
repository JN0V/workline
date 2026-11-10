package line

// A run too big for its role (ADR-0064, step 3): its prompt over the
// role's context budget, or its token cap reached; a cut of what it read
// is said, never a crossing (§2, amended 2026-11-08). The same input
// crosses the same, so it is never run again on it: it ends
// needs-input(product-owner: size), the needs-input(R2) of ADR-0063 §5.
// The issue waits on the product owner, Blocked on the board with the
// size said, and the product owner reads it at the line's next run
// (revisit.go), told the size: it narrows the issue's Scope and sends it
// back, or leaves it to a person. Once a state (Record.Sized): a second
// crossing from the same state, the product owner's own, or a project
// where it reads no blocked issue, waits for a person at once. The work is kept: no label but the state's changes, no
// branch nor pull request is touched.

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/verdict"
)

// crossed blocks st's issue for the size out says: for the product
// owner, or for a person. here: the issue is still in the state its step
// ran in; one its role moved keeps that move, the size said by its run.
// labels makes the tier labels a person may put, when it blocks for one.
func crossed(st *stateStep, out verdict.Outcome, blockNow func(rule, why string), here bool, cfg *routing.Config, labels func()) {
	if !here {
		return
	}
	if st.role == backlog.Keeper || st.tier == backlog.TierXLarge || slices.Contains(st.write.Record.Sized, st.from.Name) {
		labels() // a person's to answer: the tiers it may put are there
	} else if ok, _ := revisits(cfg); !ok {
		labels()
	}
	who := "the " + roleSaid(st.role)
	size := fmt.Sprintf("%s's run on it is too big for its context: %s", who, oneLine(out.Said))
	ok, why := revisits(cfg)
	switch {
	case st.role == backlog.Keeper:
		blockNow("size-crossed-own", size+"; no role is left to ask: a person narrows or splits it, or raises the setting named")
	case st.tier == backlog.TierXLarge:
		// The top tier, a person's (ADR-0064, §4): nothing above it.
		blockNow("size-crossed-top", fmt.Sprintf("%s, at the top size tier (`%s`): a person narrows or splits it", size, backlog.LabelSizeXLarge))
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
	if rec.SizeOf == engine.SizePredicted {
		said = "its size, predicted past its normal at refine, is asked now that its tests are written"
	}
	res.Findings = append(res.Findings, verdict.Finding{Rule: "state-revisit", Level: "info", Where: fmt.Sprintf("#%d", issue),
		Message: fmt.Sprintf("blocked, read again by the %s: %s", backlog.Keeper, said)})
	next := rec
	next.SizeOf, next.Revisited = "", clock().UTC().Format(time.RFC3339)
	// rec: the record read, so a machine error of its run counts against
	// the budget (runStateStep), the question kept for the run again.
	return &stateStep{role: backlog.Keeper, write: StateWrite{Issue: issue, Record: next}, rec: rec,
		inputs: map[string]string{engine.RevisitInput: said, engine.RevisitFromInput: rec.From, engine.SizeInput: rec.SizeOf}}
}

// sizeRetry is what the machine errors of the product owner's runs on a
// size question make of it (ADR-0063, I3): asked again, with what failed
// (inputs); a wait (nil, nil); or, its tries spent, the question dropped
// and the issue left blocked for a person (the record to write).
func sizeRetry(res *Result, rec Record, issue int) (inputs map[string]string, spent *StateWrite, wait bool) {
	errs := errorsOf(rec)
	if errs == nil {
		return nil, nil, false
	}
	where := fmt.Sprintf("#%d", issue)
	again, until, why := errs.retry(clock().UTC())
	switch {
	case why != "":
		said := fmt.Sprintf("the %s's runs on its size failed: %s; a person narrows or splits it, or raises the setting named", roleSaid(backlog.Keeper), why)
		res.Findings = append(res.Findings, verdict.Finding{Rule: "machine-error-spent", Level: "warn", Where: where, Message: said})
		now := rec
		now.SizeOf, now.Errors, now.Why = "", nil, rec.Why+"; "+said
		return nil, &StateWrite{Issue: issue, Record: now}, false
	case !again:
		res.Findings = append(res.Findings, verdict.Finding{Rule: "state-waits", Level: "info", Where: where,
			Message: fmt.Sprintf("the %s's last run on it failed (%s): run again from %s", backlog.Keeper, errs.Kind, until.UTC().Format("2006-01-02 15:04 UTC"))})
		return nil, nil, true
	}
	res.Findings = append(res.Findings, verdict.Finding{Rule: "state-retry", Level: "info", Where: where,
		Message: fmt.Sprintf("the %s's last run on it failed (%s): run again, try %d of %d", backlog.Keeper, errs.Kind, errs.By[errs.Kind], ErrorBudget)})
	return map[string]string{engine.FailedInput: errs.Said}, nil, false
}

// roleSaid is a role's name as a person reads it.
func roleSaid(role string) string { return strings.ReplaceAll(role, "-", " ") }

// unanswered is out, or a machine error when findings say the agent
// failed or its answer did not read: a run that passes on them, as the
// product owner does, decided nothing.
func unanswered(out verdict.Outcome, findings []verdict.Finding) verdict.Outcome {
	for _, f := range findings {
		switch f.Rule {
		case "agent-unavailable", "engine-error":
			return verdict.Outcome{Kind: verdict.MachineError, Of: "run", Said: f.Message}
		case "agent-invalid-output":
			return verdict.Outcome{Kind: verdict.MachineError, Of: "answer", Said: f.Message}
		}
	}
	return out
}

// tierLabels makes both size tiers' labels in the forge's list, for a
// person to put (ADR-0064, §3): none when the line judges with no write
// token; one not made is said.
func tierLabels(res *Result, f forge.Forge, noApply bool) {
	b, ok := f.(forge.Backlog)
	if !ok || noApply {
		return
	}
	for _, tier := range []string{backlog.TierLarge, backlog.TierXLarge} {
		if err := backlog.EnsureTierLabel(b, tier); err != nil {
			res.Findings = append(res.Findings, verdict.Finding{Rule: "label-not-made", Level: "warn",
				Message: fmt.Sprintf("the label %s could not be made in the forge's list, for a person to put: %v", backlog.TierLabel(tier), err)})
		}
	}
}

// splitAfterTests blocks a bug its developer would take for the product
// owner's question on its size, when that one predicted it past its
// normal at refine and readied it for its tests first (#1212): its tests
// written now, on its Draft, it is split, or granted, before any
// developer run. Asked once from its state (Sized); nil when not due.
func splitAfterTests(res *Result, cfg *routing.Config, s routing.State, is forge.Issue, now Record, comments []string) *StateWrite {
	if s.Role != developerRole || slices.Contains(now.Sized, s.Name) || backlog.SizeTier(is.Labels) != "" {
		return nil
	}
	if ok, _ := revisits(cfg); !ok {
		return nil // no product owner reads it here: the developer takes it, a crossing a person's
	}
	if kind, _ := backlog.KindOf(is.Labels); kind != backlog.KindBug {
		return nil
	}
	st, found, err := backlog.ReadState(comments, backlog.Keeper)
	if !found || err != nil || !st.SplitTests || len(st.Split) > 0 {
		return nil
	}
	why := "the product owner predicted it past its normal at refine, and its tests went first: they are written now, on its Draft, so it is split or granted before the developer takes it"
	res.Findings = append(res.Findings, verdict.Finding{Rule: "size-predicted", Level: "warn", Where: fmt.Sprintf("#%d", is.ID),
		Message: why + "; the product owner reads it at once: moved to " + BlockedState})
	r := blocking(now, s.Name, why, comments)
	r.Sized = append(slices.Clone(r.Sized), s.Name)
	r.SizeOf = engine.SizePredicted
	return &StateWrite{Issue: is.ID, Record: r, Blocked: why}
}
