// Package agent runs the "propose" step. Real coding agents (claude, codex,
// agy, opencode...) plug in here as adapters; this first version ships the
// adapters the conformance tests need.
package agent

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrUnavailable means the agent could not be reached for a reason outside the
// role: quota, authentication, network. The run ends as blocked-external.
var ErrUnavailable = errors.New("agent unavailable")

// Agent answers the question in in/task.md by writing out/intentions.yaml.
type Agent interface {
	Propose(runDir string) error
}

// Parse turns the --ai value into an agent. "none" returns nil: no agent.
//
//	none                  no AI; the role's without-ai rule applies
//	fake:<file>           replays the proposals in <file> (conformance tests)
//	unavailable:<reason>  fails like an agent whose quota or login is gone
func Parse(spec string) (Agent, error) {
	switch {
	case spec == "" || spec == "none":
		return nil, nil
	case strings.HasPrefix(spec, "fake:"):
		return fake{file: strings.TrimPrefix(spec, "fake:")}, nil
	case strings.HasPrefix(spec, "unavailable:"):
		return unavailable{reason: strings.TrimPrefix(spec, "unavailable:")}, nil
	}
	return nil, fmt.Errorf("unknown agent %q (this version knows none, fake:<file>, unavailable:<reason>)", spec)
}

type fake struct{ file string }

func (f fake) Propose(runDir string) error {
	data, err := os.ReadFile(f.file)
	if err != nil {
		return fmt.Errorf("fake agent: %w", err)
	}
	return os.WriteFile(filepath.Join(runDir, "out", "intentions.yaml"), data, 0o644)
}

type unavailable struct{ reason string }

func (u unavailable) Propose(string) error {
	return fmt.Errorf("%w: %s", ErrUnavailable, u.reason)
}
