package line

// The line of states (ADR-0059, decision 8): an event naming an issue
// runs the role its state wakes, read from the routing's table by a pure
// function (routing.Config.Wakes), not every step of the event; once,
// for a state and body the line's record does not hold yet. The
// reconciler walks every open issue for a step due that no event
// started.

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/developer"
	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/verdict"
)

// IssueEvent is the event naming one issue: its labelling on a forge, a
// state started by the step before (ADR-0059).
const IssueEvent = "issue"

// ReconcileEvent walks every open issue and starts the steps due
// (ADR-0059): hourly in CI, a command on a person's machine.
const ReconcileEvent = "reconcile"

// ReconcileInput, `list`, makes the reconciler, or the issue line, say
// the issues due and start none: a CI job then starts each one's own
// run.
const ReconcileInput = "reconcile"

// LastInput, `last`, is when the reconciler last ran, RFC 3339, as the CI
// keeps it: a gap past ReconcileLate is said in its summary, the
// forge's schedule being late or skipped (#789).
const LastInput = "last"

// ReconcileLate is the gap since the last reconcile said as late: twice
// the hour it is meant to run in (ADR-0059, decision 8).
const ReconcileLate = 2 * time.Hour

// openForge opens the forge a line reads issues from: the one given, else
// the project's; nil with none.
func openForge(o engine.Options) (forge.Forge, string, error) {
	spec := o.Forge
	if spec == "" {
		if cfg, err := role.LoadProjectConfig(o.Repo); err == nil {
			spec = cfg.Forge
		}
	}
	if spec == "" || spec == "none" {
		return nil, spec, errors.New("no forge to read the issues' states from; " + forge.Missing)
	}
	f, err := forge.Open(spec, o.Repo)
	return f, spec, err
}

// stateStep is one issue's step the line runs: the role its state wakes,
// and the record it writes once that ran.
type stateStep struct {
	role  string
	next  string
	write StateWrite
	// inputs are given the role besides the run's: what is new on a
	// blocked issue it reads again (revisit.go).
	inputs map[string]string
	// from: the state it runs in; since: how long it has waited to run,
	// the reconciler's order (ADR-0059, amended 2026-11-03); rec: its
	// record as read, kept when it waits for room after all.
	from  routing.State
	since time.Time
	rec   Record
}

// never is a step's since when how long it waited is not known: a re-try
// in its state on a record from before the line kept its last run.
var never = time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)

// waitedSince is how long the step of state s on an issue has waited to
// run: the first step in its state, since the state's label was put on
// (labeled; unknown, the oldest); a step due again in the same state,
// since its last run (rec.Ran; unknown, never: after the others).
func waitedSince(s routing.State, rec Record, had bool, labeled string) time.Time {
	if had && rec.State == s.Name {
		if t, err := time.Parse(time.RFC3339, rec.Ran); err == nil {
			return t
		}
		return never
	}
	t, _ := time.Parse(time.RFC3339, labeled)
	return t
}

