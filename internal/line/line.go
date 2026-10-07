// Package line runs the steps routing names for an event, in order, and the
// handoffs they ask for (docs/spec/routing.md).
package line

import (
	"fmt"
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
	// What the engine committed does not wake the line again (routing spec):
	// the fix a role pushed to a merge request would be judged, and pushed.
	if role := ownCommit(base.Repo, base.Inputs["range"]); role != "" {
		res.Summary = fmt.Sprintf("%s: the last commit is workline's own (%s): nothing to judge", event, role)
		return res
	}
	stopAtFirst := cfg.StopsAtFirst(event)
	var notPassed []string // with every step run: those that did not pass, and how
	for _, name := range steps {
		step, status := name, verdict.Pass
		if g, isGate := strings.CutPrefix(name, "gate:"); isGate {
			v := gate.Run(base.Repo, g)
			res.Steps = append(res.Steps, Step{Name: name, Status: v.Status, Gate: v})
			res.Findings = append(res.Findings, v.Findings...)
			status = v.Status
		} else {
			o := base
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

// ownCommit returns the role that made the last commit of rng, when the
// engine made it; "" otherwise, or with no range.
func ownCommit(repo, rng string) string {
	if rng == "" {
		return ""
	}
	head := gitrange.Head(rng)
	out, err := exec.Command("git", "-C", repo, "log", "-1", "--format=%(trailers:key="+engine.OwnTrailer+",valueonly)", head).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
