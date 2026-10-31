// Package line runs the steps routing names for an event, in order, and the
// handoffs they ask for (docs/spec/routing.md).
package line

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/gate"
	"github.com/JN0V/workline/internal/gitrange"
	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/verdict"
)

// Step is one role or gate that ran.
type Step struct {
	Name   string           `json:"name"`
	Status string           `json:"status"`
	Result *engine.Result   `json:"result,omitempty"`
	Gate   *verdict.Verdict `json:"gate,omitempty"`
	// New: what a gate found the change adds, for what shows it on the
	// merge request; on an event given a range, the gate run here.
	New *gate.Added `json:"new,omitempty"`
	// From: the file a gate's verdict was read from, the job that ran it
	// having shown its findings; empty when it ran here.
	From string `json:"from,omitempty"`
	// Seconds: how long the step took, in tenths; a gate read from the
	// job that ran it, that job's (ADR-0050).
	Seconds float64 `json:"seconds"`
}

// Result is the outcome of an event on the line.
type Result struct {
	Status     string            `json:"status"`
	Summary    string            `json:"summary,omitempty"`
	Steps      []Step            `json:"steps"`
	Findings   []verdict.Finding `json:"findings,omitempty"` // every step's
	AgentCalls int               `json:"agent-calls"`
	Calls      []agent.Call      `json:"calls,omitempty"` // every step's
	// Pending lists the runs judged with NoApply, in the line's order: what
	// `workline apply` is given in the job that holds the write token.
	Pending []string `json:"pending,omitempty"`
	// Range: the range the gates alone ran on (Gates.Only); Commits, its
	// ends as the commits they were when the gates ran, which the line
	// reading their verdicts checks it judges too.
	Range   string `json:"range,omitempty"`
	Commits string `json:"commits,omitempty"`
	// Seconds: how long the line took, the gates' job's seconds added when
	// their verdicts were read from it: the verdict waited for that job.
	// Moment: the wait it is part of and its budget (ADR-0050); none on an
	// event no person waits on.
	Seconds float64 `json:"seconds"`
	Moment  *Moment `json:"moment,omitempty"`
	// Event: the event the line ran; Main, on main after a merge, what
	// it compared (MainEvent).
	Event string   `json:"event,omitempty"`
	Main  *MainRun `json:"main,omitempty"`
	// DryRun: a release's dry run, what would hold the release were it
	// cut now (ADR-0050).
	DryRun *DryRun `json:"dry-run,omitempty"`
	// States: on the line of states (ADR-0059), what the line writes on
	// each issue it ran a step for, left to `workline apply --line` when
	// it judged with no write token; Woke, the role each issue's step
	// ran; Forge, where. Due: the issues the reconciler found due, with
	// `--input reconcile=list`.
	States []StateWrite   `json:"states,omitempty"`
	Woke   map[int]string `json:"woke,omitempty"`
	Forge  string         `json:"forge,omitempty"`
	Due    []int          `json:"due,omitempty"`
}

// Gates says how a line takes its gate steps: run there, by default. In
// CI, a job of their own runs them, with no write token and no agent's
// (Only), and the job judging the line reads their verdicts (From): the
// gates' tools run the change's code, a linter's config or plugin
// (ADR-0049, section 6).
type Gates struct {
	// Only: the event's gate steps alone, each one, for a job that runs
	// nothing else.
	Only bool
	// From: the file holding the result a line given Only printed; each
	// gate step's verdict is read there, never run.
	From string
	// Kept: the folder of the gates' findings kept by commit (ADR-0050):
	// on main after a merge, HEAD's are written there, the next base; on
	// a change, its merge base's are read there, when they stand for a
	// run, rather than running the tools again.
	Kept string

	read *Result // From, once read
	err  error
}

// MainEvent is the line after a merge, on the default branch's new tip
// (ADR-0050): the merge request's gates first, every finding kept as the
// next merge requests' base, then the event's own steps, each one.
const MainEvent = "main"