// stateOf reads where issue is on the line and whether its step is due:
// the step to run, else nil, what the line says of it added to res. A
// step past the cap is not run: the issue is to be blocked (write); nor
// one whose next state is full: it waits (write, once). open: the open
// issues, listed at first need. A stale issue goes back to refine
// first (write, stale.go), whatever its state wakes. A blocked one is
// read again by the product owner when there is news on it (revisit.go).
// A ready one without its tests goes to them (write, wip.go), whether
// the developer was woken on it already or not. t: what the limits are
// counted from besides the open issues, kept for the run.
func stateOf(res *Result, cfg *routing.Config, b forge.Backlog, is forge.Issue, open *[]forge.Issue, x *scopeIndex, t *tally) (*stateStep, *StateWrite, error) {
	where := fmt.Sprintf("#%d", is.ID)
	s, name, why := cfg.Wakes(is.Labels, IssueEvent)
	if w, err := stale(res, cfg, b, x, is, s); err != nil || w != nil {
		return nil, w, err
	}
	if s.Name == BlockedState && name == "" {
		return revisit(res, cfg, b, is, open)
	}
	if name == "" {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "state-wakes-none", Level: "info", Where: where, Message: why})
		return nil, nil, nil
	}
	rec, had, spec, refused, notes, err := readRecordAll(b, is.ID)
	if err != nil {
		return nil, nil, err
	}
	comments := forge.Bodies(notes)
	trail, err := b.Trail(is.ID, s.Label)
	if err != nil {
		return nil, nil, err
	}
	if w, held, err := retest(res, cfg, b, t, s, is, rec, had, trail.Labeled, comments); err != nil || held {
		return nil, w, err
	}
	if s.Role == "tester" && slices.Contains(is.Labels, backlog.LabelNeedsRepro) {
		// A bug its tests did not show: its reporter's answer, or the
		// delay, moves it; with no agent (ADR-0060).
		if w, held, err := repro(res, b, t, s, is, rec, had, trail.Labeled, notes); err != nil || held {
			return nil, w, err
		}
	}
	if s.Name != refineState {
		spec = "" // a review is read in refine alone: no other state woken by it
	}
	now, ok := due(cfg, s, is, trail.Labeled, spec, rec, had)
	reviewed := s.Name == refineState && slices.Contains(cfg.Events[IssueEvent], backlog.SpecReviewer)
	if !ok && reviewed && rec.Asked != rec.Body {
		// Acted on, the reviewer never asked on this body: a record from
		// before it read the spec in refine's own run (#840). Due again,
		// once, while its spec waits on a review.
		ok = specWaits(is, comments)
	}
	if !ok && reviewed && rec.Asked == rec.Body && rec.ReviewAgain != rec.Body && specWaits(is, comments) {
		// Asked on this body, and still no review (#1070): the reviewer
		// skipped it then, as it did an issue whose Need is its text on
		// top (#1062). Due once more, for the reviewer; then acted.
		ok = true
		now.ReviewAgain = rec.Body
		res.Findings = append(res.Findings, verdict.Finding{Rule: "state-review-missing", Level: "info", Where: where,
			Message: "its drafts wait on a spec review that has not come on this text: woken again, once, for the reviewer"})
	}
	if !ok && reviewed && rec.Revise != rec.Spec {
		// The review answered by asking a person for a section the product
		// owner rewrites now, its Verification or Scope (ADR-0060): due
		// again, once a review, the section rewritten without the person.
		st, _, _ := backlog.ReadState(comments, name) // one that does not read gives none to rewrite
		ok = len(backlog.Unrevised(is, st, comments)) > 0
	}
	if !ok && s.Name == refineState && name == backlog.Keeper {
		// A question, or drafts, waiting on a person past the product
		// owner's delay: woken once, to take its assumption (ADR-0060).
		key, said, err := timedOut(b, t, is, notes, rec)
		if err != nil {
			return nil, nil, err
		}
		if key != "" {
			ok, now.Timeout = true, key
			res.Findings = append(res.Findings, verdict.Finding{Rule: "state-timeout", Level: "info", Where: where, Message: said})
		}
	}
	if !ok && s.Name == refineState && name == backlog.Keeper {
		// Read on this body and left unrefined, nothing waiting on a
		// person (#951): not acted; woken again, then blocked — at once
		// when its code is the same as last read (#980).
		if lacks, sources := unrefined(is, notes); lacks != "" {
			shown := append(slices.Clone(sources), backlog.PathsIn(x.repo, is.Title+"\n"+is.Body)...)
			again, said, why := refineAgain(rec, lacks, sources, shown)
			if why != "" {
				res.Findings = append(res.Findings, verdict.Finding{Rule: "refine-capped", Level: "warn", Where: where, Message: why + ": moved to " + BlockedState})
				return nil, &StateWrite{Issue: is.ID, Record: blocking(rec, s.Name, why, comments), Blocked: why}, nil
			}
			ok, now = true, again
			res.Findings = append(res.Findings, verdict.Finding{Rule: "state-unrefined", Level: "info", Where: where, Message: said})
		}
	}
	if !ok && s.Name == refineState && name == backlog.Keeper && had && rec.Unread != rec.Body && unreadBody(notes) {
		// Woken on this body, and its state says the product owner never
		// read it (#1041): that run read nothing. Woken again, once a body.
		ok, now = true, rec
		now.Unread = rec.Body
		res.Findings = append(res.Findings, verdict.Finding{Rule: "state-unread-body", Level: "info", Where: where,
			Message: "the product owner's state says it never read this issue: woken again, once a body"})
	}
	if ok && s.Name == refineState && name == backlog.Keeper {
		now.Given = givenNow(comments) // the code this read gets: the same next time wakes it no more (#980)
	}
	if ok && reviewed {
		now.Asked = now.Body // the reviewer runs after the product owner: never due again for this alone
		now.Revise = now.Spec
	}
	if !ok && s.Role == "tester" && refused != "" {
		// The tester's run on it refused, its state the same: never left
		// as acted, which nothing would wake again (#794). Blocked, why
		// on the board; a person moves it back.
		why := "the tester wrote no tests for it in its state " + s.Name + ": " + refused
		res.Findings = append(res.Findings, verdict.Finding{Rule: "tests-refused", Level: "warn", Where: where, Message: why + ": moved to " + BlockedState})
		return nil, &StateWrite{Issue: is.ID, Record: blocking(now, s.Name, why, comments), Blocked: why}, nil
	}
	if !ok && had && s.Role == developerRole && rec.Lost != "" {
		// The developer's last run on it ended with no verdict read:
		// nothing it did reached the forge (#953). Due again, once; the
		// hop its loss counted stops a run lost each time at max-hops.
		ok, now = true, rec
		now.Lost = ""
		res.Findings = append(res.Findings, verdict.Finding{Rule: "state-lost", Level: "info", Where: where,
			Message: "the developer's last run on it, " + rec.Lost + ", ended with no verdict: woken again"})
	}
	if !ok && had && s.Role == developerRole {
		// Left free by the developer since it was woken: its claim let
		// go, or the issue it was passed over for held no more (#894).
		rec, why, err := freed(cfg, b, t, s, is, rec, comments, open)
		if err != nil {
			return nil, nil, err
		}
		if why != "" {
			ok, now = true, rec
			res.Findings = append(res.Findings, verdict.Finding{Rule: "state-freed", Level: "info", Where: where, Message: why})
		}
	}
	if !ok && s.Name == refineState && name == backlog.Keeper && had {
		// The tester found no test possible, and the product owner's run
		// on that text picked no proof nor rewrote it (#1053): neither
		// can settle it. Blocked, the tester's reason said — never left
		// in refine with nothing due.
		if reason, waits := noTestUnsettled(is, comments); waits {
			why := "the tester found no test possible, and the product owner's run picked no proof nor rewrote its text"
			res.Findings = append(res.Findings, verdict.Finding{Rule: "no-test-unsettled", Level: "warn", Where: where,
				Message: why + ": moved to " + BlockedState + "; the tester said: " + reason})
			return nil, &StateWrite{Issue: is.ID, Record: blocking(rec, s.Name, why+"; the tester said: "+reason, comments), Blocked: why + "; the tester said: " + reason}, nil
		}
	}
	if !ok {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "state-acted", Level: "info", Where: where,
			Message: fmt.Sprintf("its state, %s, woke the %s already, on the same body and labelling: nothing more to run", s.Name, name)})
		return nil, nil, nil
	}
	if mr, err := roundOn(b, t, s, is.ID); err != nil {
		return nil, nil, err
	} else if mr != nil {
		// Back from a Scope ask or a hold (#1013): its round resumes on
		// its pull request, which the developer would pass over as a
		// claim. Nothing written: the round moves it.
		res.Findings = append(res.Findings, verdict.Finding{Rule: "state-round", Level: "info", Where: where,
			Message: fmt.Sprintf("its step is the developer's review round on its pull request %d, not a new claim", mr.ID)})
		return nil, nil, nil
	}
	if cfg.MaxHops > 0 && now.Hops > cfg.MaxHops {
		why := fmt.Sprintf("%d state steps on this issue, past max-hops (%d): roles sent it back and forth; its state was %s",
			now.Hops, cfg.MaxHops, s.Name)
		res.Findings = append(res.Findings, verdict.Finding{Rule: "hops-capped", Level: "warn", Where: where, Message: why + ": moved to " + BlockedState})
		return nil, &StateWrite{Issue: is.ID, Record: blocking(now, s.Name, why, comments), Blocked: why}, nil
	}
	if why, err := waits(cfg, b, s, open, t); err != nil {
		return nil, nil, err
	} else if why != "" {
		return nil, waiting(res, s, why, rec, where, is.ID), nil
	}
	if now, err = woken(cfg, b, t, s, is.ID, now, open); err != nil {
		return nil, nil, err
	}
	res.Findings = append(res.Findings, verdict.Finding{Rule: "state-wakes", Level: "info", Where: where,
		Message: fmt.Sprintf("its state, %s, wakes the %s", s.Name, name)})
	since := waitedSince(s, rec, had, trail.Labeled)
	now.Ran = clock().UTC().Format(time.RFC3339)
	return &stateStep{role: name, next: s.Next, write: StateWrite{Issue: is.ID, Record: now}, from: s, since: since, rec: rec}, nil, nil
}

