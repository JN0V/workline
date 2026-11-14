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
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/JN0V/workline/internal/agent"
)

type outcome string

// The outcomes a fake answers with. A malformed answer is a machine
// error the agent made: an answer the engine cannot read (malformed), a
// test or code that does not build (broken, #1048). A patch whose hunk
// miscounts its lines (hunk, #1075) is not: the engine applies it as its
// lines read, its counts recounted, and it moves the issue as progress
// does (#1262). A needs-input names who must give
// what is lacking: the engine, another role, a person (ADR-0063, §4),
// each in the form a role has today. A replayed answer is the trial's
// own, from the trace (traces/answers/).
const (
	progress     outcome = "progress"
	waiting      outcome = "waiting"
	machineError outcome = "machine-error"
	malformed    outcome = "malformed"
	hunk         outcome = "hunk"
	broken       outcome = "broken"
	needsEngine  outcome = "needs-engine"
	needsRole    outcome = "needs-role"
	needsPerson  outcome = "needs-person"
	// testBeside: the developer's new test file alone, beside the header
	// the Scope names, in its component's test folder (#1061, ADR-0063
	// §9): its fix is the next call's.
	testBeside outcome = "test-beside"
	replayed   outcome = "replayed"
	// replayedUnread: the trial's answer, which the engine's reader could
	// not read: a machine error, counted against the budget.
	replayedUnread outcome = "replayed-unread"
	// overBudget: the role's task over its context budget, the agent not
	// asked (#1017): a machine error.
	overBudget outcome = "over-budget"
	// refused: the tester's ask, which it may not use: its answer
	// refused whole, a machine error of its answer, retried within the
	// budget (#1124). A seeded waiting of the tester's is one.
	refused outcome = "refused"
)

func (o outcome) machine() bool {
	return o == machineError || o == malformed || o == broken || o == replayedUnread || o == overBudget || o == refused
}

func (o outcome) needs() bool { return o == needsEngine || o == needsRole || o == needsPerson }

// simAgent is the fake agent: one state across every tick, its outcomes
// a script, call by call, progress past its end; a role's own script
// (by) instead, call by call of that role; the trial's answers
// (answers), a role's in the order it gave them, before any script.
type simAgent struct {
	script  []outcome
	by      map[string][]outcome
	answers map[string][]string
	calls   []call
	work    string // the run's folder, a volatile part of every task
	// tokens: the fresh tokens each call of a role says it spent, none
	// for a role not named: a trace whose every run crosses its target
	// (ADR-0064, §5).
	tokens map[string]int
}

// next is the outcome of the role's next call, and the trial's answer
// when it is one replayed.
func (a *simAgent) next(role string) (outcome, string) {
	if q := a.answers[role]; len(q) > 0 {
		a.answers[role] = q[1:]
		return replayed, q[0]
	}
	if s, ok := a.by[role]; ok {
		mine := 0
		for _, c := range a.calls {
			if c.Role == role && c.Kind != replayed && c.Kind != replayedUnread {
				mine++
			}
		}
		if mine < len(s) {
			return s[mine], ""
		}
		return progress, ""
	}
	n := 0
	for _, c := range a.calls {
		if !c.unscripted {
			n++
		}
	}
	if n < len(a.script) {
		return a.script[n], ""
	}
	return progress, ""
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
	// unscripted: a call the script does not choose, the tester's judge
	// of a broken build: it takes no turn of the seeded script, so a
	// seed plays the same outcomes as before that judge was asked.
	unscripted bool
}

func (c call) String() string {
	return fmt.Sprintf("{%d %s #%d %s %s}", c.Tick, c.Role, c.Issue, c.Digest, c.Kind)
}

// tick is the tick the agent answers in, set by the driver.
var tick int