// SkipInput names, comma-separated, the event's steps a run leaves out:
// workline's own schedule, its docs weekly and its backlog nightly.
const SkipInput = "skip"

// MainRun is what main's run after a merge compared: main's tip before
// it, "" when none is known, and the head; the commits between them, as
// `git log --first-parent --oneline` says them, for whoever reads why
// main changed.
type MainRun struct {
	Before  string   `json:"before,omitempty"`
	Head    string   `json:"head"`
	Commits []string `json:"commits,omitempty"`
}

// Run runs event's steps. base carries what every role run shares (repository,
// roles, agent, inputs, forge, target, scope, NoApply); Role and Event are set
// per step. The first step that does not pass stops the line: it fails
// closed. On an event the routing sets `fail-fast: false` for (schedule, by
// default), every step runs all the same and the line takes the worst
// verdict (ADR-0037). With NoApply, each step judges the tree as it is — no
// step sees what an earlier one proposed — and handoffs wait, since nothing
// is applied yet.
func Run(event string, base engine.Options) *Result { return RunWith(event, base, &Gates{}) }

// RunWith is Run, its gate steps taken as gates says.
func RunWith(event string, base engine.Options, gates *Gates) *Result {
	start := time.Now()
	res := &Result{Status: verdict.Pass, Steps: []Step{}, Event: event}
	// A release's dry run (ADR-0050): every step, no agent, nothing
	// applied; what blocks said as what would hold the release. Registered
	// first, it runs last, once the line said all it says.
	if base.DryRun {
		var dry *DryRun
		base, dry = dryRunOn(base)
		defer func() { finishDryRun(res, dry) }()
		if dry.At != "" {
			return res
		}
	}
	cfg, err := routing.Load(base.Repo)
	// Each step's seconds, and the moment's budget (ADR-0050): said, never
	// a block.
	defer func() {
		res.Seconds = Since(start)
		for _, s := range res.Steps {
			if s.From != "" {
				res.Seconds = round(res.Seconds + s.Seconds)
			}
		}
		if cfg != nil {
			Measure(res, cfg, momentOf(event, base.Inputs, gates.Only, res.AgentCalls))
		}
	}()
	if err != nil {
		f := failed(res, "routing", err.Error())
		if role.IsConfigError(err) {
			f.Findings[len(f.Findings)-1].Rule = "config-invalid"
		}
		return f
	}
	// The reconciler (ADR-0059): every open issue's step due, its steps
	// those the project routes on issue.
	if event == ReconcileEvent {
		return reconcile(res, cfg, base)
	}
	steps, ok := stepsOf(cfg, event)
	if !ok {
		return failed(res, "routing", fmt.Sprintf("routing names no step for event %q", event))
	}
	// `--input skip=<role>,…`: the line leaves those steps out, as a
	// schedule running some roles weekly and others nightly does (ADR-0051).
	if skip := strings.Split(base.Inputs[SkipInput], ","); base.Inputs[SkipInput] != "" {
		steps = slices.DeleteFunc(slices.Clone(steps), func(s string) bool { return slices.Contains(skip, s) })
	}
	// GitLab's webhook on its own trigger URL: the event names the issue
	// in $TRIGGER_PAYLOAD (ADR-0059); one that put no state's label on an
	// issue runs nothing, at once.
	if raw := os.Getenv(PayloadEnv); event == IssueEvent && base.Target == nil && raw != "" && len(cfg.States) > 0 {
		n, why, err := readPayload(raw, cfg)
		if err != nil {
			return failed(res, "payload", err.Error())
		}
		if n == 0 {
			res.Summary = event + ": " + why
			return res
		}
		base.Target = &forge.Target{Kind: "issue", ID: n}
	}
	// An event naming an issue runs the role its state wakes (ADR-0059),
	// not every step of the event; one waking none runs none, said.
	if event == IssueEvent && base.Target != nil && base.Target.Kind == "issue" && len(cfg.States) > 0 {
		return byState(res, cfg, base, event, steps)
	}
	// Main after a merge: its gates compare HEAD with main's tip before
	// the merge, whoever made it; the engine's own commits there are
	// checked as any other.
	gatesRange := base.Inputs["range"]
	if event == MainEvent {
		res.Main, gatesRange = mainRun(base.Repo, gatesRange)
		defer func() {
			if gates.Kept != "" && slices.ContainsFunc(res.Steps, func(s Step) bool { return strings.HasPrefix(s.Name, "gate:") && s.From == "" }) {
				res.Summary += "; the gates' findings kept as the next base"
			}
		}()
	}
	// What the engine committed does not wake the line on itself (routing
	// spec) — a role's fix pushed to a merge request would be judged, and
	// pushed again — nor turns a verdict green unjudged: the range is
	// judged as it stood before the engine's last commits, a merge of the
	// base through its first parent (ADR-0041). The person's commits keep
	// their verdict; what was judged already is not asked of an agent again.
	// The committer alone checks every commit, workline's own too:
	// identity, subject, secrets cost no agent and are never skipped.
	own, whole, nothingElse := "", base.Inputs, false
	if role, rng := beforeOwn(base.Repo, base.Inputs["range"]); role != "" && event != MainEvent {
		inputs := maps.Clone(base.Inputs)
		inputs["range"] = rng
		base.Inputs = inputs
		own = fmt.Sprintf(" (the last commit workline's own, %s: judged as %s)", role, rng)
		if nothingElse = rng == ""; nothingElse {
			own = fmt.Sprintf(" (the last commits workline's own, %s: the committer checked them, nothing else to judge)", role)
		}
	}
	defer func() { res.Summary += own }()
	// The run apart from the verdict (`warn=only`, ADR-0050): only the
	// steps whose role reads what only warns apart on the event, never a
	// gate; its verdict is never the head's.
	warnOnly := base.Inputs[WarnInput] == WarnOnly
	// A merge queue's or train's run (`queue=true`, ADR-0050): the gates
	// alone, on the merged result, never an agent; the steps it leaves are
	// named, their verdict the head's, as the person's yes (ADR-0042).
	queue := base.Inputs[QueueInput] == "true"
	if queue {
		var left []string
		for _, name := range steps {
			if !strings.HasPrefix(name, "gate:") && !slices.Contains(left, name) {
				left = append(left, name)
			}
		}
		defer func() { res.Summary += "; " + queueSaid(len(res.Steps), left) }()
	}
	// The gates alone: each one runs, so that each is shown.
	stopAtFirst := cfg.StopsAtFirst(event) && !gates.Only && !base.DryRun
	if gates.Only {
		res.Range = gatesRange
		res.Commits, _ = commitsOf(base.Repo, gatesRange)
	}
	// What holds the release on the forge (ADR-0051): read once, beside
	// the steps, stopping none of them; the line's verdict takes it.
	if !gates.Only && !warnOnly && !queue {
		if found, held := releaseHolds(event, base); len(found) > 0 {
			res.Findings = append(res.Findings, found...)
			defer func() {
				// Checks not done there, and nothing else holding it: read
				// again once the steps ran, till they end (#640).
				if again, h, waited := waitHolds(event, base, found); waited {
					res.Findings = slices.DeleteFunc(res.Findings, func(f verdict.Finding) bool { return slices.Contains(holdRules, f.Rule) })
					res.Findings = append(res.Findings, again...)
					found, held = again, h
				}
				status := verdict.Pass
				for _, f := range found {
					if f.Level == "" || f.Level == verdict.Block {
						status = verdict.Block
					}
				}
				res.Status = worst(res.Status, status)
				if held != "" {
					res.Summary += "; the release held: " + held
				}
			}()
		}
	}
	var notPassed []string // with every step run: those that did not pass, and how
	considered := 0
	// The gates' findings, for the role steps after them that read them
	// (the inspector, ADR-0049): the gates' job's result when read from
	// it, else the gate steps run here, written for them. The run apart
	// runs no gate: it reads that job's result, when given, for the
	// commits it judges.
	found := &foundFor{}
	defer found.remove()
	if warnOnly && gates.From != "" {
		found.read(res, base.Repo, gates, gatesRange)
	}
	for _, name := range steps {
		step, status := name, verdict.Pass
		g, isGate := strings.CutPrefix(name, "gate:")
		skip := nothingElse && name != committerRole
		if queue { // the gates on the merged result, whoever made its last commit
			skip = !isGate
		}
		if skip || gates.Only && !isGate || warnOnly && (isGate || !apartOn(base.RolesDir, name, event)) {
			continue
		}
		// A routing written before ADR-0051 still names the documentalist
		// on release: the step is skipped and said, never a release held.
		if r, err := role.Load(base.RolesDir, name); event == "release" && !isGate && err == nil && !r.Accepts(event) {
			res.Findings = append(res.Findings, verdict.Finding{Rule: "step-skipped", Level: "warn", Where: name,
				Message: fmt.Sprintf("the role %s does not run on release: step skipped; remove it from routing.events.release (ADR-0051)", name)})
			continue
		}
		considered++
		if isGate {
			s := gates.step(base.Repo, g, event, gatesRange)
			res.Steps = append(res.Steps, s)
			res.Findings = append(res.Findings, s.Gate.Findings...)
			status = s.Status
			found.add(s, gates, base.Repo, gatesRange)
		} else {
			o := base
			if name == committerRole {
				o.Inputs = whole
			}
			if file := found.file(res, base.Repo, gatesRange); file != "" {
				inputs := maps.Clone(o.Inputs)
				if inputs == nil {
					inputs = map[string]string{}
				}
				inputs[GatesInput] = file
				o.Inputs = inputs
			}
			// What only warns runs apart where the role reads it apart on
			// the event; any other step runs whole, never told to leave it.
			if _, told := o.Inputs[WarnInput]; told && !apartOn(base.RolesDir, name, event) {
				o.Inputs = maps.Clone(o.Inputs)
				delete(o.Inputs, WarnInput)
			}
			o.Role, o.Event = name, event
			step, status = runRole(res, cfg, o, 0)
		}
		if status == verdict.Pass {
			continue
		}
		if stopAtFirst {
			return stopped(res, step, status)
		}
		notPassed = append(notPassed, fmt.Sprintf("%s (%s)", step, status))
		res.Status = worst(res.Status, status)
	}
	if len(notPassed) > 0 {
		res.Summary = fmt.Sprintf("%s: every step ran; %d of %d did not pass: %s", event, len(notPassed), considered, strings.Join(notPassed, ", "))
	} else {
		res.Summary = fmt.Sprintf("%s: %d steps passed", event, len(res.Steps))
	}
	if gates.Only {
		res.Summary += "; the gates alone"
	}
	if warnOnly {
		res.Summary += "; what only warns, apart from the verdict"
		if considered == 0 {
			res.Summary = fmt.Sprintf("%s: nothing reads apart from the verdict", event)
		}
	}
	waits := func(why string) bool {
		return slices.ContainsFunc(res.Steps, func(s Step) bool { return s.Result != nil && s.Result.AIWaits == why })
	}
	if waits(engine.AIWaitsReady) {
		res.Summary += "; a draft: no agent asked, the AI review waits for it to be marked ready"
	}
	if len(res.Pending) > 0 {
		res.Summary += fmt.Sprintf("; %d runs to apply", len(res.Pending))
	}
	return res
}