// blocking is the record of an issue the line blocks, from state from:
// why, and what it is now, the base news is told against (revisit.go).
func blocking(r Record, from, why string, comments []string) Record {
	r.State, r.From, r.Why = BlockedState, from, why
	r.Since, r.Talk = clock().UTC().Format(time.RFC3339), talkHash(comments)
	return r
}

// readFailed ends res on an issue that could not be read.
func readFailed(res *Result, event, where string, err error) {
	res.Status = verdict.Block
	if errors.Is(err, forge.ErrUnreachable) {
		res.Status = verdict.BlockedExternal
	}
	res.Summary = fmt.Sprintf("%s: %s could not be read", event, where)
	res.Findings = append(res.Findings, verdict.Finding{Rule: "state-unread", Where: where, Message: err.Error()})
}

// byState runs the step an event naming an issue wakes (ADR-0059): the
// role its state wakes, when the project routes it on issue and the
// record does not say it ran already; none, said.
func byState(res *Result, cfg *routing.Config, o engine.Options, event string, steps []string) *Result {
	where := fmt.Sprintf("#%d", o.Target.ID)
	f, spec, err := openForge(o)
	var b forge.Backlog
	if err == nil {
		var ok bool
		if b, ok = f.(forge.Backlog); !ok {
			err = errors.New("this forge lists no issues' comments: the line keeps no record there")
		}
	}
	var is *forge.Issue
	if err == nil {
		switch said, deferred := waitBudget(res, f, clock()); {
		case deferred: // its step stays due: the reconciler runs it after the reset (#804)
			res.Summary = fmt.Sprintf("%s: %s deferred: %s", event, where, said)
			return res
		case said != "":
			defer func() { res.Summary += "; " + said }()
		}
		is, err = f.Issue(o.Target.ID)
	}
	if err != nil {
		readFailed(res, event, where, err)
		return res
	}
	if is.Closed {
		res.Summary = fmt.Sprintf("%s: %s is closed: no state wakes a role", event, where)
		return res
	}
	t := newTallyFor(o)
	step, block, err := stateOf(res, cfg, b, *is, new([]forge.Issue), newScopeIndex(o.Repo), t)
	if err != nil {
		readFailed(res, event, where, err)
		return res
	}
	if mr, ok := t.rounds[o.Target.ID]; ok && step == nil && block == nil {
		byRound(res, f, o, event, o.Target.ID, mr)
		return res
	}
	switch {
	case block != nil && o.Inputs[ReconcileInput] == "list":
		// Asked only: its own run blocks it, or says it waits, nothing
		// written here.
		res.Due = []int{o.Target.ID}
		why := "past its hop cap"
		switch res.Findings[len(res.Findings)-1].Rule { // stateOf's last word on it
		case "tests-refused":
			why = "to be blocked, its tests refused"
		case "refine-capped":
			why = "to be blocked, left unrefined"
		case "not-reproduced":
			why = "to be blocked, not reproduced"
		case "repro-answered":
			why = "its reporter answered, to go back to refine"
		case "state-waits":
			why = "to say it waits for room"
		case "state-stale":
			why = "stale, to go back to refine"
		case "state-stale-kept":
			why = "stale, to say it is left"
		case "state-blocked-seen":
			why = "seen blocked, the line's record to write"
		case "state-untested":
			why = "without its tests, to go to them"
		case "state-untested-kept":
			why = "without its tests, to say it is left"
		}
		res.Summary = fmt.Sprintf("%s: %s due, %s: not started (--input %s=list)", event, where, why, ReconcileInput)
	case block != nil && block.Stale != "":
		res.Summary = fmt.Sprintf("%s: %s back to refine: %s", event, where, block.Stale)
		keepState(res, cfg, f, spec, o, *block, "")
	case block != nil && block.Repro != "":
		res.Summary = fmt.Sprintf("%s: %s back to refine: %s", event, where, block.Repro)
		keepState(res, cfg, f, spec, o, *block, "")
	case block != nil && block.Kept != "":
		res.Summary = fmt.Sprintf("%s: %s stale, left: %s", event, where, block.Kept)
		keepState(res, cfg, f, spec, o, *block, "")
	case block != nil && block.Retest != "":
		res.Summary = fmt.Sprintf("%s: %s to its tests: %s", event, where, block.Retest)
		keepState(res, cfg, f, spec, o, *block, "")
	case block != nil && block.RetestKept != "":
		res.Summary = fmt.Sprintf("%s: %s without its tests, left: %s", event, where, block.RetestKept)
		keepState(res, cfg, f, spec, o, *block, "")
	case block != nil && block.Waits != "":
		res.Summary = fmt.Sprintf("%s: %s waits: %s", event, where, block.Waits)
		keepState(res, cfg, f, spec, o, *block, "")
	case block != nil && block.Seen != "":
		res.Summary = fmt.Sprintf("%s: %s %s", event, where, block.Seen)
		keepState(res, cfg, f, spec, o, *block, "")
	case block != nil:
		res.Status = verdict.Pass
		res.Summary = fmt.Sprintf("%s: %s blocked: %s", event, where, block.Blocked)
		keepState(res, cfg, f, spec, o, *block, "")
	case step == nil:
		res.Summary = fmt.Sprintf("%s: %s wakes no role: %s", event, where, res.Findings[len(res.Findings)-1].Message)
	case !slices.Contains(steps, step.role):
		res.Summary = fmt.Sprintf("%s: %s's step, the %s, left out by --input %s", event, where, step.role, SkipInput)
	case o.Inputs[ReconcileInput] == "list":
		// Asked only whether its step is due: the job that moved its state
		// starts the next run (ADR-0059, explicit chaining).
		res.Due = []int{o.Target.ID}
		res.Summary = fmt.Sprintf("%s: %s due, the %s: not started (--input %s=list)", event, where, step.role, ReconcileInput)
	default:
		runStateStep(res, cfg, f, spec, o, *step)
		res.Summary = fmt.Sprintf("%s: %s, the %s (%s)", event, where, step.role, res.Status)
	}
	return res
}

