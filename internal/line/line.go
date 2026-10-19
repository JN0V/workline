// Package line runs the steps routing names for an event, in order, and the
// handoffs they ask for (docs/spec/routing.md).
package line

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"slices"
	"strings"

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

	read *Result // From, once read
	err  error
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
	res := &Result{Status: verdict.Pass, Steps: []Step{}}
	cfg, err := routing.Load(base.Repo)
	if err != nil {
		f := failed(res, "routing", err.Error())
		if role.IsConfigError(err) {
			f.Findings[len(f.Findings)-1].Rule = "config-invalid"
		}
		return f
	}
	steps, ok := cfg.Events[event]
	if !ok {
		return failed(res, "routing", fmt.Sprintf("routing names no step for event %q", event))
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
	if role, rng := beforeOwn(base.Repo, base.Inputs["range"]); role != "" {
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
	// The gates alone: each one runs, so that each is shown.
	stopAtFirst := cfg.StopsAtFirst(event) && !gates.Only
	if gates.Only {
		res.Range = whole["range"]
		res.Commits, _ = commitsOf(base.Repo, whole["range"])
	}
	var notPassed []string // with every step run: those that did not pass, and how
	considered := 0
	for _, name := range steps {
		step, status := name, verdict.Pass
		g, isGate := strings.CutPrefix(name, "gate:")
		if nothingElse && name != committerRole || gates.Only && !isGate || warnOnly && (isGate || !apartOn(base.RolesDir, name, event)) {
			continue
		}
		considered++
		if isGate {
			s := gates.step(base.Repo, g, event, whole["range"])
			res.Steps = append(res.Steps, s)
			res.Findings = append(res.Findings, s.Gate.Findings...)
			status = s.Status
		} else {
			o := base
			if name == committerRole {
				o.Inputs = whole
			}
			// What only warns runs apart where the role reads it apart on
			// the event; any other step runs whole, never told to leave it.
			if _, told := o.Inputs[WarnInput]; told && !apartOn(base.RolesDir, name, event) {
				o.Inputs = maps.Clone(o.Inputs)
				delete(o.Inputs, WarnInput)
			}
			o.Role, o.Event = name, event
			back := func() {}
			if holdsRelease(base.RolesDir, name) {
				switch {
				// At night, gardening with a forge: first the release's
				// docs, on its merge request's head; what its fix patches
				// waits for the gardening after it (ADR-0050, §7).
				case event == "schedule" && base.OpenMergeRequest:
					o.Waiting = nightRelease(res, cfg, o, &notPassed)
				// `release` on a release tool's merge request, a person's
				// run: on its head, its fix in a merge request of its own.
				case event == "release" && o.Target != nil && o.Target.Kind == "merge-request":
					var err error
					var branch string
					if back, branch, err = engine.OnReleaseHead(o); err != nil {
						res.Steps = append(res.Steps, Step{Name: name, Status: verdict.Block})
						res.Findings = append(res.Findings, verdict.Finding{Rule: "release-head", Where: name, Message: err.Error()})
						if stopAtFirst {
							return stopped(res, name, verdict.Block)
						}
						notPassed = append(notPassed, fmt.Sprintf("%s (%s)", name, verdict.Block))
						res.Status = worst(res.Status, verdict.Block)
						continue
					}
					o.PushToMergeRequest = o.PushToMergeRequest || branch != ""
				}
			}
			step, status = runRole(res, cfg, o, 0)
			back()
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
	if waits(engine.AIWaitsNight) {
		res.Summary += "; a release: no agent asked for its docs, judged at night with the gardening"
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
		v, added := runGate(repo, name, event, rng)
		return Step{Name: "gate:" + name, Status: v.Status, Gate: v, New: added}
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
			return Step{Name: s.Name, Status: s.Status, Gate: s.Gate, From: g.From}
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
func runGate(repo, name, event, rng string) (*verdict.Verdict, *gate.Added) {
	if since := gitrange.Base(rng); since != "" && event != "schedule" && event != "release" {
		return gate.RunNew(repo, name, since)
	}
	return gate.Run(repo, name), nil
}

// holdsRelease says whether the role a step names has release duties (its
// role.yaml accepts `release`): the documentalist's hold (ADR-0017).
func holdsRelease(rolesDir, name string) bool {
	r, err := role.Load(rolesDir, name)
	return err == nil && r.Accepts("release")
}

// nightRelease runs, before a role's gardening, its run on the release
// tool's merge request open into the branch checked out, with its agent
// (ADR-0050, §7): the docs due for the release judged on that merge
// request's head, checked out for the run and put back after, their fix
// proposed in a merge request of its own (ADR-0017). Its step is said
// apart from the role's: `<role> (release #n)`. The release waiting for
// its docs is said on its merge request, and holds nothing of the night's
// line (a warning); anything else that did not pass counts. It returns
// what the run left to apply, by the task its fix goes to, for the
// gardening after it to leave alone.
func nightRelease(res *Result, cfg *routing.Config, o engine.Options, notPassed *[]string) map[string][]string {
	mr, said, err := engine.OpenRelease(o)
	res.Findings = append(res.Findings, said...)
	unread := func(err error) {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "release-unread", Level: "warn", Where: o.Role,
			Message: "the release's merge request could not be read, nor its docs judged tonight: " + err.Error()})
	}
	if err != nil {
		unread(err)
		return nil
	}
	if mr == nil {
		return nil
	}
	ro := o
	ro.Event, ro.Target, ro.Branch = "release", &forge.Target{Kind: "merge-request", ID: mr.ID}, mr.Branch
	ro.OpenMergeRequest, ro.PushToMergeRequest = false, true
	back, _, err := engine.OnReleaseHead(ro)
	if err != nil {
		unread(err)
		return nil
	}
	at, pending := len(res.Steps), len(res.Pending)
	_, status := runRole(res, cfg, ro, 0)
	back()
	label := fmt.Sprintf("%s (release #%d)", o.Role, mr.ID)
	res.Steps[at].Name = label
	if status != verdict.Pass && !engine.ReleaseHeld(res.Steps[at].Result) {
		*notPassed = append(*notPassed, fmt.Sprintf("%s (%s)", label, status))
		res.Status = worst(res.Status, status)
	}
	if left := res.Pending[pending:]; len(left) > 0 {
		return map[string][]string{"release": slices.Clone(left)}
	}
	return nil
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
	r := engine.Run(o)
	label := o.Role
	if o.Event == "handoff" {
		label += " (handoff)"
	}
	res.Steps = append(res.Steps, Step{Name: label, Status: r.Status, Result: r})
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