// committerRole is the step that checks every commit of a range, workline's
// own included (beforeOwn leaves them to it).
const committerRole = "committer"

// WarnInput is the input telling a role reading what only warns apart
// from the verdict (its role.yaml's `apart`, ADR-0050) which run this is:
// WarnApart, the verdict's, which leaves it; WarnOnly, the run apart,
// which reads only it. Given to every step, as every input is.
const (
	WarnInput = "warn"
	WarnApart = "apart"
	WarnOnly  = "only"
)

// GatesInput is the input giving a role step the gates' findings before
// it in the line: a file holding a line's result as `--gates-only --json`
// prints it, each gate step's new findings in it (Step.New). The
// inspector triages them (ADR-0049); a role that reads none ignores it.
const GatesInput = "gates"

// foundFor is what a line hands its role steps of the gates before them
// (GatesInput): the gates' job's result, its path, when the line reads
// their verdicts there; else the gate steps run here, written to a file
// of the git directory once a role step comes after one more of them,
// removed once the line ends.
type foundFor struct {
	from    string // the gates' job's result, read for the commits judged
	steps   []Step // the gate steps run here
	written int    // how many of them the file holds
	path    string // "" when none was written
	said    bool   // a file not written, said once
}

// read takes the gates' job's result for the run apart, which runs no
// gate: only for the commits the line judges, else said, none given.
func (f *foundFor) read(res *Result, repo string, g *Gates, rng string) {
	if g.read == nil && g.err == nil {
		g.read, g.err = readResult(g.From)
	}
	why := ""
	switch {
	case g.err != nil:
		why = fmt.Sprintf("the gates' job left no result to read (%v)", g.err)
	case !g.sameCommits(repo, rng):
		why = fmt.Sprintf("the gates ran on %q, this run judges %q", g.read.Range, rng)
	}
	if why != "" {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "gates-unread", Level: "warn", Where: "gates",
			Message: why + ": their findings are given to no step of this run"})
		return
	}
	f.from = g.From
}

