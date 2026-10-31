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
	write StateWrite
}

// stateOf reads where issue is on the line and whether its step is due:
// the step to run, else nil, what the line says of it added to res. A
// step past the cap is not run: the issue is to be blocked (write).
func stateOf(res *Result, cfg *routing.Config, b forge.Backlog, is forge.Issue) (*stateStep, *StateWrite, error) {
	where := fmt.Sprintf("#%d", is.ID)
	s, name, why := cfg.Wakes(is.Labels, IssueEvent)
	if name == "" {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "state-wakes-none", Level: "info", Where: where, Message: why})
		return nil, nil, nil
	}
	rec, had, err := readRecord(b, is.ID)
	if err != nil {
		return nil, nil, err
	}
	t, err := b.Trail(is.ID, s.Label)
	if err != nil {
		return nil, nil, err
	}
	now, ok := due(s, is, t.Labeled, rec, had)
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
	res.Findings = append(res.Findings, verdict.Finding{Rule: "state-wakes", Level: "info", Where: where,
		Message: fmt.Sprintf("its state, %s, wakes the %s", s.Name, name)})
	return &stateStep{role: name, write: StateWrite{Issue: is.ID, Record: now}}, nil, nil
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
	step, block, err := stateOf(res, cfg, b, *is)
	if err != nil {
		readFailed(res, event, where, err)
		return res
	}
	switch {
	case block != nil && o.Inputs[ReconcileInput] == "list":
		// Asked only: its own run blocks it, nothing written here.
		res.Due = []int{o.Target.ID}
		res.Summary = fmt.Sprintf("%s: %s due, past its hop cap: not started (--input %s=list)", event, where, ReconcileInput)
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
	if err := writeState(f, cfg, w, role); err != nil {
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
	issues, err := b.Issues()
	if err != nil {
		readFailed(res, ReconcileEvent, "issues", err)
		return res
	}
	backlog.Order(issues)
	list := o.Inputs[ReconcileInput] == "list"
	skip := strings.Split(o.Inputs[SkipInput], ",")
	started, quiet := 0, 0
	for _, is := range issues {
		_, name, _ := cfg.Wakes(is.Labels, IssueEvent)
		if name == "" || o.Inputs[SkipInput] != "" && slices.Contains(skip, name) {
			continue // no step of this project's, or left out: said by `route issue`, not by every reconcile
		}
		where := fmt.Sprintf("#%d", is.ID)
		step, block, err := stateOf(res, cfg, b, is)
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
		if started >= cfg.ReconcileMax {
			res.Findings = append(res.Findings, verdict.Finding{Rule: "reconcile-left", Level: "info", Where: where,
				Message: fmt.Sprintf("due, past reconcile-max (%d) this run: the next run starts it", cfg.ReconcileMax)})
			continue
		}
		started++
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
	return res
}
