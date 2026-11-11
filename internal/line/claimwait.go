package line

// A claim with no pull request is never silent (#1084): Doing wakes no
// role, so the line wakes the developer on it past claim-hours, or once
// its body changed since a claim that wrote no code; the developer's run
// lets it go. Until then, the line says the day it is let go. A claim
// whose agent wrote no code on the body the issue still has is never run
// again: past claim-hours, the line blocks it for a person instead.

import (
	"fmt"
	"slices"
	"time"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/developer"
	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/verdict"
)

// doingState names the state the developer's claim moves an issue to.
const doingState = "doing"

// claimHours is the developer's claim-hours, as the project sets it,
// read once a run; its default when the role does not load.
func (t *tally) claimHours() int {
	if t.hours != 0 {
		return t.hours
	}
	m := map[string]any{}
	if t.rolesDir != "" {
		if r, err := role.Load(t.rolesDir, developerRole); err == nil {
			if pc, err := role.LoadProjectConfig(t.repo); err == nil {
				m = r.MergedSettings(pc)
			}
		}
	}
	t.hours = developer.SettingsFrom(m).ClaimHours
	return t.hours
}

// claimed says whether is carries the label of the developer's claim:
// read, its pull request or none, never silent (#1084, #1271).
func claimed(cfg *routing.Config, is forge.Issue) bool {
	doing, ok := cfg.State(doingState)
	return ok && slices.Contains(is.Labels, doing.Label)
}

// closedByMR is the open merge request t read that closes issue id.
func closedByMR(t *tally, id int) (forge.MergeRequest, bool) {
	i := slices.IndexFunc(t.mrs, func(m forge.MergeRequest) bool { return slices.Contains(forge.Closes(m.Title, m.Body), id) })
	if i < 0 {
		return forge.MergeRequest{}, false
	}
	return t.mrs[i], true
}

// mrWaits says what a claim waits on once its pull request mr is open
// (#1271), naming who acts next. A ready one: its review, its checks and
// its merge. A Draft: the developer's, a round on it at its checks' end
// or a red one; a person's only when its record says so (a cap reached,
// its agent failed, its rounds handed off: developer.DraftHolder). A
// change to the issue's body is the developer's review round's on mr
// (developer.RoundDue): said by the reconciler when due.
func mrWaits(res *Result, b forge.Backlog, doing routing.State, is forge.Issue, mr forge.MergeRequest) {
	msg := fmt.Sprintf("its claim, %s, waits on its pull request #%d: its review, its checks, then its merge", doing.Label, mr.ID)
	if r, ok := refusedOn(b, is.ID, mr.ID); ok && !mr.Draft {
		// Its merge was due and the forge refused it: said, never a
		// review it does not wait on (#1354).
		msg = fmt.Sprintf("its claim, %s, waits on its pull request #%d, its merge refused at its head %s: %s", doing.Label, mr.ID, short(r.Head), r.Said)
		if r.Conflict != "" {
			msg += fmt.Sprintf("; it waits to be brought up to date with %s", r.Conflict)
		}
	}
	if mr.Draft {
		msg = fmt.Sprintf("its claim, %s, waits on its pull request #%d, a Draft: ", doing.Label, mr.ID) + draftHolder(b, mr)
	}
	res.Findings = append(res.Findings, verdict.Finding{Rule: "state-waits", Level: "info", Where: fmt.Sprintf("#%d", is.ID),
		Message: msg + "; a change to its body is the developer's, at a review round on it"})
}

// refusedOn is the refused merge of merge request mr that issue id's
// comments say, false when none does or they cannot be read: the claim
// then reads as waiting on its review, its checks and its merge.
func refusedOn(b forge.Backlog, id, mr int) (engine.Refusal, bool) {
	comments, err := b.Comments(forge.Target{Kind: "issue", ID: id})
	if err != nil {
		return engine.Refusal{}, false
	}
	r, ok := engine.ReadRefusal(comments)
	return r, ok && r.MR == mr
}

// draftHolder says who acts next on mr, the developer's Draft.
func draftHolder(b forge.Backlog, mr forge.MergeRequest) string {
	f, ok := b.(forge.Forge)
	if !ok {
		return "the developer's, a round on it once its checks end or one is red"
	}
	why, err := developer.DraftHolder(f, mr)
	switch {
	case err != nil:
		return "who holds it not read (" + forge.Said(err) + ")"
	case why != "":
		return why
	}
	return "the developer's, a round on it once its checks end or one is red"
}