// add keeps a gate step for the role steps after it. One read from the
// gates' job stands for that job's whole result, when it is for the
// commits the line judges: else the step is a check-error, and no
// finding of it is given.
func (f *foundFor) add(s Step, g *Gates, repo, rng string) {
	if s.From == "" {
		f.steps = append(f.steps, s)
		return
	}
	if f.from == "" && g.err == nil && g.read != nil && g.sameCommits(repo, rng) {
		f.from = s.From
	}
}

// file is the gates' findings for a role step, "" when no gate came
// before it: the gates' job's result, or the gate steps run here, written
// again when one more ran since. A file that cannot be written gives
// none, said once: the role says it was given none.
func (f *foundFor) file(res *Result, repo, rng string) string {
	if f.from != "" {
		return f.from
	}
	if len(f.steps) == 0 {
		return ""
	}
	if f.written == len(f.steps) {
		return f.path
	}
	dir := filepath.Join(repo, ".git", "workline", "gates")
	if gd, err := exec.Command("git", "-C", repo, "rev-parse", "--absolute-git-dir").Output(); err == nil {
		dir = filepath.Join(strings.TrimSpace(string(gd)), "workline", "gates")
	}
	r := Result{Status: verdict.Pass, Steps: f.steps, Range: rng}
	r.Commits, _ = commitsOf(repo, rng)
	data, err := json.MarshalIndent(r, "", "  ")
	if err == nil {
		err = os.MkdirAll(dir, 0o755)
	}
	path := filepath.Join(dir, time.Now().UTC().Format("20060102T150405.000000000")+"-line.json")
	if err == nil {
		err = os.WriteFile(path, data, 0o644)
	}
	f.remove()
	f.written = len(f.steps)
	if err != nil {
		if !f.said {
			f.said = true
			res.Findings = append(res.Findings, verdict.Finding{Rule: "gates-not-written", Level: "warn", Where: "gates",
				Message: fmt.Sprintf("the gates' findings could not be written for the steps after them (%v): they are given to none", err)})
		}
		return ""
	}
	f.path = path
	return path
}

