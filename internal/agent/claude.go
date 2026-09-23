package agent

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

// claude runs Claude Code headless. It gets no tools, no MCP servers and no
// session, and runs outside the repository so the project's CLAUDE.md is not
// loaded: the role's facets are its whole context.
type claude struct{}

// Until the model grid is generated (docs/spec/model-grid.md), Claude's own
// aliases stand for the tiers.
var claudeTier = map[string]string{"light": "haiku", "standard": "sonnet", "frontier": "opus"}
var claudeEffort = map[string]string{"none": "low", "low": "low", "medium": "medium", "high": "high", "max": "max"}

func (claude) Propose(req Request) error {
	bin, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("%w: claude is not installed", ErrUnavailable)
	}
	system, user, err := Prompt(req)
	if err != nil {
		return err
	}
	args := []string{"-p", "--output-format", "text", "--tools", "", "--strict-mcp-config",
		"--no-session-persistence", "--system-prompt", system}
	if m := claudeTier[req.Role.Model.Tier]; m != "" {
		args = append(args, "--model", m)
	}
	if e := claudeEffort[req.Role.Model.Effort]; e != "" {
		args = append(args, "--effort", e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = os.TempDir()
	cmd.Stdin = strings.NewReader(user)
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	if err := cmd.Run(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("%w: claude did not answer within 3 minutes", ErrUnavailable)
		}
		return fmt.Errorf("%w: claude failed: %s", ErrUnavailable, lastLine(errOut.String()+out.String()))
	}
	proposals, err := proposalsFrom(out.String())
	if err != nil {
		_ = os.WriteFile(filepath.Join(req.RunDir, "out", "agent-answer.txt"), out.Bytes(), 0o644)
		return fmt.Errorf("%w: %v", ErrInvalidOutput, err)
	}
	return os.WriteFile(filepath.Join(req.RunDir, "out", "intentions.yaml"), proposals, 0o644)
}

// proposalsFrom extracts the YAML list from an answer, tolerating a code fence.
func proposalsFrom(answer string) ([]byte, error) {
	s := strings.TrimSpace(answer)
	if i := strings.Index(s, "```"); i >= 0 {
		s = s[i+3:]
		s = strings.TrimPrefix(strings.TrimPrefix(s, "yaml"), "yml")
		if j := strings.Index(s, "```"); j >= 0 {
			s = s[:j]
		}
	}
	s = strings.TrimSpace(s)
	var list []map[string]any
	if err := yaml.Unmarshal([]byte(s), &list); err != nil || len(list) == 0 {
		return nil, fmt.Errorf("expected a YAML list of proposals")
	}
	return []byte(s + "\n"), nil
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return lines[len(lines)-1]
}
