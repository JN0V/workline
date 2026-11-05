package line

// A run's typed outcome on the line (ADR-0063, §6): a machine error
// moves no label, no state, no hop (I3). Its step is run again on the
// same input after a back-off, within a budget of tries per kind of
// error; spent, the issue waits for a person, the error said. A
// needs-input of the engine, the code its agent named, is given at the
// step's next run, after the same back-off: not in the run that named
// it, which would double a reconcile's forge requests; named again, a
// person gives it.

import (
	"fmt"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/verdict"
)

// ErrorBudget is how many times the line runs a step again after a
// machine error of one kind, within ErrorWindow of the first; the next
// error of that kind waits for a person (ADR-0063, decided 4).
const ErrorBudget = 2

// ErrorWindow bounds the tries after the first machine error: past it,
// the budget is spent. A window that gave it back would loop once a day.
const ErrorWindow = 24 * time.Hour

// ErrorBackoff is the least time between a machine error and the run
// again: the same reconcile, or an event right after, starts nothing.
const ErrorBackoff = 10 * time.Minute

// NeedsInput gives a role run again for a needs-input of the engine what
// its agent said it lacked: the paths it named, comma-separated.
const NeedsInput = "needs"

// Errors are the machine errors of a step's runs on its input, in its
// state: by kind, how many; when the first and the last came, the
// last's kind and what it said. A new input — the state entered anew,
// its body or labelling changed by a person — starts none.
type Errors struct {
	By    map[string]int `json:"by"`
	First string         `json:"first"`
	Last  string         `json:"last"`
	Kind  string         `json:"kind"`
	Said  string         `json:"said,omitempty"`
}

// Needs is the code a step's agent said it lacked on its input, the
// last time (ADR-0063, §6): when, and how many times in a row. Given at
// the next run, after ErrorBackoff; named twice, a person gives it.
type Needs struct {
	Said  string `json:"said"`
	At    string `json:"at"`
	Times int    `json:"times"`
}

// typedDue is what the last run's typed outcome makes of a step whose
// input is the same as its record's (rec): due again (now, with the
// inputs the run gets), a wait said (waits), or the issue blocked for a
// person (block). Neither a machine error nor short code: not due, rec.
func typedDue(res *Result, name string, s routing.State, is forge.Issue, rec Record, comments []string) (due bool, now Record, inputs map[string]string, block *StateWrite, waits bool) {
	where, at := fmt.Sprintf("#%d", is.ID), clock().UTC()
	wait := func(why string, until time.Time) (bool, Record, map[string]string, *StateWrite, bool) {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "state-waits", Level: "info", Where: where,
			Message: why + ": run again from " + until.UTC().Format("2006-01-02 15:04 UTC")})
		return false, rec, nil, nil, true
	}
	blocked := func(rule, why string) (bool, Record, map[string]string, *StateWrite, bool) {
		res.Findings = append(res.Findings, verdict.Finding{Rule: rule, Level: "warn", Where: where, Message: why + ": moved to " + BlockedState})
		return false, rec, nil, &StateWrite{Issue: is.ID, Record: blocking(rec, s.Name, why, comments), Blocked: why}, false
	}
	if errs := errorsOf(rec); errs != nil {
		again, until, spent := errs.retry(at)
		switch {
		case spent != "":
			return blocked("machine-error-spent", fmt.Sprintf("the %s's runs on it failed in its state %s: %s", name, s.Name, spent))
		case !again:
			return wait(fmt.Sprintf("the %s's last run on it failed (%s)", name, errs.Kind), until)
		}
		now = rec
		now.Errors, now.Lost = errs, ""
		res.Findings = append(res.Findings, verdict.Finding{Rule: "state-retry", Level: "info", Where: where,
			Message: fmt.Sprintf("the %s's last run on it failed (%s): run again, try %d of %d", name, errs.Kind, errs.By[errs.Kind], ErrorBudget)})
		return true, now, map[string]string{engine.FailedInput: errs.Said}, nil, false
	}
	if n := rec.Needs; n != nil {
		last, _ := time.Parse(time.RFC3339, n.At)
		switch {
		case n.Times > 1:
			return blocked("needs-spent", fmt.Sprintf("the %s said again it lacks code in its state %s, given what it named: %s; a person names the files it needs in a comment", name, s.Name, n.Said))
		case at.Before(last.Add(ErrorBackoff)):
			return wait(fmt.Sprintf("the %s said it lacks code (%s)", name, n.Said), last.Add(ErrorBackoff))
		}
		res.Findings = append(res.Findings, verdict.Finding{Rule: "state-given", Level: "info", Where: where,
			Message: fmt.Sprintf("the %s said it lacks code: run again, given %s", name, n.Said)})
		return true, rec, map[string]string{NeedsInput: n.Said}, nil, false
	}
	return false, rec, nil, nil, false
}