// remove takes away the file written for the line's role steps: they
// read it in their first pass, and keep what they read.
func (f *foundFor) remove() {
	if f.path != "" {
		os.Remove(f.path)
		f.path = ""
	}
}

// QueueInput is the input saying the run is a merge queue's or train's,
// on the merged result, `true` (ADR-0050): the line runs its gates
// alone, no agent asked; the head's verdict and the person's yes stay
// the merge request's (ADR-0042), so a queue's run writes none.
const QueueInput = "queue"

// queueSaid is what a queue's run says of itself: the gates it ran on
// the merged result, else that it had none, and the steps it left to
// the head's verdict.
func queueSaid(gates int, left []string) string {
	said := "a merge queue's or train's run: the gates alone, on the merged result"
	if gates == 0 {
		said = "a merge queue's or train's run: no gate in the line, nothing checked on the merged result"
	}
	if len(left) > 0 {
		said += "; not run, the head's verdict holds them: " + strings.Join(left, ", ")
	}
	return said
}

// apartOn says whether the role a step names reads what only warns apart
// on event; a role that does not load reads nothing apart: the verdict's
// line, which runs it, says why.
func apartOn(rolesDir, name, event string) bool {
	r, err := role.Load(rolesDir, name)
	return err == nil && r.ApartOn(event)
}

