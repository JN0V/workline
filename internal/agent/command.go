package agent

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
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
// in YAML, and turns). Where the role gave it a worktree (WorktreeDir), it
// runs there instead, its path in WORKLINE_WORKTREE, its caps in
// WORKLINE_MAX_SECONDS and WORKLINE_MAX_TURNS: it edits the files in place,
// and an answer left empty proposes nothing else (ADR-0056).
type command struct{ script string }

// WorktreeDir is the folder of a run where a role's pre checks out the
// code its agent edits in place: a cmd: agent runs there (ADR-0056).
const WorktreeDir = "worktree"

// Worktree is the run's worktree, "" when its role made none.
func Worktree(runDir string) string {
	dir := filepath.Join(runDir, WorktreeDir)
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return ""
	}
	return dir
}

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
	cmd.Dir, cmd.WaitDelay = os.TempDir(), waitAfterTimeout
	cmd.Env = append(withoutTokens(os.Environ()), "WORKLINE_TIER="+call.Tier, "WORKLINE_EFFORT="+call.Effort, "WORKLINE_CALL="+report.Name())
	tree := Worktree(req.RunDir)
	if tree != "" {
		// An agent that loops (ADR-0056): told its caps, killed whole at
		// its time, a child of its own left after it answered too.
		cmd.Dir = tree
		cmd.Env = append(cmd.Env, "WORKLINE_WORKTREE="+tree, fmt.Sprintf("WORKLINE_MAX_SECONDS=%d", int(math.Ceil(limit.Seconds()))))
		if req.MaxTurns > 0 {
			cmd.Env = append(cmd.Env, fmt.Sprintf("WORKLINE_MAX_TURNS=%d", req.MaxTurns))
		}
		inGroup(cmd)
	}
	cmd.Stdin = strings.NewReader(system + "\n\n" + user)
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	runErr := settled(ctx, cmd.Run())
	if tree != "" {
		_ = killGroup(cmd)
	}
	if data, err := os.ReadFile(report.Name()); err == nil {
		var said struct {
			Model        string  `yaml:"model"`
			TokensIn     int     `yaml:"tokens-in"`
			TokensCached int     `yaml:"tokens-cached"`
			TokensOut    int     `yaml:"tokens-out"`
			CostUSD      float64 `yaml:"cost-usd"`
			Turns        int     `yaml:"turns"`
		}
		if yaml.Unmarshal(data, &said) == nil {
			call.Model, call.TokensIn, call.TokensCached, call.TokensOut, call.CostUSD, call.Turns =
				said.Model, said.TokensIn, said.TokensCached, said.TokensOut, said.CostUSD, said.Turns
		}
	}
	if runErr != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return call, fmt.Errorf("%w: the command did not answer within %s (the role's model.timeout)", ErrUnavailable, limit)
		}
		return call, fmt.Errorf("%w: the command failed: %s", ErrUnavailable, lastLine(errOut.String()+out.String()))
	}
	// The answer as it came, kept for audit, as Claude's is.
	_ = os.WriteFile(filepath.Join(req.RunDir, "out", "agent-answer.txt"), out.Bytes(), 0o644)
	if tree != "" && strings.TrimSpace(out.String()) == "" {
		out.WriteString("[]") // its edits are its answer
	}
	proposals, mended, err := proposalsFrom(out.String(), req.Role.Intentions...)
	call.Mended = mended
	if err != nil {
		return call, fmt.Errorf("%w: %v", ErrInvalidOutput, err)
	}
	return call, os.WriteFile(filepath.Join(req.RunDir, "out", "intentions.yaml"), proposals, 0o644)
}
