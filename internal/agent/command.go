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

	"go.yaml.in/yaml/v3"
)

// command runs any command as the agent, through sh -c: another provider's
// CLI, a wrapper around one, or a factory of one's own. It reads the prompt on
// its input, persona first, and answers with the proposals on its output. It
// runs outside the repository, like claude, so no project file is read by
// accident. It is told the tier and effort asked in WORKLINE_TIER and
// WORKLINE_EFFORT, to pick its model; it may say what answered in the file
// WORKLINE_CALL names (model, tokens-in, tokens-cached, tokens-out, cost-usd,
// in YAML).
type command struct{ script string }

func (c command) Propose(req Request) (Call, error) {
	call := Call{Agent: "cmd", Tier: req.tier(), Effort: req.Role.Model.Effort}
	call.Asked = call.Tier
	system, user, err := Prompt(req)
	if err != nil {
		return call, err
	}
	report, err := os.CreateTemp("", "workline-call-*.yaml")
	if err != nil {
		return call, err
	}
	report.Close()
	defer os.Remove(report.Name())
	limit, err := req.Role.Model.AnswerTimeout()
	if err != nil {
		return call, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", c.script)
	cmd.Dir = os.TempDir()
	cmd.Env = append(os.Environ(), "WORKLINE_TIER="+call.Tier, "WORKLINE_EFFORT="+call.Effort, "WORKLINE_CALL="+report.Name())
	cmd.Stdin = strings.NewReader(system + "\n\n" + user)
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	runErr := cmd.Run()
	if data, err := os.ReadFile(report.Name()); err == nil {
		var said struct {
			Model        string  `yaml:"model"`
			TokensIn     int     `yaml:"tokens-in"`
			TokensCached int     `yaml:"tokens-cached"`
			TokensOut    int     `yaml:"tokens-out"`
			CostUSD      float64 `yaml:"cost-usd"`
		}
		if yaml.Unmarshal(data, &said) == nil {
			call.Model, call.TokensIn, call.TokensCached, call.TokensOut, call.CostUSD =
				said.Model, said.TokensIn, said.TokensCached, said.TokensOut, said.CostUSD
		}
	}
	if runErr != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return call, fmt.Errorf("%w: the command did not answer within %s (the role's model.timeout)", ErrUnavailable, limit)
		}
		return call, fmt.Errorf("%w: the command failed: %s", ErrUnavailable, lastLine(errOut.String()+out.String()))
	}
	proposals, err := proposalsFrom(out.String())
	if err != nil {
		_ = os.WriteFile(filepath.Join(req.RunDir, "out", "agent-answer.txt"), out.Bytes(), 0o644)
		return call, fmt.Errorf("%w: %v", ErrInvalidOutput, err)
	}
	return call, os.WriteFile(filepath.Join(req.RunDir, "out", "intentions.yaml"), proposals, 0o644)
}