// step is a gate step: run here, or its verdict read from the job that
// ran it. One not there is an error, never a pass: that job did not run
// it, or ended before writing.
func (g *Gates) step(repo, name, event, rng string) Step {
	if g.From == "" {
		start := time.Now()
		v, added := runGate(repo, name, event, rng, g.Kept)
		return Step{Name: "gate:" + name, Status: v.Status, Gate: v, New: added, Seconds: Since(start)}
	}
	if g.read == nil && g.err == nil {
		g.read, g.err = readResult(g.From)
	}
	not := func(why string) Step {
		return Step{Name: "gate:" + name, Status: verdict.Block, From: g.From, Gate: &verdict.Verdict{Status: verdict.Block,
			Summary:  fmt.Sprintf("gate %s: no verdict", name),
			Findings: []verdict.Finding{{Rule: "check-error", Where: name, Message: why}}}}
	}
	switch {
	case g.err != nil:
		return not(fmt.Sprintf("the gates' job left no result to read (%v): it did not run, or ended before writing it", g.err))
	case !g.sameCommits(repo, rng):
		return not(fmt.Sprintf("the gates ran on %q, this line judges %q", g.read.Range, rng))
	}
	for _, s := range g.read.Steps {
		if s.Name == "gate:"+name && s.Gate != nil {
			return Step{Name: s.Name, Status: s.Status, Gate: s.Gate, From: g.From, Seconds: s.Seconds}
		}
	}
	return not(fmt.Sprintf("the gates' job, %s, holds no verdict of it", g.From))
}

// sameCommits says whether the gates ran on the commits the line judges:
// their range's ends as they were when the gates ran, against this one's
// now, however each spells them — a branch's name, its commit. A name
// moved in between is other commits. A result naming no commits is read
// by its range's words alone.
func (g *Gates) sameCommits(repo, rng string) bool {
	if g.read.Commits == "" {
		return g.read.Range == rng
	}
	now, ok := commitsOf(repo, rng)
	return ok && now == g.read.Commits
}

// commitsOf is a range's ends as commits, base..head; false when one is
// no commit here.
func commitsOf(repo, rng string) (string, bool) {
	var out []string
	for _, rev := range []string{gitrange.Base(rng), gitrange.Head(rng)} {
		if rev == "" {
			out = append(out, "")
			continue
		}
		sha, err := exec.Command("git", "-C", repo, "rev-parse", "--verify", "-q", rev+"^{commit}").Output()
		if err != nil {
			return "", false
		}
		out = append(out, strings.TrimSpace(string(sha)))
	}
	return strings.Join(out, ".."), true
}

// readResult reads a line's result as `--json` printed it.
func readResult(file string) (*Result, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var r Result
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("%s: %v", file, err)
	}
	return &r, nil
}

