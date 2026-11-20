package line

// A run's typed outcome on the line (ADR-0063, §6): a machine error
// moves no label, no state, no hop (I3). Its step is run again on the
// same input after a back-off, within a budget of tries per kind of
// error; spent, the issue waits for a person, the error said. A
// needs-input of the engine, the code its agent named, is Due at once:
// the role runs again, once, in the same step, given it; said lacking
// again, a person names the files.

import (
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/kit/verdict"
	"github.com/JN0V/workline/internal/routing"
)

// ErrorBudget is how many times the line runs a step again after a
// machine error of one kind, within ErrorWindow of the first; the next
// error of that kind waits for a person (ADR-0063, decided 4).
const ErrorBudget = 2

// ErrorWindow bounds the tries after the first machine error: past it,
// the budget is spent. A window that gave it back would loop once a day.
const ErrorWindow = 24 * time.Hour

// AnswerBudget is how many runs in a row may end with no answer the
// role could use (kind answer, as the tester's no-test-written): the same
// input gets the same answer, so each try is an agent call for nothing.
const AnswerBudget = 2

// AnswerHold is the least time between such a run and the next on the
// same body: longer than a machine error's back-off, a body changed by a
// person lets it try at once (#1917).
const AnswerHold = 6 * time.Hour

// ErrorBackoff is the least time between a machine error and the run
// again: the same reconcile, or an event right after, starts nothing.
const ErrorBackoff = 10 * time.Minute

// Errors are the machine errors of a step's runs on its input, in its
// state: by kind, how many and when its first came, its own window; when
// the last came, its kind and what it said. A new input — the state
// entered anew, its body or labelling changed by a person — starts none.
type Errors struct {
	By    map[string]int    `json:"by"`
	First map[string]string `json:"first,omitempty"`
	Last  string            `json:"last"`
	Kind  string            `json:"kind"`
	Said  string            `json:"said,omitempty"`
}

// Needs is the code a step's agent said it lacked on its input, the
// last time (ADR-0063, §5): when, and how many times in a row. Given at
// once, the role run again in the same step; said lacking once given
// it (Times 2), a person names the files.
type Needs struct {
	Said  string `json:"said"`
	At    string `json:"at"`
	Times int    `json:"times"`
}

// LastRun is the input a role's run was on, its state, body, input
// labels, spec review, the code the product owner was given and the
// comments the engine did not write, and what its agent said it lacked
// there, if any (#1157). Times 2: said again once given it. A step due
// anew on that same input, a revisit having moved the issue back with
// nothing in it changed, is not run as before: given what was named
// (Times 1), or blocked again for a person, no agent called.
type LastRun struct {
	State  string `json:"state"`
	Body   string `json:"body"`
	Labels string `json:"labels,omitempty"`
	Spec   string `json:"spec,omitempty"`
	Given  string `json:"given,omitempty"`
	Talk   string `json:"talk,omitempty"`
	Said   string `json:"said,omitempty"`
	Times  int    `json:"times,omitempty"`
	// Idle: its run left the issue as it found it, nothing drafted
	// (#1800).
	Idle bool `json:"idle,omitempty"`
}

// lastOn is role's last run in r when its step, due on now, is on its
// input: talk and given, its comments' hash and the code it would be
// given.
func lastOn(r Record, role string, now Record, talk, given string) (LastRun, bool) {
	l, ok := r.Last[role]
	return l, ok && l.State == now.State && l.Body == now.Body && l.Labels == now.Labels &&
		l.Spec == now.Spec && l.Given == given && l.Talk == talk
}

// withLast is last with role's run set to l, or taken out when l is nil:
// a copy, the record read never changed.
func withLast(last map[string]LastRun, role string, l *LastRun) map[string]LastRun {
	next := maps.Clone(last)
	if l == nil {
		delete(next, role)
		if len(next) == 0 {
			return nil
		}
		return next
	}
	if next == nil {
		next = map[string]LastRun{}
	}
	next[role] = *l
	return next
}

// saysNew says whether what an agent said it lacks names none of what it
// was given for its last saying: each a path, the two lists split on
// commas, an anchor (`a.go#F`) or a folder's slash aside.
func saysNew(given, said string) bool {
	bare := func(w string) string {
		w, _, _ = strings.Cut(strings.TrimSpace(w), "#")
		return strings.TrimSuffix(strings.TrimPrefix(w, "./"), "/")
	}
	was := map[string]bool{}
	for _, w := range strings.Split(given, ",") {
		was[bare(w)] = true
	}
	some := false
	for _, w := range strings.Split(said, ",") {
		if w = bare(w); w == "" {
			continue
		}
		if was[w] {
			return false
		}
		some = true
	}
	return some
}

// needsCode says whether an outcome is a needs-input of the engine: the
// code its agent said it lacked.
func needsCode(o verdict.Outcome) bool {
	return o.Kind == verdict.NeedsInput && o.Of == verdict.FromEngine
}

