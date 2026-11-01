package line

// The line of states (ADR-0059, decision 8): an event naming an issue
// runs the role its state wakes, read from the routing's table by a pure
// function (routing.Config.Wakes), not every step of the event; once,
// for a state and body the line's record does not hold yet. The
// reconciler walks every open issue for a step due that no event
// started.

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/backlog"
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
}

// stateOf reads where issue is on the line and whether its step is due:
// the step to run, else nil, what the line says of it added to res. A
// step past the cap is not run: the issue is to be blocked (write); nor
// one whose next state is full: it waits (write, once). open: the open
// issues, listed at first need.
func stateOf(res *Result, cfg *routing.Config, b forge.Backlog, is forge.Issue, open *[]forge.Issue) (*stateStep, *StateWrite, error) {
	where := fmt.Sprintf("#%d", is.ID)
	s, name, why := cfg.Wakes(is.Labels, IssueEvent)
	if name == "" {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "state-wakes-none", Level: "info", Where: where, Message: why})
		return nil, nil, nil
	}
	rec, had, spec, refused, err := readRecordSpec(b, is.ID)
	if err != nil {
		return nil, nil, err
	}
	t, err := b.Trail(is.ID, s.Label)
	if err != nil {
		return nil, nil, err
	}
	if s.Name != refineState {
		spec = "" // a review is read in refine alone: no other state woken by it
	}
	now, ok := due(s, is, t.Labeled, spec, rec, had)
	if !ok && s.Role == "tester" && refused != "" {
		// The tester's run on it refused, its state the same: never left
		// as acted, which nothing would wake again (#794). Blocked, why
		// on the board; a person moves it back.
		why := "the tester wrote no tests for it in its state " + s.Name + ": " + refused
		res.Findings = append(res.Findings, verdict.Finding{Rule: "tests-refused", Level: "warn", Where: where, Message: why + ": moved to " + BlockedState})
		now.State = BlockedState
		return nil, &StateWrite{Issue: is.ID, Record: now, Blocked: why}, nil
	}
	if !ok {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "state-acted", Level: "info", Where: where,
			Message: fmt.Sprintf("its state, %s, woke the %s already, on the same body and labelling: nothing more to run", s.Name, name)})
		return nil, nil, nil
	}
	if cfg.MaxHops > 0 && now.Hops > cfg.MaxHops {
		why := fmt.Sprintf("%d state steps on this issue, past max-hops (%d): roles sent it back and forth; its state was %s",
			now.Hops, cfg.MaxHops, s.Name)
		res.Findings = append(res.Findings, verdict.Finding{Rule: "hops-capped", Level: "warn", Where: where, Message: why + ": moved to " + BlockedState})
		now.State = BlockedState
		return nil, &StateWrite{Issue: is.ID, Record: now, Blocked: why}, nil
	}
	if why, err := waits(cfg, b, s, open); err != nil {
		return nil, nil, err
	} else if why != "" {
		return nil, waiting(res, s, why, rec, where, is.ID), nil
	}
	res.Findings = append(res.Findings, verdict.Finding{Rule: "state-wakes", Level: "info", Where: where,
		Message: fmt.Sprintf("its state, %s, wakes the %s", s.Name, name)})
	return &stateStep{role: name, next: s.Next, write: StateWrite{Issue: is.ID, Record: now}}, nil, nil
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
		switch said, deferred := waitBudget(res, f, time.Now()); {
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
	step, block, err := stateOf(res, cfg, b, *is, new([]forge.Issue))
	if err != nil {
		readFailed(res, event, where, err)
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
		case "state-waits":
			why = "to say it waits for room"
		}
		res.Summary = fmt.Sprintf("%s: %s due, %s: not started (--input %s=list)", event, where, why, ReconcileInput)
	case block != nil && block.Waits != "":
		res.Summary = fmt.Sprintf("%s: %s waits: %s", event, where, block.Waits)
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
	_, status := runRole(res, cfg, o, 0)
	res.Status = worst(res.Status, status)
	if status == verdict.BlockedExternal {
		return
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
// the steps due, up to reconcile-max a run: a safety net for an event
// lost, never the driver. With `--input reconcile=list`, it says them
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
	switch said, deferred := waitBudget(res, f, time.Now()); {
	case deferred: // the next hour's reconcile, after the reset (#804)
		res.Summary = fmt.Sprintf("%s: deferred: %s", ReconcileEvent, said)
		lateSaid(res, o.Inputs[LastInput], time.Now())
		return res
	case said != "":
		defer func() { res.Summary += "; " + said }()
	}
	issues, err := b.Issues()
	if err != nil {
		readFailed(res, ReconcileEvent, "issues", err)
		return res
	}
	backlog.Order(issues)
	list := o.Inputs[ReconcileInput] == "list"
	skip := strings.Split(o.Inputs[SkipInput], ",")
	started, quiet := 0, 0
	for i, is := range issues {
		_, name, _ := cfg.Wakes(is.Labels, IssueEvent)
		if name == "" || o.Inputs[SkipInput] != "" && slices.Contains(skip, name) {
			continue // no step of this project's, or left out: said by `route issue`, not by every reconcile
		}
		where := fmt.Sprintf("#%d", is.ID)
		step, block, err := stateOf(res, cfg, b, is, &issues)
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
			quiet++
			continue
		}
		if block != nil && block.Waits != "" { // no agent, no step: outside reconcile-max
			if list {
				res.Due = append(res.Due, is.ID)
			} else {
				keepState(res, cfg, f, spec, o, *block, "")
			}
			continue
		}
		if started >= cfg.ReconcileMax {
			res.Findings = append(res.Findings, verdict.Finding{Rule: "reconcile-left", Level: "info", Where: where,
				Message: fmt.Sprintf("due, past reconcile-max (%d) this run: the next run starts it", cfg.ReconcileMax)})
			continue
		}
		started++
		if step != nil {
			anticipate(cfg, &issues[i], step.next) // the steps after it see it in its next state
		}
		switch {
		case list: // its own run blocks it, past the cap: nothing written here
			res.Due = append(res.Due, is.ID)
		case block != nil:
			keepState(res, cfg, f, spec, o, *block, "")
		default:
			runStateStep(res, cfg, f, spec, o, *step)
		}
	}
	switch {
	case list:
		res.Summary = fmt.Sprintf("%s: %d %s due:", ReconcileEvent, len(res.Due), plural(len(res.Due), "issue", "issues"))
		for _, n := range res.Due {
			res.Summary += " #" + strconv.Itoa(n)
		}
	case started == 0:
		res.Summary = fmt.Sprintf("%s: nothing due (%d %s on the line acted on already)", ReconcileEvent, quiet, plural(quiet, "issue", "issues"))
	default:
		res.Summary = fmt.Sprintf("%s: %d %s due, %s run (%s)", ReconcileEvent, started, plural(started, "issue", "issues"), plural(started, "its step", "their steps"), res.Status)
	}
	lateSaid(res, o.Inputs[LastInput], time.Now())
	return res
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
