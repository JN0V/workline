package line

// The line of states (ADR-0059, decision 8): an event naming an issue
// runs the role its state wakes, read from the routing's table by a pure
// function (routing.Config.Wakes), not every step of the event.

import (
	"errors"
	"fmt"

	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/verdict"
)

// IssueEvent is the event naming one issue: its labelling on a forge, a
// state started by the step before (ADR-0059).
const IssueEvent = "issue"

// openForge opens the forge a line reads issues from: the one given, else
// the project's; nil with none.
func openForge(o engine.Options) (forge.Forge, error) {
	spec := o.Forge
	if spec == "" {
		if cfg, err := role.LoadProjectConfig(o.Repo); err == nil {
			spec = cfg.Forge
		}
	}
	if spec == "" || spec == "none" {
		return nil, nil
	}
	return forge.Open(spec, o.Repo)
}

// byState is the steps an event naming an issue runs: the one its state
// wakes, when the project routes it there; none, said, when it wakes
// none. ok is false, with the line's result, when the issue could not be
// read.
func byState(res *Result, cfg *routing.Config, o engine.Options, event string) (steps []string, ok bool) {
	where := fmt.Sprintf("#%d", o.Target.ID)
	f, err := openForge(o)
	if err == nil && f == nil {
		err = errors.New("no forge to read its state from; " + forge.Missing)
	}
	var is *forge.Issue
	if err == nil {
		is, err = f.Issue(o.Target.ID)
	}
	if err != nil {
		res.Status = verdict.Block
		if errors.Is(err, forge.ErrUnreachable) {
			res.Status = verdict.BlockedExternal
		}
		res.Summary = fmt.Sprintf("%s: %s's state could not be read", event, where)
		res.Findings = append(res.Findings, verdict.Finding{Rule: "state-unread", Where: where, Message: err.Error()})
		return nil, false
	}
	if is.Closed {
		res.Summary = fmt.Sprintf("%s: %s is closed: no state wakes a role", event, where)
		return nil, true
	}
	s, name, why := cfg.Wakes(is.Labels, event)
	if name == "" {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "state-wakes-none", Level: "info", Where: where, Message: why})
		res.Summary = fmt.Sprintf("%s: %s wakes no role: %s", event, where, why)
		return nil, true
	}
	res.Findings = append(res.Findings, verdict.Finding{Rule: "state-wakes", Level: "info", Where: where,
		Message: fmt.Sprintf("its state, %s, wakes the %s", s.Name, name)})
	return []string{name}, true
}