// sameState says whether the issue of step st is still in the state the
// step ran in, read again: a role that moved it is not run again there.
// One that does not read is not either.
func sameState(f forge.Forge, cfg *routing.Config, st stateStep) bool {
	is, err := f.Issue(st.write.Issue)
	if err != nil {
		return false
	}
	s, ok := cfg.StateOf(is.Labels)
	return ok && s.Name == st.from.Name
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
		res.Waiting = append(res.Waiting, is.ID) // an error backoff pending: the heartbeat goes on
		return false, rec, nil, nil, true
	}
	blocked := func(rule, why string) (bool, Record, map[string]string, *StateWrite, bool) {
		res.Findings = append(res.Findings, verdict.Finding{Rule: rule, Level: "warn", Where: where, Message: why + ": moved to " + BlockedState})
		return false, rec, nil, &StateWrite{Issue: is.ID, Record: blocking(rec, s.Name, why, comments), Blocked: why}, false
	}
	if errs := errorsOf(rec); errs != nil {
		again, until, spent := errs.retry(at)
		switch {
		case spent != "" && name == "tester" && errs.Kind == "answer":
			return blocked("no-test-written-spent", fmt.Sprintf("the tester could not propose a test file in its state %s: %s", s.Name, spent))
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
		if n.Times > 1 {
			return blocked("needs-spent", fmt.Sprintf("the %s said again it lacks code in its state %s, given what it named: %s; a person names the files it needs in a comment", name, s.Name, n.Said))
		}
		// Not given in its step (it did not end, or read): given now.
		res.Findings = append(res.Findings, verdict.Finding{Rule: "state-given", Level: "info", Where: where,
			Message: fmt.Sprintf("the %s said it lacks code: run again, given %s", name, n.Said)})
		return true, rec, map[string]string{engine.NeedsInput: n.Said}, nil, false
	}
	return false, rec, nil, nil, false
}

// errorsOf are the machine errors a record holds: its own, or a
// developer's run lost before they were typed (#953), one of kind run,
// with no window: its time was not kept as one.
func errorsOf(r Record) *Errors {
	if r.Errors != nil || r.Lost == "" {
		return r.Errors
	}
	return &Errors{By: map[string]int{"run": 1}, Last: r.Lost, Kind: "run", Said: engine.ErrNoVerdict.Error()}
}

// with is e and one more error, o, at now.
func (e *Errors) with(o verdict.Outcome, now time.Time) *Errors {
	n := &Errors{By: map[string]int{}, First: map[string]string{}}
	if e != nil {
		for k, v := range e.By {
			n.By[k] = v
		}
		for k, v := range e.First {
			n.First[k] = v
		}
	}
	if n.By[o.Of] == 0 {
		n.First[o.Of] = now.Format(time.RFC3339) // its kind's first error: its window starts
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
	first, err1 := time.Parse(time.RFC3339, e.First[e.Kind]) // the window is its kind's
	last, err2 := time.Parse(time.RFC3339, e.Last)
	switch {
	case e.Kind == verdict.Size || e.Kind == "budget":
		// A size error recorded before it was blocked at once (#1160),
		// "budget" its name then: no try mends it.
		return false, time.Time{}, fmt.Sprintf("its task is too big for its context (%s): %s", e.Kind, e.Said)
	case e.Kind == ApplyKind && e.By[e.Kind] >= ApplyBudget:
		return false, time.Time{}, fmt.Sprintf("its patch did not apply on %d runs in a row (%s): %s; the line does not compare heads: a person restarts it, worth doing once its tester's Draft or `main` has moved", e.By[e.Kind], e.Kind, e.Said)
	case e.Kind == "answer" && e.By[e.Kind] >= AnswerBudget:
		return false, time.Time{}, fmt.Sprintf("its run ended %d times with no answer it could use on the same body, so it could not propose what it was asked (%s): %s; a person rewrites its body and puts its label back, or does the work", e.By[e.Kind], e.Kind, e.Said)
	case e.Kind == "answer" && err2 == nil && now.Before(last.Add(AnswerHold)):
		return false, last.Add(AnswerHold), ""
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

// ApplyKind is the kind of machine error of a developer's patch that does
// not apply on the tree the apply job meets it on: the base moved since
// the tester's Draft (#1775).
const ApplyKind = "apply"

// ApplyBudget is how many such failures in a row stop the line: a patch
// that missed twice misses again until its base moves, and each run is an
// agent call.
const ApplyBudget = 2

// CountApplyFailure writes, for each issue the line ran a step for, the
// failure of its run's apply as a machine error in its record, said on
// the issue with its labels and state kept: the apply job could not apply
// what the judge found, said is what it printed, run the folder of the
// run that failed: its issue alone is counted. A failure of the same
// input before it is counted with it.
func CountApplyFailure(f forge.Forge, repo string, res *Result, said, run string) error {
	if f == nil || len(res.States) == 0 {
		return nil
	}
	cfg, err := routing.Load(repo)
	if err != nil {
		return err
	}
	kind := "run"
	if strings.Contains(said, errNotAppliedText) {
		kind = ApplyKind
	}
	now := clock().UTC()
	var errs []error // one issue's failure to read or write does not stop the others
	for _, w := range res.States {
		role := res.Woke[w.Issue]
		if role == "" || w.Blocked != "" || !slices.ContainsFunc(w.Runs, func(r string) bool { return filepath.Base(r) == filepath.Base(run) }) {
			continue
		}
		b, ok := f.(forge.Backlog)
		if !ok {
			errs = append(errs, fmt.Errorf("#%d: the forge has no comments to read its previous failures from", w.Issue))
			continue
		}
		comments, err := b.Comments(forge.Target{Kind: "issue", ID: w.Issue})
		if err != nil {
			errs = append(errs, fmt.Errorf("#%d: read its previous failures: %w", w.Issue, err))
			continue
		}
		var prev *Errors
		if _, rec, had := recordCommentIn(comments); had && rec.State == w.Record.State && rec.Body == w.Record.Body {
			prev = errorsOf(rec)
		}
		w.Record.Errors = prev.with(verdict.Outcome{Kind: verdict.MachineError, Of: kind, Said: said}, now)
		w.Record.Lost, w.Record.Needs, w.Failed, w.After = "", nil, true, false
		if err := writeState(f, cfg, repo, w, role); err != nil {
			errs = append(errs, fmt.Errorf("#%d: %w", w.Issue, err))
		}
	}
	return errors.Join(errs...)
}

// errNotAppliedText is what the engine says of a diff git does not apply.
const errNotAppliedText = "the diff does not apply"