func (a *simAgent) Propose(r agent.Request) (agent.Call, error) {
	c := agent.Call{Agent: "fake", Model: "sim", Tier: r.Role.Model.Tier, TokensIn: a.tokens[r.Role.Name]}
	task, _ := os.ReadFile(filepath.Join(r.RunDir, "in", "task.md"))
	if d := os.Getenv("SIM_DUMP"); d != "" {
		_ = os.WriteFile(filepath.Join(d, fmt.Sprintf("%d-%s.md", len(a.calls), r.Role.Name)), task, 0o644)
	}
	if r.Role.Name == "judge" && strings.Contains(string(task), "Their build broke there") {
		// The tester's judge of a broken build (ADR-0065), as a real one
		// reads it: red when each compiler line names the fix the issue
		// adds (fixed_<n>, fixed<n>), the test's own mistake otherwise.
		note := "yes: each error names the fix the issue adds."
		errs, _, _ := strings.Cut(string(task)[strings.Index(string(task), "## The build's errors"):], "\nThe output")
		for _, l := range strings.Split(errs, "\n") {
			if strings.Contains(l, ": ") && !strings.Contains(l, "fixed") {
				note = "no: " + strings.NewReplacer(`"`, `'`, `\`, `/`).Replace(strings.TrimSpace(l))
				break
			}
		}
		a.calls = append(a.calls, call{Tick: tick, Role: r.Role.Name, Digest: a.digest(received(r.RunDir, task)), Kind: progress, runDir: r.RunDir, unscripted: true})
		return c, os.WriteFile(filepath.Join(r.RunDir, "out", "intentions.yaml"), []byte("- note: \""+note+"\"\n"), 0o644)
	}
	kind, answer := a.next(r.Role.Name)
	if kind == replayed {
		return a.replay(r, task, answer)
	}
	kind, answer = a.answerOf(kind, r.Role.Name, task)
	a.calls = append(a.calls, call{Tick: tick, Role: r.Role.Name, Digest: a.digest(received(r.RunDir, task)), Kind: kind, runDir: r.RunDir})
	if kind == machineError {
		return c, fmt.Errorf("sim: the agent's run failed")
	}
	if kind == overBudget {
		return c, fmt.Errorf("%w: sim: the task is over the role's context budget", agent.ErrOverBudget)
	}
	answer = strings.ReplaceAll(answer, "ISSUE", issueIn(task))
	return c, os.WriteFile(filepath.Join(r.RunDir, "out", "intentions.yaml"), []byte(answer), 0o644)
}

// unused are the trial's answers no call took, by role.
func (a *simAgent) unused() map[string]int {
	out := map[string]int{}
	for role, q := range a.answers {
		if len(q) > 0 {
			out[role] = len(q)
		}
	}
	return out
}

// replay gives the trial's answer through the engine's own reader, as a
// command agent printing it: a code fence taken off, an answer it cannot
// read a machine error (replayedUnread), as the trial's agent's was.
func (a *simAgent) replay(r agent.Request, task []byte, answer string) (agent.Call, error) {
	file := filepath.Join(r.RunDir, "replayed-answer.txt")
	if err := os.WriteFile(file, []byte(answer), 0o644); err != nil {
		return agent.Call{}, err
	}
	reader, err := agent.Parse("cmd:cat " + strconv.Quote(file))
	if err != nil {
		return agent.Call{}, err
	}
	c, err := reader.Propose(r)
	kind := replayed
	if err != nil {
		kind = replayedUnread
	}
	a.calls = append(a.calls, call{Tick: tick, Role: r.Role.Name, Digest: a.digest(received(r.RunDir, task)), Kind: kind, runDir: r.RunDir})
	return c, err
}

// answerOf is a role's answer of kind, and the kind it is. A role with
// no form today for a kind answers its progress, said as progress: no
// property holds it to a kind it never gave.
func (a *simAgent) answerOf(kind outcome, role string, task []byte) (outcome, string) {
	switch {
	case kind == machineError || kind == overBudget:
		return kind, ""
	case kind == malformed:
		return kind, "- ready: {issue: [not, a, number\n"
	case kind == waiting && role == "tester" || kind == refused:
		return refused, ask
	case kind == waiting:
		return kind, ask
	case kind == hunk && role == "tester":
		// #1075: a new file's hunk counting fewer lines than it adds.
		file, content := a.testOf(task)
		return kind, diffNew(file, content, -3) + coversOf(task, strings.HasSuffix(file, ".go"))
	case kind == hunk && role == "developer":
		if file, code, ok := a.goCode(task); ok {
			// On a Go issue, in the file its progress writes: within its
			// bounds, the miscount the only fault.
			return kind, diffWhole(file, code, code+goFix, -2)
		}
		h := headerIn(task, "## What you may change")
		old, _ := os.ReadFile(filepath.Join(a.work, "repo", h))
		return kind, diffWhole(h, string(old), string(old)+fix, -2)
	case kind == broken && role == "tester":
		// #1048: a test calling a constructor that does not exist, an
		// error in the standard library's header, not a missing name.
		file, content := a.testOf(task)
		content = strings.Replace(content, "#include <unity.h>\n", "#include <unity.h>\n#include <memory>\n#include <string>\n", 1)
		content = strings.Replace(content, "\nint main() {", "\nvoid test_wrong_call(void) { auto made = std::make_unique<std::string>(std::string(), \"Kitchen\", 1, 2, 3); TEST_ASSERT_NOT_NULL(made.get()); }\n\nint main() {", 1)
		if strings.HasSuffix(file, ".go") {
			// In Go, a value of the wrong type: no missing name either.
			content += "\nfunc TestWrongCall(t *testing.T) {\n\tvar n int = \"Kitchen\"\n\t_ = n\n}\n"
		}
		// Its scenarios declared, as progress's (ADR-0065, change 4): the
		// build is what breaks, never the answer's shape.
		return kind, patch(file, content) + coversOf(task, strings.HasSuffix(file, ".go"))
	case kind == broken && role == "developer":
		if file, code, ok := a.goCode(task); ok {
			return kind, patch(file, code+"\nfunc fixedISSUE( bool { return true }\n")
		}
		h := headerIn(task, "## What you may change")
		old, _ := os.ReadFile(filepath.Join(a.work, "repo", h))
		return kind, patch(h, string(old)+"\nnamespace DomoticsCore {\ninline bool fixed_ISSUE( { return true; }\n}  // namespace DomoticsCore\n")
	case kind == needsEngine && role == "product-owner":
		// The trial's product owner with no code given: the folder it
		// names, its files given at the next run (trial-35's first answer).
		// On a Go issue, its package's file, as the tester and the
		// developer pick it (#1262): never the C++ fixture's header.
		h := headerIn(task, "## Scope")
		if _, ok := headerFound(task, "## Scope"); !ok {
			if d, ok := a.goDir(task, "## Scope"); ok {
				h, _ = a.goFile(d)
			}
		}
		firstLine := func(file string) string {
			code, _ := os.ReadFile(filepath.Join(a.work, "repo", file))
			for _, l := range strings.Split(string(code), "\n") {
				if l = strings.TrimSpace(l); len(l) > 12 && !strings.HasPrefix(l, "#") {
					return l
				}
			}
			return ""
		}
		line := firstLine(h)
		named := []string{h}
		if line != "" && strings.Contains(string(task), line) {
			// The file the Scope names is given already: naming it alone
			// is no need for code (#1440). A source file the task did not
			// name is the one it lacks, named beside it.
			root := filepath.Join(a.work, "repo")
			exts := []string{".go", ".h", ".cpp"}
			if _, ok := headerFound(task, "## Scope"); !ok {
				exts = []string{".mod"} // a Go issue: never a header, which would make it C++'s
			}
			_ = filepath.WalkDir(root, func(g string, e os.DirEntry, _ error) error {
				rel, err := filepath.Rel(root, g)
				if err != nil || e.IsDir() || len(named) > 1 || !slices.Contains(exts, filepath.Ext(g)) || strings.HasSuffix(g, "_test.go") || strings.HasPrefix(rel, ".") || strings.Contains(string(task), filepath.ToSlash(rel)) {
					return nil
				}
				named = append(named, filepath.ToSlash(rel))
				return nil
			})
		}
		return kind, fmt.Sprintf("- sources: {issue: ISSUE, sources: [%s], quote: {path: %q, text: %q}, why: \"no code was given: the file the title names\"}\n", strings.Join(slices.Collect(func(yield func(string) bool) {
			for _, n := range named {
				if !yield(fmt.Sprintf("%q", n)) {
					return
				}
			}
		}), ", "), h, line)
	case kind == needsEngine && role == "tester":
		// A tester short of code (#1047, #1059): the header the Scope
		// names, said lacking, never "no test possible".
		return kind, fmt.Sprintf("- note: \"needs: %s — the code the scenarios call is not in the task\"\n", headerIn(task, "## Scope"))
	case kind == needsRole && role == "tester":
		// No test possible: the product owner decides.
		return kind, "- note: \"proof: none — the scenario reads a person's screen, no test can see it\"\n"
	case kind == testBeside && role == "developer":
		h := headerIn(task, "## What you may change")
		return kind, patch(componentOf(h)+"/test/test_more_ISSUE/test_more_ISSUE.cpp",
			fmt.Sprintf("#include <unity.h>\n#include <DomoticsCore/%s>\n\nvoid test_more(void) { TEST_ASSERT_TRUE(true); }\n\nint main() {\n    UNITY_BEGIN();\n    RUN_TEST(test_more);\n    return UNITY_END();\n}\n", path.Base(h)))
	case kind == needsRole && role == "developer":
		// Its code, and a file beyond its Scope the code needs: both
		// applied, the file listed (ADR-0065, change 2); no one asked.
		return kind, a.progressOf(role, task) + patch(outside(headerIn(task, "## What you may change")), "#pragma once\n// what the fix needs here\n")
	case kind == needsPerson && role == "product-owner":
		return kind, "- ask: {issue: ISSUE, questions: \"Which of the two screens shows it?\", why: \"only the reporter saw it\"}\n"
	}
	return progress, a.progressOf(role, task)
}

// ask is a question to a person: the product owner's wait. The tester
// may not ask: its ask is refused, a machine error (refused).
const ask = "- ask: {issue: ISSUE, questions: \"Which one is meant?\", why: \"two readings\"}\n"

// fix is the developer's code: what the tester's tests call.
const fix = "\nnamespace DomoticsCore {\ninline bool fixed_ISSUE() { return true; }\n}  // namespace DomoticsCore\n"

// outside is a header of another component than h's: outside its Scope.
func outside(h string) string {
	if componentOf(h) == "DomoticsCore-Wifi" {
		return "DomoticsCore-NTP/include/DomoticsCore/NTP.h"
	}
	return "DomoticsCore-Wifi/include/DomoticsCore/Wifi.h"
}

// diffNew is a patch adding file whole as a unified diff, its hunk
// header off by off lines.
func diffNew(file, content string, off int) string {
	lines := strings.Split(strings.TrimSuffix(content, "\n"), "\n")
	var b strings.Builder
	fmt.Fprintf(&b, "--- /dev/null\n+++ b/%s\n@@ -0,0 +1,%d @@\n", file, len(lines)+off)
	for _, l := range lines {
		b.WriteString("+" + l + "\n")
	}
	return diffAnswer(b.String())
}

// diffWhole is a patch replacing file's old content by its new one as a
// unified diff, the new count off by off lines.
func diffWhole(file, old, new string, off int) string {
	o := strings.Split(strings.TrimSuffix(old, "\n"), "\n")
	n := strings.Split(strings.TrimSuffix(new, "\n"), "\n")
	var b strings.Builder
	fmt.Fprintf(&b, "--- a/%s\n+++ b/%s\n@@ -1,%d +1,%d @@\n", file, file, len(o), len(n)+off)
	for _, l := range o {
		b.WriteString("-" + l + "\n")
	}
	for _, l := range n {
		b.WriteString("+" + l + "\n")
	}
	return diffAnswer(b.String())
}

// diffAnswer is a patch answer holding a diff, as a block scalar.
func diffAnswer(diff string) string {
	var b strings.Builder
	b.WriteString("- patch: |\n")
	for _, l := range strings.Split(strings.TrimSuffix(diff, "\n"), "\n") {
		b.WriteString("    " + l + "\n")
	}
	return b.String()
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
// An issue whose Scope names no header but a Go package of the fixture,
// as workline's own: the same in Go, fixedISSUE.
func (a *simAgent) progressOf(role string, task []byte) string {
	switch role {
	case "tester":
		file, code := a.testOf(task)
		return patch(file, code) + coversOf(task, strings.HasSuffix(file, ".go"))
	case "developer":
		if file, code, ok := a.goCode(task); ok {
			return patch(file, code+goFix)
		}
		h := headerIn(task, "## What you may change")
		code, _ := os.ReadFile(filepath.Join(a.work, "repo", h))
		return patch(h, string(code)+fix)
	}
	return productOwnerOf(task)
}

// goFix is the developer's code on a Go issue: what its tests call.
const goFix = "\n// fixedISSUE is the fix.\nfunc fixedISSUE() bool { return true }\n"

// goCode is the Go file the developer's code goes in, and its content,
// when the task's bounds name no header but a Go package of the fixture.
func (a *simAgent) goCode(task []byte) (string, string, bool) {
	if _, ok := headerFound(task, "## What you may change"); ok {
		return "", "", false
	}
	d, ok := a.goDir(task, "## What you may change")
	if !ok {
		return "", "", false
	}
	file, code := a.goFile(d)
	return file, code, true
}

// goPath is a path a task names in backticks.
var goPath = regexp.MustCompile("`([\\w./-]+)`")

// goDir is the first Go package of the fixture the task names from its
// section from on: a folder, or a Go file's.
func (a *simAgent) goDir(task []byte, from string) (string, bool) {
	i := strings.Index(string(task), from)
	if i < 0 {
		return "", false
	}
	for _, m := range goPath.FindAllSubmatch(task[i:], -1) {
		p := strings.TrimSuffix(string(m[1]), "/")
		if strings.HasPrefix(p, "DomoticsCore") {
			continue
		}
		if strings.HasSuffix(p, ".go") {
			p = path.Dir(p)
		}
		if gos, _ := filepath.Glob(filepath.Join(a.work, "repo", p, "*.go")); len(gos) > 0 {
			return p, true
		}
	}
	return "", false
}

// goFile is a Go package's first file but its tests, and its content.
func (a *simAgent) goFile(dir string) (string, string) {
	gos, _ := filepath.Glob(filepath.Join(a.work, "repo", dir, "*.go"))
	gos = slices.DeleteFunc(gos, func(f string) bool { return strings.HasSuffix(f, "_test.go") })
	if len(gos) == 0 {
		return dir + "/fix.go", "package " + path.Base(dir) + "\n"
	}
	data, _ := os.ReadFile(gos[0])
	return path.Join(dir, filepath.Base(gos[0])), string(data)
}

var goPackage = regexp.MustCompile(`(?m)^package (\w+)`)

// testOf is the tester's new suite: its file and its content.
func (a *simAgent) testOf(task []byte) (string, string) {
	if _, ok := headerFound(task, "## Scope"); !ok {
		if d, ok := a.goDir(task, "## Scope"); ok {
			_, code := a.goFile(d)
			pkg := "main"
			if m := goPackage.FindStringSubmatch(code); m != nil {
				pkg = m[1]
			}
			var b strings.Builder
			fmt.Fprintf(&b, "package %s\n\nimport \"testing\"\n", pkg)
			for i, sc := range scenariosIn(task) {
				fmt.Fprintf(&b, "\n// Scenario: %s\nfunc TestScenario%d(t *testing.T) {\n\tif !fixedISSUE() {\n\t\tt.Fatal(\"not fixed\")\n\t}\n}\n", sc, i+1)
			}
			return d + "/issue_ISSUE_test.go", b.String()
		}
	}
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
	return componentOf(h) + "/test/test_issue_ISSUE/test_issue_ISSUE.cpp", b.String()
}

// coversOf declares the tester's test of each scenario, in the order
// testOf writes them (ADR-0065, change 4).
func coversOf(task []byte, goTest bool) string {
	var b strings.Builder
	for i, sc := range scenariosIn(task) {
		name := fmt.Sprintf("test_scenario_%d", i+1)
		if goTest {
			name = fmt.Sprintf("TestScenario%d", i+1)
		}
		fmt.Fprintf(&b, "- covers: {scenario: %q, tests: [%q]}\n", sc, name)
	}
	return b.String()
}

// productOwnerOf is the product owner's answer that moves the issue on.
func productOwnerOf(task []byte) string {
	if m := sentBack.FindSubmatch(task); m != nil {
		// The developer's code needs files outside the Scope (ADR-0063,
		// §9): the Scope widened with them, its own kept, then ready.
		files := strings.Split(string(m[1]), "`, `")
		scope := strings.TrimSpace(sectionOf(task, "## Scope"))
		for _, f := range files {
			scope += fmt.Sprintf("\n- `%s`: what the developer's code needs.", f)
		}
		q := make([]string, len(files))
		for i, f := range files {
			q[i] = strconv.Quote(f)
		}
		return fmt.Sprintf("- refine: {issue: ISSUE, scope: %q, sources: [%s], why: \"the developer's code needs them\"}\n- ready: {issue: ISSUE, why: \"its sections are there\"}\n",
			strings.TrimSpace(scope), strings.Join(q, ", "))
	}
	switch {
	case strings.Contains(string(task), "), on its size. What crossed:") && strings.Contains(string(task), "`size` with `tier: large`"):
		// A run too big for its role, the issue atomic as the fake sees
		// it (ADR-0064, §3): `large` granted, sent back where it was.
		to := "refine"
		if m := unblockTo.FindSubmatch(task); m != nil {
			to = string(m[1])
		}
		return "- size: {issue: ISSUE, tier: large, why: \"one need, no part of it passes alone\"}\n- unblock: {issue: ISSUE, to: " + to + ", why: \"granted a larger size\"}\n"
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

// sentBack is the product owner's task on an issue the developer sent
// back to refine for files outside its Scope: the files.
var sentBack = regexp.MustCompile("Sent back to refine by the developer: its code needs `([^\\n]+?)`, outside")

// sectionOf is the text of the task's section from, up to the next
// heading of its level or above.
func sectionOf(task []byte, from string) string {
	_, after, ok := strings.Cut(string(task), from+"\n")
	if !ok {
		return ""
	}
	if i := regexp.MustCompile(`(?m)^#{1,2} `).FindStringIndex(after); i != nil {
		after = after[:i[0]]
	}
	return after
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

// headerFound is the first header the task's section from names, and
// whether there is one: what the product owner wrote there, not the code
// the picker gave below it, found by the names the issue uses, in any
// file of the project (ADR-0063 §8).
func headerFound(task []byte, from string) (string, bool) {
	if i := strings.Index(string(task), from); i >= 0 {
		section := task[i+len(from):]
		if j := strings.Index(string(section), "\n## "); j >= 0 {
			section = section[:j]
		}
		if m := headerRe.Find(section); m != nil {
			return string(m), true
		}
	}
	return "", false
}

// headerIn is the first header the task's section from names, else from
// that section on, else anywhere, else the core's: the tester and the
// developer pick the same one from the Scope the product owner wrote.
func headerIn(task []byte, from string) string {
	if h, ok := headerFound(task, from); ok {
		return h
	}
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
	// The section alone: a later one (a Verification, the feedback) may
	// list bold names that are none of the issue's scenarios.
	if i := strings.Index(after, "\n## "); i >= 0 {
		after = after[:i]
	}
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

// received is what an agent's prompt is made of: the task, then the
// feedback a run asked again in place gives it (agent/prompt.go).
func received(runDir string, task []byte) []byte {
	fb, err := os.ReadFile(filepath.Join(runDir, "out", "feedback.md"))
	if err != nil {
		return task
	}
	return append(append(slices.Clone(task), "\n\n"...), fb...)
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