// runStateStep runs one issue's step, then keeps its record: never for a
// step an outside failure stopped, which the next event or reconciler
// runs again.
func runStateStep(res *Result, cfg *routing.Config, f forge.Forge, spec string, o engine.Options, st stateStep) {
	o.Role, o.Event = st.role, IssueEvent
	o.Target = &forge.Target{Kind: "issue", ID: st.write.Issue}
	if len(st.inputs) > 0 {
		in := maps.Clone(o.Inputs)
		if in == nil {
			in = map[string]string{}
		}
		maps.Copy(in, st.inputs)
		o.Inputs = in
	}
	before, steps := len(res.Findings), len(res.Steps)
	_, status := runRole(res, cfg, o, 0)
	res.Status = worst(res.Status, status)
	if status == verdict.BlockedExternal {
		return
	}
	if st.role == developerRole && status != verdict.Pass {
		// Its own run ended with no verdict read, its work lost: said on
		// the issue with the record, never as acted on (#953). Any other
		// error, after what it pushed perhaps, is not a loss.
		if why := lost(res.Steps[steps:]); why != "" {
			st.write.Lost = why
			st.write.Record.Lost = clock().UTC().Format(time.RFC3339)
			st.write.Record.Hops++
		}
	}
	if st.role == backlog.Keeper {
		// Held for its spec review: said on the issue with the record,
		// not in the job's log alone (#840).
		where := fmt.Sprintf("#%d", st.write.Issue)
		for _, f := range res.Findings[before:] {
			if f.Rule == "spec-not-reviewed" && f.Where == where {
				st.write.Holds = f.Message
			}
		}
	}
	if st.role == backlog.Keeper && st.write.Record.State == refineState && status == verdict.Pass &&
		slices.Contains(cfg.Events[IssueEvent], backlog.SpecReviewer) {
		// The spec the product owner refined, read by the reviewer at
		// once, as the night's line does after it (#128): its review
		// written, the step is due again, the product owner reading it.
		o.Role = backlog.SpecReviewer
		_, rs := runRole(res, cfg, o, 0)
		res.Status = worst(res.Status, rs)
		if rs == verdict.BlockedExternal {
			return
		}
	}
	keepState(res, cfg, f, spec, o, st.write, st.role)
}

