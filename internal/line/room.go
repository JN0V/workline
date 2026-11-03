package line

// Who goes first, and room that frees up (ADR-0059, amended 2026-11-03,
// #960). The reconciler starts the steps due the one waiting longest
// first: a first step in its state waits since the state's label was put
// on, a step due again in the same state since its last run. A step
// started waits anew from then, behind the others: with more due than
// reconcile-max, each starts within a bounded number of runs, whatever
// its age. And an event that moved an issue, asked with `--input
// room=true`, lists besides it the issues whose step waited for room in
// a state with a limit, due now: room that frees up wakes them at once,
// not at the next reconcile.

import (
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/verdict"
)

// RoomInput, `true` with `--input reconcile=list` on an event naming an
// issue, lists besides it the issues waiting for room that have it now:
// for the CI job that moved a state to start each one's run.
const RoomInput = "room"

// dueStep is a step the reconciler found due, counted in reconcile-max:
// at, its issue in the walk's; step, the role to wake, or block, a write
// its own run makes with no agent.
type dueStep struct {
	at    int
	step  *stateStep
	block *StateWrite
}

// since is how long d waited to run: a block, the line's own write, goes
// first.
func (d dueStep) since() time.Time {
	if d.step == nil {
		return time.Time{}
	}
	return d.step.since
}

// startDue starts the steps due, the one waiting longest first, up to
// reconcile-max with those started already. A step whose next state
// filled up with a step started before it in this run waits: said, its
// wait written when it is run here; in CI (list) left to the next event
// or run, no run started only to say it.
func startDue(res *Result, cfg *routing.Config, f forge.Forge, spec string, o engine.Options, b forge.Backlog, issues []forge.Issue, dues []dueStep, t *tally, started *int, list bool) {
	slices.SortStableFunc(dues, func(x, y dueStep) int { return x.since().Compare(y.since()) })
	for _, d := range dues {
		id := issues[d.at].ID
		where := fmt.Sprintf("#%d", id)
		if *started >= cfg.ReconcileMax {
			res.Findings = append(res.Findings, verdict.Finding{Rule: "reconcile-left", Level: "info", Where: where,
				Message: fmt.Sprintf("due, past reconcile-max (%d) this run: the next run starts it", cfg.ReconcileMax)})
			continue
		}
		if d.step != nil && d.step.next != "" && cfg.Limit(d.step.next) > 0 {
			why, err := waits(cfg, b, d.step.from, &issues, t)
			if err != nil {
				res.Status = worst(res.Status, verdict.Block)
				res.Findings = append(res.Findings, verdict.Finding{Rule: "state-unread", Where: where, Message: err.Error()})
				continue
			}
			if why != "" {
				res.Findings = slices.DeleteFunc(res.Findings, func(f verdict.Finding) bool { return f.Rule == "state-wakes" && f.Where == where })
				if w := waiting(res, d.step.from, why, d.step.rec, where, id); w != nil && !list {
					keepState(res, cfg, f, spec, o, *w, "")
				}
				continue
			}
		}
		*started++
		if d.step != nil {
			anticipate(cfg, &issues[d.at], d.step.next) // the steps after it see it in its next state
			t.moved[id] = true                          // its tests, the tester's result, not in what the run read before
		}
		switch {
		case list: // its own run blocks it, past the cap: nothing written here
			res.Due = append(res.Due, id)
		case d.block != nil:
			keepState(res, cfg, f, spec, o, *d.block, "")
		default:
			runStateStep(res, cfg, f, spec, o, *d.step)
		}
	}
}

// roomDue adds to res.Due, besides those in it, the issues whose step
// waits for room in a state with a limit, and has it now: the room the
// event's move freed, or one an event before missed; it returns them.
// Their steps alone, up to reconcile-max, the one waiting longest first;
// a state still full is not read further. What it could not read is
// said, the reconciler starting them within the hour.
func roomDue(res *Result, cfg *routing.Config, o engine.Options) []int {
	unread := func(err error) {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "room-unread", Level: "warn", Where: "issues",
			Message: "the issues waiting for room not read; the reconciler starts them: " + forge.Said(err)})
	}
	f, spec, err := openForge(o)
	if err != nil {
		unread(err)
		return nil
	}
	b, ok := f.(forge.Backlog)
	if !ok {
		return nil
	}
	issues, err := b.Issues()
	if err != nil {
		unread(err)
		return nil
	}
	backlog.Order(issues)
	x, t := newScopeIndex(o.Repo), newTallyFor(o)
	full := map[string]bool{}
	var dues []dueStep
	read := &Result{} // what the line says of each: its own event's, not this one's
	for i, is := range issues {
		if slices.Contains(res.Due, is.ID) {
			continue
		}
		s, name, _ := cfg.Wakes(is.Labels, IssueEvent)
		if name == "" || s.Next == "" || cfg.Limit(s.Next) == 0 {
			continue
		}
		no, seen := full[s.Next]
		if !seen {
			why, err := waits(cfg, b, s, &issues, t)
			if err != nil {
				unread(err)
				return nil
			}
			no, full[s.Next] = why != "", why != ""
		}
		if no {
			continue
		}
		step, _, err := stateOf(read, cfg, b, is, &issues, x, t)
		if err != nil {
			unread(err)
			continue
		}
		if step != nil { // a write with no agent, a wait or a block, is the reconciler's
			dues = append(dues, dueStep{at: i, step: step})
		}
	}
	before, started := len(res.Due), 0
	startDue(res, cfg, f, spec, o, b, issues, dues, t, &started, true)
	woken := res.Due[before:]
	for _, n := range woken {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "state-room", Level: "info", Where: fmt.Sprintf("#%d", n),
			Message: "its next state has room now: its step due"})
	}
	return woken
}

// roomSaid is what a summary says of the issues room woke: "" for none.
func roomSaid(woken []int) string {
	if len(woken) == 0 {
		return ""
	}
	said := fmt.Sprintf("room: %d %s due:", len(woken), plural(len(woken), "issue", "issues"))
	for _, n := range woken {
		said += " #" + strconv.Itoa(n)
	}
	return said
}