// runGate runs a gate step. Given a range, as on a merge request or before
// a push, its thresholds read only the findings the change adds since the
// range's base, and what it adds is said; on a schedule or a release,
// every finding (ADR-0049). The range is the one given, workline's own
// commits in it too: the tree checked out is what the gate's tools read.
//
// On main after a merge, what the merge adds since main's tip before it
// counts, and every finding is kept, by HEAD's commit, in the folder kept
// names: the base of the merge requests based on it. With no commit
// before it to compare with, every finding is said, none counted but a
// check that did not work. On a change, kept is where the merge base's
// findings are read, when main kept them (ADR-0050).
func runGate(repo, name, event, rng, kept string) (*verdict.Verdict, *gate.Added) {
	if event == MainEvent {
		since := gitrange.Base(rng)
		v, add := gate.RunWith(repo, name, gate.Options{Since: since, Kept: kept, Keep: true})
		if since == "" {
			nothingCounted(v)
		}
		return v, add
	}
	if since := gitrange.Base(rng); since != "" && event != "schedule" && event != "release" {
		return gate.RunWith(repo, name, gate.Options{Since: since, Kept: kept})
	}
	return gate.Run(repo, name), nil
}

// nothingCounted lowers a gate's findings over its thresholds to
// warnings, its verdict then given by what is left: a check that did not
// work still blocks (principle 12).
func nothingCounted(v *verdict.Verdict) {
	v.Status = verdict.Pass
	for i, f := range v.Findings {
		if f.Rule == "over-threshold" {
			v.Findings[i].Level = "warn"
		}
		if l := v.Findings[i].Level; l == "" || l == verdict.Block {
			v.Status = verdict.Block
		}
	}
	v.Summary += "; no commit before it to compare with: every finding said, none counted"
}

// mainRun is what main's run after a merge compares: main's tip before
// it, the range's start when it is a commit here — not the zeros a
// forge gives a branch's first push —, else HEAD's first parent; and the
// range the gates run on, from it to HEAD, "" when there is none before.
func mainRun(repo, rng string) (*MainRun, string) {
	commit := func(rev string) string {
		out, err := exec.Command("git", "-C", repo, "rev-parse", "--verify", "-q", rev+"^{commit}").Output()
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(out))
	}
	m := &MainRun{Head: commit("HEAD")}
	if b := gitrange.Base(rng); strings.Trim(b, "0") != "" {
		m.Before = commit(b)
	}
	if m.Before == "" {
		m.Before = commit("HEAD^1")
	}
	if m.Before == "" || m.Head == "" {
		return m, ""
	}
	if out, err := exec.Command("git", "-C", repo, "log", "--first-parent", "--format=%h %s", "-n", "50", m.Before+".."+m.Head).Output(); err == nil {
		for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			if l != "" {
				m.Commits = append(m.Commits, l)
			}
		}
	}
	return m, m.Before + ".." + m.Head
}