// keepState writes w at once, or leaves it to `workline apply --line`
// when the line judges with no write token.
func keepState(res *Result, cfg *routing.Config, f forge.Forge, spec string, o engine.Options, w StateWrite, role string) {
	if res.Woke == nil {
		res.Woke = map[int]string{}
	}
	res.Woke[w.Issue] = role
	if o.NoApply {
		res.States = append(res.States, w)
		res.Forge = spec
		return
	}
	if err := writeState(f, cfg, o.Repo, w, role); err != nil {
		res.Status = worst(res.Status, verdict.Block)
		res.Findings = append(res.Findings, verdict.Finding{Rule: "state-unwritten", Where: fmt.Sprintf("#%d", w.Issue), Message: err.Error()})
	}
}

// reconcile walks every open issue, in the backlog's order, and starts
// the steps due, up to reconcile-max a run, the one waiting longest
// first (room.go): a safety net for an event lost, never the driver. With `--input reconcile=list`, it says them
// and starts none (Result.Due), for a CI job starting each one's run.
func reconcile(res *Result, cfg *routing.Config, o engine.Options) *Result {
	f, spec, err := openForge(o)
	b, ok := f.(forge.Backlog)
	if err == nil && !ok {
		err = errors.New("this forge lists no issues: nothing to reconcile")
	}
	if err != nil {
		readFailed(res, ReconcileEvent, "issues", err)
		return res
	}
	switch said, deferred := waitBudget(res, f, clock()); {
	case deferred: // the next hour's reconcile, after the reset (#804)
		res.Summary = fmt.Sprintf("%s: deferred: %s", ReconcileEvent, said)
		lateSaid(res, o.Inputs[LastInput], clock())
		return res
	case said != "":
		defer func() { res.Summary += "; " + said }()
	}
	list := o.Inputs[ReconcileInput] == "list"
	if list && o.Inputs[RoomInput] == "true" { // the steps room woke alone, for the job that freed it (room.go)
		res.Summary = ReconcileEvent + ": " + cmp.Or(roomSaid(roomDue(res, cfg, o)), "no issue waiting for room has it now")
		return res
	}
	issues, err := b.Issues()
	if err != nil {
		readFailed(res, ReconcileEvent, "issues", err)
		return res
	}
	backlog.Order(issues)
	skip := strings.Split(o.Inputs[SkipInput], ",")
	started, quiet, stales := 0, 0, 0
	x, t := newScopeIndex(o.Repo), newTallyFor(o)
	type retested struct {
		at    int // in issues
		write StateWrite
	}
	var retests []retested
	var dues []dueStep
	revisited, _ := revisits(cfg)
	for i, is := range issues {
		s, name, _ := cfg.Wakes(is.Labels, IssueEvent)
		if revisited && s.Name == BlockedState && name == "" {
			name = backlog.Keeper // read again when there is news (revisit.go)
		}
		left := name == "" || o.Inputs[SkipInput] != "" && slices.Contains(skip, name)
		if left && cfg.StaleDays(s.Name) == 0 {
			continue // no step of this project's, or left out: said by `route issue`, not by every reconcile
		}
		where := fmt.Sprintf("#%d", is.ID)
		var step *stateStep
		var block *StateWrite
		var err error
		if left { // its staleness alone checked
			block, err = stale(res, cfg, b, x, is, s)
		} else {
			step, block, err = stateOf(res, cfg, b, is, &issues, x, t)
		}
		if errors.Is(err, forge.ErrUnreachable) {
			readFailed(res, ReconcileEvent, where, err)
			return res
		}
		if err != nil { // that issue alone: the others are still walked
			res.Status = worst(res.Status, verdict.Block)
			res.Findings = append(res.Findings, verdict.Finding{Rule: "state-unread", Where: where, Message: err.Error()})
			continue
		}
		if step == nil && block == nil {
			if !left {
				quiet++
			}
			continue
		}
		if block != nil && block.Stale != "" {
			anticipate(cfg, &issues[i], refineState) // out of its state's count for the steps after it
		}
		if block != nil && block.Retest != "" {
			retests = append(retests, retested{i, *block})
			continue
		}
		if block != nil && (block.Waits != "" || block.Stale != "" || block.Kept != "" || block.Seen != "" || block.RetestKept != "" || block.Repro != "") { // no agent, no step: outside reconcile-max
			if list {
				res.Due = append(res.Due, is.ID)
			} else {
				keepState(res, cfg, f, spec, o, *block, "")
			}
			if block.Stale != "" && !list {
				stales++
			}
			continue
		}
		dues = append(dues, dueStep{i, step, block})
	}
	startDue(res, cfg, f, spec, o, b, issues, dues, t, &started, list)
	// A move to its tests asks no agent, but wakes the tester at once:
	// counted in reconcile-max after the steps due, the ready issues from
	// before the line go to their tests a few a run, never a flood of
	// agent calls, never ahead of an issue whose step is due.
	for _, r := range retests {
		if started >= cfg.ReconcileMax {
			res.Findings = append(res.Findings, verdict.Finding{Rule: "reconcile-left", Level: "info", Where: fmt.Sprintf("#%d", r.write.Issue),
				Message: fmt.Sprintf("to go to its tests, past reconcile-max (%d) this run: the next run moves it", cfg.ReconcileMax)})
			continue
		}
		started++
		if s, ok := cfg.StateOf(issues[r.at].Labels); ok {
			if to, ok := testsState(cfg, s); ok {
				anticipate(cfg, &issues[r.at], to.Name)
			}
		}
		if list {
			res.Due = append(res.Due, r.write.Issue)
		} else {
			keepState(res, cfg, f, spec, o, r.write, "")
		}
	}
	issued := started
	rounds := roundsDue(res, cfg, f, o, &issued)
	switch {
	case list:
		res.Summary = fmt.Sprintf("%s: %d %s due:", ReconcileEvent, len(res.Due), plural(len(res.Due), "issue", "issues"))
		for _, n := range res.Due {
			res.Summary += " #" + strconv.Itoa(n)
		}
		res.Reviews = rounds
	case started == 0:
		res.Summary = fmt.Sprintf("%s: nothing due (%d %s on the line acted on already)", ReconcileEvent, quiet, plural(quiet, "issue", "issues"))
	default:
		res.Summary = fmt.Sprintf("%s: %d %s due, %s run (%s)", ReconcileEvent, started, plural(started, "issue", "issues"), plural(started, "its step", "their steps"), res.Status)
	}
	if stales > 0 {
		res.Summary += fmt.Sprintf("; %d stale %s back to refine", stales, plural(stales, "issue", "issues"))
	}
	if len(rounds) > 0 {
		res.Summary += fmt.Sprintf("; %d review %s due:", len(rounds), plural(len(rounds), "round", "rounds"))
		for _, r := range rounds {
			res.Summary += fmt.Sprintf(" merge request %d", r.ID)
		}
	}
	lateSaid(res, o.Inputs[LastInput], clock())
	return res
}

