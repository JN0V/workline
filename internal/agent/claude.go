package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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

func (claude) Propose(req Request) (Call, error) {
	call := Call{Agent: "claude", Tier: req.tier(), Effort: req.Role.Model.Effort}
	bin, err := exec.LookPath("claude")
	if err != nil {
		return call, fmt.Errorf("%w: claude is not installed", ErrUnavailable)
	}
	system, user, err := Prompt(req)
	if err != nil {
		return call, err
	}
	args := []string{"-p", "--output-format", "json", "--tools", "", "--strict-mcp-config",
		"--no-session-persistence", "--system-prompt", system}
	if m := claudeTier[call.Tier]; m != "" {
		args = append(args, "--model", m)
	}
	if e := claudeEffort[call.Effort]; e != "" {
		args = append(args, "--effort", e)
	}
	limit, err := req.Role.Model.AnswerTimeout()
	if err != nil {
		return call, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = os.TempDir()
	cmd.Stdin = strings.NewReader(user)
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	runErr := cmd.Run()
	answer, parsed := claudeAnswer(out.Bytes(), &call)
	if runErr != nil || (parsed && answer.IsError) {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return call, fmt.Errorf("%w: claude did not answer within %s (the role's model.timeout)", ErrUnavailable, limit)
		}
		said := out.String()
		if parsed {
			said = answer.Result
		}
		return call, fmt.Errorf("%w: claude failed: %s", ErrUnavailable, lastLine(errOut.String()+said))
	}
	if !parsed {
		answer.Result = out.String() // not the JSON asked for: read it as the answer itself
	}
	proposals, err := proposalsFrom(answer.Result)
	if err != nil {
		_ = os.WriteFile(filepath.Join(req.RunDir, "out", "agent-answer.txt"), []byte(answer.Result), 0o644)
		return call, fmt.Errorf("%w: %v", ErrInvalidOutput, err)
	}
	return call, os.WriteFile(filepath.Join(req.RunDir, "out", "intentions.yaml"), proposals, 0o644)
}

// claudeResult is the part of `claude -p --output-format json` workline reads.
type claudeResult struct {
	Result     string  `json:"result"`
	IsError    bool    `json:"is_error"`
	CostUSD    float64 `json:"total_cost_usd"`
	ModelUsage map[string]struct {
		InputTokens              int `json:"inputTokens"`
		OutputTokens             int `json:"outputTokens"`
		CacheReadInputTokens     int `json:"cacheReadInputTokens"`
		CacheCreationInputTokens int `json:"cacheCreationInputTokens"`
	} `json:"modelUsage"`
}

// claudeAnswer reads Claude Code's JSON answer, and records in call the model
// that wrote it and what the call used. Claude Code may call a small model on
// the side; the model that wrote the most is the one answering, the others
// count in the cost only. false: stdout was not that JSON.
func claudeAnswer(stdout []byte, call *Call) (claudeResult, bool) {
	var r claudeResult
	if err := json.Unmarshal(bytes.TrimSpace(stdout), &r); err != nil {
		return r, false
	}
	most := -1
	for m, u := range r.ModelUsage {
		if u.OutputTokens > most || (u.OutputTokens == most && m < call.Model) {
			call.Model, most = m, u.OutputTokens
		}
		call.TokensIn += u.InputTokens + u.CacheReadInputTokens + u.CacheCreationInputTokens
		call.TokensCached += u.CacheReadInputTokens
		call.TokensOut += u.OutputTokens
	}
	call.CostUSD = r.CostUSD
	return r, true
}

// proposalsFrom extracts the YAML list from an answer, tolerating a code fence.
// The answer itself is tried first: a proposal may hold a code fence of its
// own (a doc's code block, moved by a patch). Only when it does not read is it
// taken from between the first and the last fence lines.
func proposalsFrom(answer string) ([]byte, error) {
	s := strings.TrimSpace(answer)
	var list []map[string]any
	if err := yaml.Unmarshal([]byte(s), &list); err != nil || len(list) == 0 {
		lines := strings.Split(s, "\n")
		first, last := -1, -1
		for i, l := range lines {
			if strings.HasPrefix(l, "```") {
				if first < 0 {
					first = i
				}
				last = i
			}
		}
		if first < 0 || last == first {
			return nil, fmt.Errorf("expected a YAML list of proposals")
		}
		s = strings.TrimSpace(strings.Join(lines[first+1:last], "\n"))
		list = nil
		if err := yaml.Unmarshal([]byte(s), &list); err != nil || len(list) == 0 {
			return nil, fmt.Errorf("expected a YAML list of proposals")
		}
	}
	for i, m := range list {
		if len(m) != 1 {
			return nil, fmt.Errorf("proposal %d holds %d kinds; each holds exactly one", i+1, len(m))
		}
	}
	return []byte(s + "\n"), nil
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return lines[len(lines)-1]
}
