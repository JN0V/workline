package tester

import (
	"os"
	"path/filepath"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/role"
)

// askedAgainChars is the room kept in the prompt for an answer asked
// again, as the product owner and the reviewer keep it.
const askedAgainChars = 2000

// fitTask is the task build gives with code-lines-max lines of code, or
// fewer when the prompt would be over the tester's context budget:
// code-lines-max counts lines, a budget counts tokens, and long lines fill
// it first (#1017). It shrinks the lines a quarter at a time, down to one;
// a task that still does not fit is given as is, and the engine refuses
// it, said on the issue.
func fitTask(runDir, repo string, s Settings, build func(Settings) string) string {
	task := build(s)
	fits := fitsBudget(runDir, repo)
	if fits == nil {
		return task
	}
	lines := s.CodeLinesMax
	if lines <= 0 {
		lines = 1500
	}
	for !fits(task) && lines > 1 {
		lines = max(lines*3/4, 1)
		less := s
		less.CodeLinesMax = lines
		task = build(less)
	}
	return task
}

// fitsBudget says whether a task keeps the agent's prompt within the
// tester's context budget, measured as the engine measures it — the same
// facets, agent.Prompt, agent.Tokens —; nil when the role or its budget is
// not known.
func fitsBudget(runDir, repo string) func(task string) bool {
	name := os.Getenv("WORKLINE_ROLE")
	if name == "" {
		name = "tester"
	}
	r, err := role.Load(os.Getenv("WORKLINE_ROLES_DIR"), name)
	if err != nil || r.Context.Budget <= 0 {
		return nil
	}
	budget := r.Context.Budget
	asked := *r.Asked()
	asked.Context.Budget = 0 // measured here, refused by the engine only
	return func(task string) bool {
		if err := os.WriteFile(filepath.Join(runDir, "in", "task.md"), []byte(task), 0o644); err != nil {
			return true
		}
		system, user, err := agent.Prompt(agent.Request{RunDir: runDir, Repo: repo, Role: &asked})
		if err != nil {
			return true // the engine says why when it asks
		}
		return agent.Tokens(len(system)+len(user)+askedAgainChars) <= budget
	}
}
