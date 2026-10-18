// Package line runs the steps routing names for an event, in order, and the
// handoffs they ask for (docs/spec/routing.md).
package line

import (
	"fmt"
	"maps"
	"os/exec"
	"strings"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/engine"
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
}

// Run runs event's steps. base carries what every role run shares (repository,
// roles, agent, inputs, forge, target, scope, NoApply); Role and Event are set
// per step. The first step that does not pass stops the line: it fails
// closed. On an event the routing sets `fail-fast: false` for (schedule, by
// default), every step runs all the same and the line takes the worst
// verdict (ADR-0037). With NoApply, each step judges the tree as it is — no
// step sees what an earlier one proposed — and handoffs wait, since nothing
// is applied yet.
func Run(event string, base engine.Options) *Result {
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
	stopAtFirst := cfg.StopsAtFirst(event)
	var notPassed []string // with every step run: those that did not pass, and how
	for _, name := range steps {
		step, status := name, verdict.Pass
		if nothingElse && name != committerRole {
			continue
		}
		if g, isGate := strings.CutPrefix(name, "gate:"); isGate {
			v := runGate(base.Repo, g, event, whole["range"])
			res.Steps = append(res.Steps, Step{Name: name, Status: v.Status, Gate: v})
			res.Findings = append(res.Findings, v.Findings...)
			status = v.Status
		} else {
			o := base
			if name == committerRole {
				o.Inputs = whole
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
		res.Summary = fmt.Sprintf("%s: every step ran; %d of %d did not pass: %s", event, len(notPassed), len(steps), strings.Join(notPassed, ", "))
	} else {
		res.Summary = fmt.Sprintf("%s: %d steps passed", event, len(res.Steps))
	}
	if len(res.Pending) > 0 {
		res.Summary += fmt.Sprintf("; %d runs to apply", len(res.Pending))
	}
	return res
}

// committerRole is the step that checks every commit of a range, workline's
// own included (beforeOwn leaves them to it).
const committerRole = "committer"

// runGate runs a gate step. Given a range, as on a merge request or before
// a push, its thresholds read only the findings the change adds since the
// range's base; on a schedule or a release, every finding (ADR-0049). The
// range is the one given, workline's own commits in it too: the tree
// checked out is what the gate's tools read.
func runGate(repo, name, event, rng string) *verdict.Verdict {
	if since := gitrange.Base(rng); since != "" && event != "schedule" && event != "release" {
		return gate.RunNew(repo, name, since)
	}
	return gate.Run(repo, name)
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
