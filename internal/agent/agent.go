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
}

// Agent answers the question in in/task.md by writing out/intentions.yaml.
type Agent interface {
	Propose(Request) error
}

// Parse turns an agent spec into an agent. "none" (or "") returns nil: no agent.
//
//	none                  no AI; the role's without-ai rule applies
//	claude                Claude Code, headless (claude -p)
//	fake:<file>           replays the proposals in <file> (conformance tests)
//	unavailable:<reason>  fails like an agent whose quota or login is gone
func Parse(spec string) (Agent, error) {
	switch {
	case spec == "" || spec == "none":
		return nil, nil
	case spec == "claude":
		return claude{}, nil
	case strings.HasPrefix(spec, "fake:"):
		return fake{file: strings.TrimPrefix(spec, "fake:")}, nil
	case strings.HasPrefix(spec, "unavailable:"):
		return unavailable{reason: strings.TrimPrefix(spec, "unavailable:")}, nil
	}
	return nil, fmt.Errorf("unknown agent %q (known: none, claude, fake:<file>, unavailable:<reason>)", spec)
}

type fake struct{ file string }

func (f fake) Propose(r Request) error {
	data, err := os.ReadFile(f.file)
	if err != nil {
		return fmt.Errorf("fake agent: %w", err)
	}
	return os.WriteFile(filepath.Join(r.RunDir, "out", "intentions.yaml"), data, 0o644)
}

type unavailable struct{ reason string }

func (u unavailable) Propose(Request) error {
	return fmt.Errorf("%w: %s", ErrUnavailable, u.reason)
}