// stepsOf is an event's steps. On main after a merge, the merge
// request's gates come first, each once: their findings there are the
// next base, and two merge requests green alone may break main together.
func stepsOf(cfg *routing.Config, event string) ([]string, bool) {
	steps, ok := cfg.Events[event]
	if event != MainEvent || !ok {
		return steps, ok
	}
	var out []string
	for _, s := range cfg.Events["merge-request"] {
		if strings.HasPrefix(s, "gate:") && !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	for _, s := range steps {
		if !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	return out, true
}

// worst is the verdict of a line whose steps all ran: a block on the work
// first, then a person's decision, then an outside failure, which is never
// a verdict on the work.
func worst(a, b string) string {
	rank := map[string]int{verdict.Pass: 0, verdict.BlockedExternal: 1, verdict.Human: 2, verdict.Block: 3}
	if rank[b] > rank[a] {
		return b
	}
	return a
}

// runRole runs one role, then the handoffs it asked for. It returns the
// step that did not pass and its verdict, or the role and pass; a handoff
// past max-handoffs is a block.
func runRole(res *Result, cfg *routing.Config, o engine.Options, depth int) (string, string) {
	start := time.Now()
	r := engine.Run(o)
	label := o.Role
	if o.Event == "handoff" {
		label += " (handoff)"
	}
	res.Steps = append(res.Steps, Step{Name: label, Status: r.Status, Result: r, Seconds: Since(start)})
	res.Findings = append(res.Findings, r.Findings...)
	res.AgentCalls += r.AgentCalls
	res.Calls = append(res.Calls, r.Calls...)
	if len(r.Pending) > 0 { // a run gone round: a run folder a round
		res.Pending = append(res.Pending, r.Pending...)
	} else if r.ToApply {
		res.Pending = append(res.Pending, r.RunDir)
	}
	if r.Status != verdict.Pass {
		return label, r.Status
	}
	for _, h := range r.Handoffs {
		m, _ := h.(map[string]any)
		to, _ := m["role"].(string)
		reason, _ := m["reason"].(string)
		if depth+1 > cfg.MaxHandoffs {
			res.Findings = append(res.Findings, verdict.Finding{Rule: "routing-error", Where: label,
				Message: fmt.Sprintf("more than %d handoffs in a row; stopped before %s", cfg.MaxHandoffs, to)})
			return label, verdict.Block
		}
		next := o
		next.Role, next.Event, next.AI = to, "handoff", o.AI
		next.Inputs = map[string]string{"handoff-from": o.Role, "handoff-reason": reason}
		if d, ok := o.Inputs[engine.DraftInput]; ok { // a draft's handoff gets no agent either (ADR-0050)
			next.Inputs[engine.DraftInput] = d
		}
		next.Targets = nil
		if step, status := runRole(res, cfg, next, depth+1); status != verdict.Pass {
			return step, status
		}
	}
	return label, verdict.Pass
}

func stopped(res *Result, step, status string) *Result {
	res.Status = status
	res.Summary = fmt.Sprintf("stopped at %s (%s)", step, status)
	return res
}

func failed(res *Result, where, msg string) *Result {
	res.Status = verdict.Block
	res.Summary = "the line could not run"
	res.Findings = append(res.Findings, verdict.Finding{Rule: "routing-error", Where: where, Message: msg})
	return res
}

// beforeOwn returns the role that made the last commit of rng, when the
// engine made it, and rng ending before the engine's last commits: each
// left for its parent, a merge for its first, the branch's side (`workline
// follow`'s merge of the base, ADR-0041). The range is "" when nothing is
// left in it; role is "" when its last commit is a person's, or with no
// range. A trailer is not taken on trust (engine.OwnCommit): a merge of
// anything but the range's base, or bringing more than it, a commit
// changing more than doc pages, is judged as a person's. The committer
// checks them all the same (Run).
func beforeOwn(repo, rng string) (role, before string) {
	if rng == "" {
		return "", rng
	}
	head, base := gitrange.Head(rng), gitrange.Base(rng)
	cur := head
	for range 1000 {
		out, err := exec.Command("git", "-C", repo, "log", "-1", "--format=%P%x1f%(trailers:key="+engine.OwnTrailer+",valueonly,separator=%x2c)", cur).Output()
		if err != nil {
			break
		}
		parents, by, _ := strings.Cut(strings.TrimSpace(string(out)), "\x1f")
		if by = strings.TrimSpace(by); by == "" || engine.OwnCommit(repo, cur, base) == "" {
			break
		}
		if role == "" {
			role = by
		}
		first, _, _ := strings.Cut(parents, " ")
		if cur = first; cur == "" {
			return role, "" // the history's first commit: nothing before it
		}
	}
	if role == "" {
		return "", rng
	}
	before = cur
	if from, _, ok := strings.Cut(rng, ".."); ok {
		before = from + ".." + cur
	} else if words := strings.Fields(rng); len(words) > 1 {
		for i, w := range words {
			if w == head {
				words[i] = cur
			}
		}
		before = strings.Join(words, " ")
	}
	out, err := exec.Command("git", append([]string{"-C", repo, "rev-list", "--count"}, gitrange.Args(before)...)...).Output()
	if err == nil && strings.TrimSpace(string(out)) == "0" {
		return role, ""
	}
	return role, before
}
