package line

// A bug not reproduced (ADR-0060, decision 4): the tester's tests of a
// bug passed on the base, so it put workline:needs-repro on it and asked
// its reporter how to reproduce it. The line reads it with no agent:
//
//   - its body edited since the tester's run: the tester reads it again;
//   - a person's answer after the mark: back to refine, the mark off, for
//     the product owner to fold the answer into its steps;
//   - no answer past the product owner's delay (acts.ask.wait), counted
//     from when the forge says the mark was put on: Blocked, its reason
//     "not reproduced", the mark off (ADR-0059). Never another kind;
//   - else it waits, said.

import (
	"fmt"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/board"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/verdict"
)

// repro reads a bug in the tester's state marked workline:needs-repro:
// a write moving it — to Blocked, or back to refine —, or nothing while
// it waits; held false when its body changed since the tester's run, the
// step then due as any.
func repro(res *Result, b forge.Backlog, t *tally, s routing.State, is forge.Issue, rec Record, had bool, labeled string, notes []forge.Note) (*StateWrite, bool, error) {
	where := fmt.Sprintf("#%d", is.ID)
	if had && rec.State == s.Name && rec.Body != bodyHash(is.Body) {
		return nil, false, nil // its steps edited: the tester writes its tests again
	}
	now := Record{State: s.Name, Body: bodyHash(is.Body), Labeled: labeled, Hops: rec.Hops, Reached: rec.Reached, Spent: rec.Spent, Counted: rec.Counted}
	trail, err := b.Trail(is.ID, backlog.LabelNeedsRepro)
	if err != nil {
		return nil, false, err
	}
	since, known := labeledAt(trail.Labeled)
	if !known {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "repro-undated", Level: "info", Where: where,
			Message: fmt.Sprintf("its tests pass on the base, but the forge does not say when it got %s: it waits for its reporter", backlog.LabelNeedsRepro)})
		return nil, true, nil
	}
	for _, n := range backlog.Counted(notes, is) {
		if at, ok := labeledAt(n.Created); ok && at.After(since) {
			why := fmt.Sprintf("@%s answered after the tester could not reproduce it: the product owner folds the answer into its steps, then readies it for its tests again", n.Author)
			res.Findings = append(res.Findings, verdict.Finding{Rule: "repro-answered", Level: "info", Where: where, Message: why + ": back to refine"})
			return &StateWrite{Issue: is.ID, Record: now, Repro: why, Off: []string{backlog.LabelNeedsRepro}}, true, nil
		}
	}
	w := t.askWait()
	if w.Passed(since, clock()) {
		why := fmt.Sprintf("%s: its tests passed on the base, and its reporter gave no way to reproduce it in %s (acts.ask.wait)", backlog.NotReproduced, w.Say())
		res.Findings = append(res.Findings, verdict.Finding{Rule: "not-reproduced", Level: "warn", Where: where, Message: why + ": moved to " + BlockedState})
		return &StateWrite{Issue: is.ID, Record: blocking(now, s.Name, why, forge.Bodies(notes)), Blocked: why, Off: []string{backlog.LabelNeedsRepro}}, true, nil
	}
	until := "until it is answered (acts.ask.wait never)"
	if !w.Never {
		until = fmt.Sprintf("until %s (acts.ask.wait %s), then it is blocked as %s", since.Add(w.D).UTC().Format("2006-01-02 15:04 UTC"), w.Say(), backlog.NotReproduced)
	}
	res.Findings = append(res.Findings, verdict.Finding{Rule: "repro-waits", Level: "info", Where: where,
		Message: "its tests pass on the base: it waits for its reporter " + until})
	return nil, true, nil
}

// reproBody writes a bug's move back to refine, its reporter having
// answered — its state's labels and the mark off, its card to the
// board's refine column — and is the line's comment saying why.
func reproBody(f forge.Forge, cfg *routing.Config, repo string, t forge.Target, w StateWrite) (string, error) {
	add, remove, err := cfg.Move(refineState)
	if err != nil {
		return "", err
	}
	card := board.Along(f, repo, refineState)
	add, remove = card.Fold(add, append(remove, w.Off...))
	if err := f.Label(t, add, remove); err != nil {
		return "", err
	}
	card.Write(f, t.ID)
	return backlog.Said("workline's line", "moves this bug back to refine", w.Repro,
		"the product owner reads the answer; its tests are written again once it is ready") + "\n\n" + recordMarker(w.Record), nil
}
