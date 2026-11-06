package tester

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/verdict"
	"github.com/JN0V/workline/internal/work"
)

// askedAgainChars is the room kept in the prompt for an answer asked
// again, as the product owner and the reviewer keep it.
const askedAgainChars = 2000

// fitTask is the task build gives with code-lines-max lines of code, or
// fewer when the prompt would be over the tester's context budget:
// code-lines-max counts lines, a budget counts tokens, and long lines fill
// it first (#1017). It shrinks the lines a quarter at a time, down to one;
// a task that still does not fit is given as is, and the engine refuses
// it, said on the issue. lines are the lines of code it was built with.
func fitTask(runDir, repo string, s Settings, build func(Settings) string) (task string, lines int) {
	task = build(s)
	lines = s.CodeLinesMax
	if lines <= 0 {
		lines = 1500
	}
	fits := fitsBudget(runDir, repo)
	if fits == nil {
		return task, lines
	}
	for !fits(task) && lines > 1 {
		lines = max(lines*3/4, 1)
		less := s
		less.CodeLinesMax = lines
		task = build(less)
	}
	return task, lines
}

// sayCuts says, for the person, what the task built with lines of code
// cut of the code the issue names (#1159): cuts, as writeCode gave them;
// code-lines-max lowered to fit the context budget said too, the setting
// to change then the role's budget, not code-lines-max.
func sayCuts(runDir string, s Settings, lines int, cuts []string) error {
	if len(cuts) == 0 {
		return nil
	}
	set := s.CodeLinesMax
	if set <= 0 {
		set = 1500
	}
	raise := "raise " + verdict.Setting("tester", "code-lines-max")
	if lines < set {
		raise = fmt.Sprintf("code-lines-max (%d) was lowered to %d lines to fit the tester's context budget: raise context.budget in its role.yaml, or narrow the issue's %s", set, lines, work.Name(work.Scope))
	}
	return verdict.SayCut(runDir, fmt.Sprintf("the code given to the tester's agent is cut past code-lines-max (%d lines): %s; %s", lines, strings.Join(cuts, "; "), raise))
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
