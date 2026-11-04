package simulator

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/JN0V/workline/internal/agent"
)

type outcome string

// The outcomes a fake answers with. A malformed answer is a machine
// error the agent made: an answer the engine cannot read.
const (
	progress     outcome = "progress"
	waiting      outcome = "waiting"
	machineError outcome = "machine-error"
	malformed    outcome = "malformed"
)

func (o outcome) machine() bool { return o == machineError || o == malformed }

// simAgent is the fake agent: one state across every tick, its outcomes
// a script, call by call, progress past its end.
type simAgent struct {
	script []outcome
	calls  []call
	work   string // the run's folder, a volatile part of every task
}

// call is one call of the fake: the issue its run was on, read from the
// run once it ended (attribute), never guessed from the task's text.
type call struct {
	Tick   int
	Role   string
	Issue  int
	Digest string
	Kind   outcome
	runDir string
}

func (c call) String() string {
	return fmt.Sprintf("{%d %s #%d %s %s}", c.Tick, c.Role, c.Issue, c.Digest, c.Kind)
}

// tick is the tick the agent answers in, set by the driver.
var tick int

func (a *simAgent) Propose(r agent.Request) (agent.Call, error) {
	c := agent.Call{Agent: "fake", Model: "sim", Tier: r.Role.Model.Tier}
	task, _ := os.ReadFile(filepath.Join(r.RunDir, "in", "task.md"))
	kind := progress
	if n := len(a.calls); n < len(a.script) {
		kind = a.script[n]
	}
	a.calls = append(a.calls, call{Tick: tick, Role: r.Role.Name, Digest: a.digest(task), Kind: kind, runDir: r.RunDir})
	var answer string
	switch kind {
	case machineError:
		return c, fmt.Errorf("sim: the agent's run failed")
	case malformed:
		answer = "- ready: {issue: [not, a, number\n"
	case waiting:
		answer = "- ask: {issue: ISSUE, questions: \"Which one is meant?\", why: \"two readings\"}\n"
	default:
		answer = progressOf(r.Role.Name)
	}
	answer = strings.ReplaceAll(answer, "ISSUE", issueIn(task))
	return c, os.WriteFile(filepath.Join(r.RunDir, "out", "intentions.yaml"), []byte(answer), 0o644)
}

var issueRef = regexp.MustCompile(`#(\d+)`)

// issueIn is the issue an answer names: the first the task names. Only
// the answer's text uses it; the checks read the run's own (attribute).
func issueIn(task []byte) string {
	if m := issueRef.FindSubmatch(task); m != nil {
		return string(m[1])
	}
	return "0"
}

// attribute gives the calls from n on the issue their run was on: the
// target the engine kept in the run's out/run.yaml; a run that failed
// before writing it, the one issue the line woke in that run. Neither
// leaves -1, which every check takes for a defect of the simulator.
func (a *simAgent) attribute(n int, woke map[int]string) {
	only := -1
	if len(woke) == 1 {
		for id := range woke {
			only = id
		}
	}
	for i := n; i < len(a.calls); i++ {
		a.calls[i].Issue = only
		data, err := os.ReadFile(filepath.Join(a.calls[i].runDir, "out", "run.yaml"))
		if err != nil {
			continue
		}
		var run struct {
			Target *struct {
				Kind string `yaml:"kind"`
				ID   int    `yaml:"id"`
			} `yaml:"target"`
		}
		if yaml.Unmarshal(data, &run) == nil && run.Target != nil && run.Target.Kind == "issue" {
			a.calls[i].Issue = run.Target.ID
		}
	}
}

// progressOf is a role's answer that moves the issue on; ISSUE is the
// issue's number (issueIn).
func progressOf(role string) string {
	switch role {
	case "tester":
		return "- patch:\n    file: tests/test_issue.sh\n    content: |\n      # Scenario: the fix\n      . ./lib.sh\n      [ \"$(mul 2 3)\" = 6 ]\n"
	case "developer":
		return "- patch:\n    file: lib.sh\n    content: |\n      add() { echo $(($1 + $2)); }\n      mul() { echo $(($1 * $2)); }\n"
	}
	return "- ready: {issue: ISSUE, why: \"its sections are there\"}\n"
}

// volatile are the parts of a task the simulator's digest leaves out,
// each with why: a part that changes on every run makes every input new.
var volatile = []*regexp.Regexp{
	regexp.MustCompile(`\d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ`),      // a time stamp: the clock moved, not the input
	regexp.MustCompile(`\d{8}T\d{6}\.\d+-[a-z-]+`),             // a run folder's name
	regexp.MustCompile(`(?m)^.*(?:minutes?|hours?|days?) ago`), // an age, read from the clock
}

// digest is the simulator's own digest of what the fake received: the
// oracle I2 is checked against, independent of the engine's.
func (a *simAgent) digest(task []byte) string {
	s := string(task)
	if a.work != "" {
		s = regexp.MustCompile(regexp.QuoteMeta(a.work)).ReplaceAllString(s, "<work>") // the run's own folder
	}
	for _, v := range volatile {
		s = v.ReplaceAllString(s, "")
	}
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:6])
}
