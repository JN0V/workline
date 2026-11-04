package simulator

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
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
	if d := os.Getenv("SIM_DUMP"); d != "" {
		_ = os.WriteFile(filepath.Join(d, fmt.Sprintf("%d-%s.md", len(a.calls), r.Role.Name)), task, 0o644)
	}
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
		answer = a.progressOf(r.Role.Name, task)
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
// issue's number (issueIn). The tester's: a new suite beside the header
// the task names first, a test a scenario, each calling the fix the code
// lacks, DomoticsCore::fixed_ISSUE: red on the base by a missing name.
// The developer's: that fix, added to the header it may change first.
func (a *simAgent) progressOf(role string, task []byte) string {
	switch role {
	case "tester":
		h := headerIn(task, "## Scope")
		var b strings.Builder
		fmt.Fprintf(&b, "#include <unity.h>\n#include <DomoticsCore/%s>\n\n", path.Base(h))
		scs := scenariosIn(task)
		for i, sc := range scs {
			fmt.Fprintf(&b, "// Scenario: %s\nvoid test_scenario_%d(void) { TEST_ASSERT_TRUE(DomoticsCore::fixed_ISSUE()); }\n\n", sc, i+1)
		}
		b.WriteString("int main() {\n    UNITY_BEGIN();\n")
		for i := range scs {
			fmt.Fprintf(&b, "    RUN_TEST(test_scenario_%d);\n", i+1)
		}
		b.WriteString("    return UNITY_END();\n}\n")
		return patch(componentOf(h)+"/test/test_issue_ISSUE/test_issue_ISSUE.cpp", b.String())
	case "developer":
		h := headerIn(task, "## What you may change")
		code, _ := os.ReadFile(filepath.Join(a.work, "repo", h))
		return patch(h, string(code)+"\nnamespace DomoticsCore {\ninline bool fixed_ISSUE() { return true; }\n}  // namespace DomoticsCore\n")
	}
	switch {
	case strings.Contains(string(task), "Blocked (workline:blocked)"):
		// Read again, what blocked it gone: back to the state it was
		// blocked from, as the task names it.
		to := "refine"
		if m := unblockTo.FindSubmatch(task); m != nil {
			to = string(m[1])
		}
		return "- unblock: {issue: ISSUE, to: " + to + ", why: \"what blocked it is gone\"}\n"
	case strings.Contains(string(task), "flaky-n-of-n"):
		// The tester found no test possible: the proof the need is met,
		// picked from the kinds the task lists.
		return "- ready: {issue: ISSUE, why: \"the suite proves it\", proof: {kind: existing-suite, reason: \"the suite runs the code it changes\"}}\n"
	}
	return "- ready: {issue: ISSUE, why: \"its sections are there\"}\n"
}

// unblockTo is the state a blocked issue's task offers it back to, when
// it names one besides refine.
var unblockTo = regexp.MustCompile("`to: (\\w[\\w-]*)`, the state it was blocked from")

// patch is an answer writing content whole to file.
func patch(file, content string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "- patch:\n    file: %s\n    content: |\n", file)
	for _, l := range strings.Split(strings.TrimSuffix(content, "\n"), "\n") {
		if l == "" {
			b.WriteString("\n")
			continue
		}
		b.WriteString("      " + l + "\n")
	}
	return b.String()
}

var (
	headerRe   = regexp.MustCompile(`\bDomoticsCore-[A-Za-z]+/include/DomoticsCore/\w+\.h\b`)
	scenarioRe = regexp.MustCompile(`(?m)^- \*\*(.+?):\*\*`)
)

// fallback is the header a task naming none gets: the core's.
const fallback = "DomoticsCore-Core/include/DomoticsCore/EventBus.h"

// headerIn is the first header the task names from its section from on,
// else anywhere, else the core's: the tester and the developer pick the
// same one from the Scope the product owner wrote.
func headerIn(task []byte, from string) string {
	if i := strings.Index(string(task), from); i >= 0 {
		if m := headerRe.Find(task[i:]); m != nil {
			return string(m)
		}
	}
	if m := headerRe.Find(task); m != nil {
		return string(m)
	}
	return fallback
}

func componentOf(header string) string { c, _, _ := strings.Cut(header, "/"); return c }

// scenariosIn are the names of the scenarios a tester's task lists, one
// at least.
func scenariosIn(task []byte) []string {
	_, after, _ := strings.Cut(string(task), "## The scenarios, one test each")
	var out []string
	for _, m := range scenarioRe.FindAllStringSubmatch(after, -1) {
		if !slices.Contains(out, m[1]) {
			out = append(out, m[1])
		}
	}
	if len(out) == 0 {
		out = []string{"the fix"}
	}
	return out
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