// claimStep reads an issue the developer claimed: done false when it
// holds no claim; what it waits on when an open merge request closes it
// (its round, read elsewhere: roundsDue). Else the developer's step when its claim is to be let go,
// claim-hours passed or its body changed since the claim, once for
// that; none, the day it is let go said, while it waits. A claim whose
// agent wrote no code on the body it has now is not let go past
// claim-hours by the developer, which would take it again on the same
// input: the line blocks the issue for a person (write), its claim's
// label off with every state's. s: the state the issue's labels say.
func claimStep(res *Result, cfg *routing.Config, b forge.Backlog, t *tally, s routing.State, is forge.Issue) (*stateStep, *StateWrite, bool, error) {
	doing, ok := cfg.State(doingState)
	if !ok || !slices.Contains(is.Labels, doing.Label) {
		return nil, nil, false, nil
	}
	if err := t.mergeRequests(b, "whether a claim has its pull request"); err != nil {
		return nil, nil, false, err
	}
	if mr, ok := closedByMR(t, is.ID); ok {
		mrWaits(res, b, doing, is, mr)
		return nil, nil, true, nil
	}
	rec, had, _, _, notes, err := readRecordAll(b, is.ID)
	if err != nil {
		return nil, nil, false, err
	}
	trail, err := b.Trail(is.ID, doing.Label)
	if err != nil {
		return nil, nil, false, err
	}
	where, h := fmt.Sprintf("#%d", is.ID), t.claimHours()
	at, dated := backlog.Moment(trail.Labeled)
	free := at.Add(time.Duration(h) * time.Hour)
	now := rec
	if !had {
		now = Record{State: s.Name, Body: bodyHash(is.Body), V: recordV}
	}
	act := developer.ClaimActIn(forge.Bodies(notes))
	same := developer.NoCodeOnSameBody(act, is.Body)
	key, why := "", ""
	switch {
	case dated && !clock().Before(free) && same:
		// Its agent wrote no code on this input: run again, it would get
		// the same one (ADR-0063, I2). A person's, as code said lacking
		// twice is (needs-spent).
		why = fmt.Sprintf("the developer's agent wrote no code on this issue, and its claim, %s put on %s, passed claim-hours (%d) with its body unchanged: "+
			"its claim let go, the developer not run on the same input again; a person rewrites its body, splits it, or takes it",
			doing.Label, backlog.Day(trail.Labeled), h)
		res.Findings = append(res.Findings, verdict.Finding{Rule: "no-code-spent", Level: "warn", Where: where, Message: why + ": moved to " + BlockedState})
		return nil, &StateWrite{Issue: is.ID, Record: blocking(now, s.Name, why, forge.Bodies(notes)), Blocked: why}, true, nil
	case dated && !clock().Before(free):
		key = "hours:" + trail.Labeled
		why = fmt.Sprintf("its claim, %s put on %s, has no pull request past claim-hours (%d): wakes the developer to let it go",
			doing.Label, backlog.Day(trail.Labeled), h)
	case developer.NoCodeOnOtherBody(act, is.Body):
		key = "body:" + bodyHash(is.Body)
		why = "its claim wrote no code, and its body changed since: wakes the developer to let it go"
	}
	if key == "" || rec.Claim == key {
		msg := fmt.Sprintf("its claim, %s, has no pull request", doing.Label)
		switch {
		case key != "":
			msg += ": the developer was woken to let it go already"
		case dated && same:
			msg += fmt.Sprintf(": its agent wrote no code; blocked for a person on %s (claim-hours: %d), or let go at once on a change to its body", free.UTC().Format("2006-01-02 15:04 UTC"), h)
		case dated:
			msg += fmt.Sprintf(": let go on %s (claim-hours: %d), or at once on a change to its body", free.UTC().Format("2006-01-02 15:04 UTC"), h)
		default:
			msg += ": the forge does not say when it began; let go on a change to its body"
		}
		res.Findings = append(res.Findings, verdict.Finding{Rule: "state-waits", Level: "info", Where: where, Message: msg})
		return nil, nil, true, nil
	}
	now.Claim, now.Ran = key, clock().UTC().Format(time.RFC3339)
	res.Findings = append(res.Findings, verdict.Finding{Rule: "state-wakes", Level: "info", Where: where, Message: why})
	return &stateStep{role: developerRole, write: StateWrite{Issue: is.ID, Record: now}, from: s, since: at, rec: rec, tier: backlog.SizeTier(is.Labels)}, nil, true, nil
}