// roundsDue are the developer's merge requests a review round is due on
// that no event started: the reviewer's verdict, a gate's report or a
// person's comment its last round did not read (ADR-0054, step 6). Read
// only where the project routes the developer; counted in reconcile-max
// with the issues' steps (started). In CI (list), its job starts each
// round's own run; on a machine, each is said with its command: a round
// works on its branch, checked out.
func roundsDue(res *Result, cfg *routing.Config, f forge.Forge, o engine.Options, started *int) []Review {
	if !cfg.Runs("developer") {
		return nil
	}
	mrs, err := f.ListMergeRequests()
	if err != nil {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "rounds-unread", Level: "warn", Where: "merge requests",
			Message: "the developer's pull requests not read, so no review round an event missed is started: " + forge.Said(err)})
		return nil
	}
	var due []Review
	for _, mr := range mrs {
		if !strings.HasPrefix(mr.Branch, developer.Branch) {
			continue
		}
		if !roundDue(res, f, o.Repo, mr) {
			continue
		}
		if *started >= cfg.ReconcileMax {
			res.Findings = append(res.Findings, verdict.Finding{Rule: "reconcile-left", Level: "info", Where: fmt.Sprintf("merge request %d", mr.ID),
				Message: fmt.Sprintf("a review round due, past reconcile-max (%d) this run: the next run starts it", cfg.ReconcileMax)})
			continue
		}
		*started++
		due = append(due, Review{ID: mr.ID, Branch: mr.Branch})
		if o.Inputs[ReconcileInput] != "list" {
			roundSaid(res, mr)
		}
	}
	return due
}

