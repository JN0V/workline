// Package line runs the steps routing names for an event, in order, and the
// handoffs they ask for (docs/spec/routing.md).
package line

import (
	"fmt"
	"strings"

	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/gate"
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
	// Pending lists the runs judged with NoApply, in the line's order: what
	// `workline apply` is given in the job that holds the write token.
	Pending []string `json:"pending,omitempty"`
}

// Run runs event's steps. base carries what every role run shares (repository,
// roles, agent, inputs, forge, target, scope, NoApply); Role and Event are set
// per step. The first step that does not pass stops the line: it fails closed.
// With NoApply, each step judges the tree as it is — no step sees what an
// earlier one proposed — and handoffs wait, since nothing is applied yet.
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
	for _, name := range steps {
		if g, isGate := strings.CutPrefix(name, "gate:"); isGate {
			v := gate.Run(base.Repo, g)
			res.Steps = append(res.Steps, Step{Name: name, Status: v.Status, Gate: v})
			res.Findings = append(res.Findings, v.Findings...)
			if v.Status != verdict.Pass {
				return stopped(res, name, v.Status)
			}
			continue
		}
		o := base
		o.Role, o.Event = name, event
		if !runRole(res, cfg, o, 0) {
			return res
		}
	}
	res.Summary = fmt.Sprintf("%s: %d steps passed", event, len(res.Steps))
	if len(res.Pending) > 0 {
		res.Summary += fmt.Sprintf("; %d runs to apply", len(res.Pending))
	}
	return res
}

// runRole runs one role, then the handoffs it asked for; false stops the line.
func runRole(res *Result, cfg *routing.Config, o engine.Options, depth int) bool {
	r := engine.Run(o)
	label := o.Role
	if o.Event == "handoff" {
		label += " (handoff)"
	}
	res.Steps = append(res.Steps, Step{Name: label, Status: r.Status, Result: r})
	res.Findings = append(res.Findings, r.Findings...)
	res.AgentCalls += r.AgentCalls
	if r.ToApply {
		res.Pending = append(res.Pending, r.RunDir)
	}
	if r.Status != verdict.Pass {
		stopped(res, label, r.Status)
		return false
	}
	for _, h := range r.Handoffs {
		m, _ := h.(map[string]any)
		to, _ := m["role"].(string)
		reason, _ := m["reason"].(string)
		if depth+1 > cfg.MaxHandoffs {
			failed(res, label, fmt.Sprintf("more than %d handoffs in a row; stopped before %s", cfg.MaxHandoffs, to))
			return false
		}
		next := o
		next.Role, next.Event, next.AI = to, "handoff", o.AI
		next.Inputs = map[string]string{"handoff-from": o.Role, "handoff-reason": reason}
		next.Targets = nil
		if !runRole(res, cfg, next, depth+1) {
			return false
		}
	}
	return true
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
