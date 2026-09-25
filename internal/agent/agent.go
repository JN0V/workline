// Package agent runs the "propose" step: it gives the role's question to a
// coding agent and collects its proposals in out/intentions.yaml.
package agent

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/JN0V/workline/internal/role"
)

// ErrUnavailable means the agent could not be reached for a reason outside the
// role: quota, authentication, network, not installed. The run ends as
// blocked-external if the role cannot pass without it.
var ErrUnavailable = errors.New("agent unavailable")

// ErrInvalidOutput means the agent answered, but not with proposals the engine
// can read. Its answer is dropped, as if it had proposed nothing.
var ErrInvalidOutput = errors.New("agent answered without valid proposals")

// Request is everything an agent gets for one decision.
type Request struct {
	RunDir string
	Repo   string
	Role   *role.Role
	Tier   string // overrides the role's tier, when a retry steps up
}

// tier is the tier asked of the agent: the role's, unless a retry stepped up.
func (r Request) tier() string {
	if r.Tier != "" {
		return r.Tier
	}
	return r.Role.Model.Tier
}

// Call records one question put to an agent: what was asked, and what
// answered. The same tier names a newer model when the agent's aliases move,
// so a score is only worth something next to the exact model.
type Call struct {
	Agent  string `json:"agent"`
	Tier   string `json:"tier,omitempty"`
	Effort string `json:"effort,omitempty"` // the role's level, before the agent maps it
	Model  string `json:"model,omitempty"`  // the one that wrote the answer, as the agent reports it
	// What the call used, side calls included, as the agent reports it. On a
	// subscription the tokens are what counts; the cost is the list price.
	TokensIn     int     `json:"tokens-in,omitempty"`     // the whole input, cache included
	TokensCached int     `json:"tokens-cached,omitempty"` // the part of it read from cache
	TokensOut    int     `json:"tokens-out,omitempty"`
	CostUSD      float64 `json:"cost-usd,omitempty"`
	Seconds      float64 `json:"seconds"`
}

// Agent answers the question in in/task.md by writing out/intentions.yaml,
// and says what answered, even when it fails.
type Agent interface {
	Propose(Request) (Call, error)
}

// Parse turns an agent spec into an agent. "none" (or "") returns nil: no agent.
//
//	none                  no AI; the role's without-ai rule applies
//	claude                Claude Code, headless (claude -p), on the model the role's tier asks
//	claude:<model>        on that model, whatever the tier: an alias (sonnet) or an exact id
//	claude:<model>@<effort>, claude:@<effort>
//	                      and at that effort (none, low, medium, high, max), whatever the role's
//	fake:<file>           replays the proposals in <file> (conformance tests)
//	unavailable:<reason>  fails like an agent whose quota or login is gone
func Parse(spec string) (Agent, error) {
	switch {
	case spec == "" || spec == "none":
		return nil, nil
	case spec == "claude":
		return claude{}, nil
	case strings.HasPrefix(spec, "claude:"):
		model, effort, _ := strings.Cut(strings.TrimPrefix(spec, "claude:"), "@")
		if _, ok := claudeEffort[effort]; effort != "" && !ok {
			return nil, fmt.Errorf("agent %q: unknown effort %q (known: none, low, medium, high, max)", spec, effort)
		}
		if model == "" && effort == "" {
			return nil, fmt.Errorf("agent %q: name a model, an effort, or both (claude:sonnet@high)", spec)
		}
		return claude{model: model, effort: effort}, nil
	case strings.HasPrefix(spec, "fake:"):
		return fake{file: strings.TrimPrefix(spec, "fake:")}, nil
	case strings.HasPrefix(spec, "unavailable:"):
		return unavailable{reason: strings.TrimPrefix(spec, "unavailable:")}, nil
	}
	return nil, fmt.Errorf("unknown agent %q (known: none, claude, claude:<model>@<effort>, fake:<file>, unavailable:<reason>)", spec)
}

type fake struct{ file string }

func (f fake) Propose(r Request) (Call, error) {
	call := Call{Agent: "fake", Tier: r.tier(), Effort: r.Role.Model.Effort,
		Model: strings.TrimSuffix(filepath.Base(f.file), ".yaml")}
	data, err := os.ReadFile(f.file)
	if err != nil {
		return call, fmt.Errorf("fake agent: %w", err)
	}
	return call, os.WriteFile(filepath.Join(r.RunDir, "out", "intentions.yaml"), data, 0o644)
}

type unavailable struct{ reason string }

func (u unavailable) Propose(r Request) (Call, error) {
	return Call{Agent: "unavailable", Tier: r.tier(), Effort: r.Role.Model.Effort}, fmt.Errorf("%w: %s", ErrUnavailable, u.reason)
}