// lateSaid adds to res how late this reconcile is, when the last one, at
// last (RFC 3339), was over ReconcileLate ago: a step an event missed
// waited that long (#789). Nothing when last is empty or unreadable.
func lateSaid(res *Result, last string, now time.Time) {
	t, err := time.Parse(time.RFC3339, last)
	if err != nil {
		return
	}
	gap := now.Sub(t)
	if gap <= ReconcileLate {
		return
	}
	said := fmt.Sprintf("%d h %02d min since the last reconcile (%s), over %d h: a step an event missed waited that long",
		int(gap.Hours()), int(gap.Minutes())%60, t.UTC().Format("2006-01-02 15:04 UTC"), int(ReconcileLate.Hours()))
	res.Summary += "; late: " + said
	res.Findings = append(res.Findings, verdict.Finding{Rule: "reconcile-late", Level: "warn", Message: said})
}

// lost is why the developer's own run, the first of steps, ended with no
// verdict read (engine.ErrNoVerdict): "" when it did not; a handoff's
// run, or another error, is never a loss.
func lost(steps []Step) string {
	if len(steps) == 0 || steps[0].Result == nil || !steps[0].Result.NoVerdict {
		return ""
	}
	for _, f := range steps[0].Result.Findings {
		if f.Rule == "engine-error" {
			return f.Message
		}
	}
	return engine.ErrNoVerdict.Error()
}