// errorsOf are the machine errors a record holds: its own, or a
// developer's run lost before they were typed (#953), one of kind run,
// with no window: the first error of its tries is the next.
func errorsOf(r Record) *Errors {
	if r.Errors != nil || r.Lost == "" {
		return r.Errors
	}
	return &Errors{By: map[string]int{"run": 1}, Last: r.Lost, Kind: "run", Said: engine.ErrNoVerdict.Error()}
}

// with is e and one more error, o, at now.
func (e *Errors) with(o verdict.Outcome, now time.Time) *Errors {
	n := &Errors{By: map[string]int{}, First: now.Format(time.RFC3339)}
	if e != nil {
		for k, v := range e.By {
			n.By[k] = v
		}
		if e.First != "" {
			n.First = e.First
		}
	}
	n.By[o.Of]++
	n.Last, n.Kind, n.Said = now.Format(time.RFC3339), o.Of, oneLine(o.Said)
	return n
}

// oneLine is a run's word on its error, one line, short enough for a
// record kept in a comment.
func oneLine(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > 600 {
		s = string(r[:600]) + "…"
	}
	return s
}

// retry is what a step's machine errors make of it at now: run again
// (due), wait until a time (until), or wait for a person (spent, why).
func (e *Errors) retry(now time.Time) (due bool, until time.Time, spent string) {
	first, err1 := time.Parse(time.RFC3339, e.First)
	last, err2 := time.Parse(time.RFC3339, e.Last)
	switch {
	case e.By[e.Kind] > ErrorBudget:
		return false, time.Time{}, fmt.Sprintf("its run failed %d times (%s): %s", e.By[e.Kind], e.Kind, e.Said)
	case err1 == nil && now.Sub(first) >= ErrorWindow:
		return false, time.Time{}, fmt.Sprintf("its run failed (%s), and was run again for %d h: %s", e.Kind, int(ErrorWindow.Hours()), e.Said)
	case err2 == nil && now.Before(last.Add(ErrorBackoff)):
		return false, last.Add(ErrorBackoff), ""
	}
	return true, time.Time{}, ""
}

// outcomeOf is the outcome of the woken role's own run, the first of
// steps, which ended with status; a progress when none ran.
func outcomeOf(steps []Step, status string) verdict.Outcome {
	if len(steps) == 0 || steps[0].Result == nil {
		return verdict.Outcome{Kind: verdict.Progress}
	}
	return verdict.OutcomeOf(status, steps[0].Result.Findings)
}

// failedBody is what the line's comment says of a step's machine error:
// what failed, and when it runs again, or that a person is waited on.
func failedBody(w StateWrite, role string) string {
	e := w.Record.Errors
	who := "the " + strings.ReplaceAll(role, "-", " ")
	last, _ := time.Parse(time.RFC3339, e.Last)
	next := fmt.Sprintf("the line runs %s again on it from %s, %d of %d tries for this kind of error; the issue's labels and state are kept",
		who, last.Add(ErrorBackoff).UTC().Format("2006-01-02 15:04 UTC"), min(e.By[e.Kind], ErrorBudget), ErrorBudget)
	if e.By[e.Kind] > ErrorBudget {
		next = "its tries are spent: the line moves it to Blocked for a person at its next run, the error said"
	}
	return backlog.Said("workline's line", fmt.Sprintf("saw %s's run on this issue fail (%s)", who, e.Kind), e.Said, next)
}
