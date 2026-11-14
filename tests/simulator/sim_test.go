// Package simulator runs the line over many ticks, in process, on frozen
// real traces and a seeded fake agent, and checks after each tick the
// properties of ADR-0063 (step 1); docs/spec/conformance.md says how.
package simulator

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/clock"
	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/line"
	"github.com/JN0V/workline/internal/routing"
)

// start is the simulator's first tick, on its own clock: the evening the
// trial's round 3 ended.
var start = time.Date(2026, 11, 4, 20, 0, 0, 0, time.UTC)

// period is the simulated time between two reconciles: K ticks cover
// a blocked issue's week before it is read again.
const period = 8 * time.Hour

// The budgets K is computed from (ADR-0063, §5): the machine-error
// budget and the judgement-rounds cap, today's defaults.
const (
	budget = 2
	rounds = 3
)

// trace is one frozen trace: a fake forge's file in traces/; fresh, its
// issue taken back to how it was opened, no state, no comment; end, where
// an agent that always progresses must leave it: "done", or "waiting:"
// and what it waits on.
type trace struct {
	name  string
	file  string
	fresh bool
	end   string
	// by is a role's outcomes, call by call of that role, progress past
	// them: a case of one answer kind on a real issue.
	by map[string][]outcome
	// replay: the trial's own answers are given first, from
	// traces/answers/<file>/, a role's in the order it gave them.
	replay bool
	// tokens: the fresh tokens each call of a role says it spent.
	tokens map[string]int
}

// traces are the trial's issues as round 3 left them on 2026-11-04 (#757),
// and the five it merged taken back to how they were opened, the path
// that works. Each blocked one was blocked by the machine, not by a
// person's question (pending.txt says how): an agent that progresses
// takes every one to Done. Then workline's own eleventh run (#757,
// 2026-11-04), its three bugs merged with no person, as they were opened;
// the answer kinds on the trial's issues, each named for what it plays;
// the trial's answers replayed on each issue, as it was opened.
var traces = func() []trace {
	var ts []trace
	trial := []int{3, 4, 5, 16, 18, 19, 20, 33, 34, 35, 36, 37}
	for _, n := range trial {
		f := fmt.Sprintf("trial-%d", n)
		ts = append(ts, trace{name: f, file: f, end: "done"})
	}
	for _, n := range []int{3, 4, 5, 16, 18} {
		f := fmt.Sprintf("trial-%d", n)
		ts = append(ts, trace{name: f + "-fresh", file: f, fresh: true, end: "done"})
	}
	for _, n := range []int{1002, 1003, 1004} {
		f := fmt.Sprintf("workline-%d", n)
		ts = append(ts, trace{name: f, file: f, fresh: true, end: "done"})
	}
	one := func(role string, o outcome) map[string][]outcome { return map[string][]outcome{role: {o}} }
	ts = append(ts,
		// #1048: the tester's test calls a constructor the code lacks, on
		// #20 as in the trial; on #3, whose Scope names its file; then
		// the developer's code that does not build.
		trace{name: "trial-20-broken-test", file: "trial-20", fresh: true, end: "done", by: one("tester", broken)},
		trace{name: "trial-3-broken-test", file: "trial-3", fresh: true, end: "done", by: one("tester", broken)},
		trace{name: "trial-3-broken-code", file: "trial-3", fresh: true, end: "done", by: one("developer", broken)},
		// #1075: a new file's hunk miscounted, the tester's; then the
		// developer's, on the header it changes. On #3: #37 as it was
		// opened has no Need the fake can ready.
		trace{name: "trial-3-hunk-test", file: "trial-3", fresh: true, end: "done", by: one("tester", hunk)},
		trace{name: "trial-3-hunk-code", file: "trial-3", fresh: true, end: "done", by: one("developer", hunk)},
		// The developer's agent run failing once (seed 26 of the first
		// seeded runs on these cases, frozen).
		trace{name: "trial-3-error-code", file: "trial-3", fresh: true, end: "done", by: one("developer", machineError)},
		// #1017: the tester's task over its context budget, its agent not
		// asked: read as "no test proposed".
		trace{name: "trial-35-over-budget", file: "trial-35", fresh: true, end: "done", by: one("tester", overBudget)},
		// #1124: the tester's ask, which it may not use: refused whole,
		// run again with the refusal said, then Done.
		trace{name: "trial-35-tester-refused", file: "trial-35", fresh: true, end: "done", by: one("tester", refused)},
		// The product owner's answer unreadable twice in its run.
		trace{name: "trial-35-po-unread", file: "trial-35", fresh: true, end: "done", by: map[string][]outcome{"product-owner": {malformed, malformed}}},
		// Needs input, from each source: the engine (the product owner's
		// files), the product owner (the tester's no test), a person (the
		// product owner's question, unanswered: the issue waits on its
		// reporter). The developer's file beyond its Scope asks no one:
		// applied, listed (ADR-0065, change 2).
		trace{name: "trial-35-needs-engine", file: "trial-35", fresh: true, end: "done", by: one("product-owner", needsEngine)},
		trace{name: "trial-35-needs-po-test", file: "trial-35", fresh: true, end: "done", by: one("tester", needsRole)},
		// #1047, #1059: the tester short of code says what it lacks.
		trace{name: "trial-35-tester-needs", file: "trial-35", fresh: true, end: "done", by: one("tester", needsEngine)},
		trace{name: "trial-35-needs-po-code", file: "trial-35", fresh: true, end: "done", by: one("developer", needsRole)},
		trace{name: "trial-35-needs-person", file: "trial-35", fresh: true, end: "waiting:", by: one("product-owner", needsPerson)},
		// #1061: the developer's new test file beside its Scope, alone,
		// as the trial's round wrote it: beside the Scope, no ask.
		trace{name: "trial-35-test-beside", file: "trial-35", fresh: true, end: "done", by: one("developer", testBeside)},
		// ADR-0064, step 5: every run crosses its target. The tester's
		// first, 350k past its cap of 300k, its answer unread, crosses its
		// size: the product owner, asked, grants `large`; then each run
		// 350k, the developer's 600k, past its cap of 500k: the issue
		// reaches its backstop, 1.6M, and waits for a person, within the
		// hard caps.
		trace{name: "trial-35-every-run-crosses", file: "trial-35", fresh: true, end: "waiting:", by: one("tester", malformed),
			tokens: map[string]int{"product-owner": 350000, "tester": 350000, "developer": 600000}},
		// #1262, seed 36: the tester's test that does not build, judged
		// its own mistake; an answer not read; then a new file's hunk
		// miscounted, its retry at the next tick.
		trace{name: "trial-18-hunk-retried", file: "trial-18", fresh: true, end: "done", by: map[string][]outcome{"tester": {broken, malformed, hunk}}},
		// #1262, seed 39: the product owner's needs-input on a Go issue,
		// the file of the package its Scope names given at once.
		trace{name: "workline-1004-needs-engine", file: "workline-1004", fresh: true, end: "done", by: one("product-owner", needsEngine)},
		// #1271: the developer's code that does not build on a Go issue:
		// its pull request a Draft, the issue waits on it, said; a change
		// to its body is the developer's review round's.
		trace{name: "workline-1004-broken-code", file: "workline-1004", fresh: true, end: "waiting:a Draft", by: one("developer", broken)},
	)
	for _, n := range trial {
		f := fmt.Sprintf("trial-%d", n)
		ts = append(ts, trace{name: f + "-replay", file: f, fresh: true, end: "done", replay: true})
	}
	return ts
}()

// config is the trial's line: the product owner, the tester and the
// developer, each woken by an issue's state; the tester's and the
// developer's settings the trial copy's, its g++ error shapes; go's for
// the Go packages workline's own issues name.
const config = `roles:
  tester:
    settings:
      test-command: tools/workline-tests.sh {report}
      code-lines-max: 1000
      test-paths: ["DomoticsCore-*/test/**", "**/*_test.go"]
      build-errors:
        - error: '^(?P<file>[^\s:]+\.go):\d+:\d+: (?P<message>.+)$'
          missing-name: ['^undefined: \w+']
          test-name: '(?m)^func\s+(Test\w+)\s*\('
        - error: '^(?P<file>[^\s:]+\.(?:cpp|cc|c|h|hpp)):\d+:\d+: error: (?P<message>.+)$'
          missing-name:
            - "^[‘'][\\w:]+[’'] was not declared in this scope"
            - "^[‘'][\\w:]+[’'] has not been declared"
            - "^[‘'][\\w:]+[’'] is not a member of [‘'][\\w:]+[’']"
            - "^[‘'](?:const )?(?:class|struct) [^’']+[’'] has no member named [‘']\\w+[’']"
          test-name: '(?m)^(?:static\s+)?void\s+(test_\w+)\s*\(\s*(?:void)?\s*\)'
  developer:
    settings:
      proof-command: tools/workline-tests.sh {report} {packages}
routing:
  events:
    issue: [product-owner, tester, developer]
`

// hermetic: no machine's git config, no user's lists, no CI's variables.
func hermetic(t *testing.T, work string) {
	for _, kv := range os.Environ() {
		if name, _, _ := strings.Cut(kv, "="); strings.HasPrefix(name, "WORKLINE_") || name == "CI" || name == "GITHUB_ACTIONS" || name == "GITLAB_CI" {
			t.Setenv(name, "")
		}
	}
	for k, v := range map[string]string{
		"GIT_CONFIG_GLOBAL": "/dev/null", "GIT_CONFIG_NOSYSTEM": "1",
		"GIT_AUTHOR_NAME": "Fixture", "GIT_AUTHOR_EMAIL": "fixture@example.invalid",
		"GIT_COMMITTER_NAME": "Fixture", "GIT_COMMITTER_EMAIL": "fixture@example.invalid",
		"GIT_AUTHOR_DATE": "2026-01-03T00:00:00Z", "GIT_COMMITTER_DATE": "2026-01-03T00:00:00Z",
		"XDG_CONFIG_HOME": filepath.Join(work, "config"), "WORKLINE_MODELS_SEEN": filepath.Join(work, "models-seen.yaml"),
		"WORKLINE_SUITE_CACHE": suiteCache,
	} {
		t.Setenv(k, v)
	}
}

// repoTemplate is the C++ project every trace runs on, built once, with
// its remote beside it, origin.git: where the roles push their pull
// requests' branches.
var repoTemplate string

// suiteCache is the folder where the fixture's script keeps its built C++
// suites by the content of their sources, shared by every trace.
var suiteCache string

func TestMain(m *testing.M) {
	// With no g++, nothing here ran: said, and never a pass (principle 12).
	if _, err := exec.LookPath("g++"); err != nil {
		fmt.Fprintln(os.Stderr, "the simulator did not run: its C++ fixture needs g++, not found on PATH")
		os.Exit(1)
	}
	dir, err := os.MkdirTemp("", "workline-sim-")
	if err != nil {
		panic(err)
	}
	repoTemplate = filepath.Join(dir, "repo")
	suiteCache = filepath.Join(dir, "suites")
	if err := os.MkdirAll(suiteCache, 0o755); err != nil {
		panic(err)
	}
	if err := fixtureRepo(repoTemplate); err != nil {
		panic(err)
	}
	// The role scripts run the engine's binary: one built from this tree,
	// not this test's.
	bin := filepath.Join(dir, "workline")
	if out, err := exec.Command("go", "build", "-o", bin, "../../cmd/workline").CombinedOutput(); err != nil {
		panic(fmt.Sprintf("the engine: %v\n%s", err, out))
	}
	engine.Self = func() string { return bin }
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

// world is one trace's run: its repository, its forge file, its agent.
type world struct {
	repo, forge, roles string
	cfg                *routing.Config
	agent              *simAgent
}

func newWorld(t *testing.T, tr trace, a *simAgent) *world {
	work := t.TempDir()
	hermetic(t, work)
	repo := filepath.Join(work, "repo")
	for _, d := range []string{"repo", "origin.git"} {
		if out, err := exec.Command("cp", "-a", filepath.Join(filepath.Dir(repoTemplate), d), filepath.Join(work, d)).CombinedOutput(); err != nil {
			t.Fatalf("the repository: %v\n%s", err, out)
		}
	}
	if err := gitIn(repo, "remote", "set-url", "origin", filepath.Join(work, "origin.git")); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join("traces", tr.file+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var s forge.FakeState
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatal(err)
	}
	for i := range s.Issues {
		is := &s.Issues[i]
		// The fix the fake agents write, named by the issue as a real
		// one names what the change adds: a missing name the issue never
		// names is a broken build (#1111).
		is.Title += fmt.Sprintf(" (fixed_%d, fixed%d)", is.ID, is.ID)
		if !tr.fresh {
			continue
		}
		is.Closed, is.Reason, is.Comments, is.LabelEvents, is.Labeled = false, "", nil, nil, nil
		// As a person puts it on the line: its kind, and refine.
		is.Labels = append(slices.DeleteFunc(is.Labels, func(l string) bool { return strings.HasPrefix(l, "workline:") }), "workline:to-refine")
		is.LabelEvents = []forge.LabelEvent{{Label: "workline:to-refine", Added: true, Writer: true,
			Note: forge.Note{Author: "JN0V", Insider: true, Created: start.Add(-time.Hour).Format(time.RFC3339)}}}
	}
	if tr.fresh {
		s.MergeRequests = nil
	}
	data, _ = json.Marshal(s)
	f := filepath.Join(work, "forge.json")
	if err := os.WriteFile(f, data, 0o644); err != nil {
		t.Fatal(err)
	}
	// The forge's reads, counted by every process the line runs.
	if err := os.WriteFile(forge.ReadsLog(f), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	a.by, a.tokens = tr.by, tr.tokens
	if tr.replay {
		a.answers = answers(t, tr.file)
	}
	roles, _ := filepath.Abs(filepath.Join("..", "..", "roles"))
	cfg, err := routing.Load(repo)
	if err != nil {
		t.Fatal(err)
	}
	a.work = work
	w := &world{repo: repo, forge: f, roles: roles, cfg: cfg, agent: a}
	w.trialBranches(t)
	return w
}

// answers are the trial's agent answers on a trace's issue, a role's in
// the order it gave them: traces/answers/<file>/<n>-<role>.txt, each
// the agent's answer a run of the trial kept (its out/agent-answer.txt),
// run by run of the round. A developer's `[]` is left out: its code was
// in its worktree, which the run did not keep; the fake writes its own.
// A <n>-<role>.txt.covers beside one is given after it.
func answers(t *testing.T, file string) map[string][]string {
	paths, err := filepath.Glob(filepath.Join("traces", "answers", file, "*.txt"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("no answers kept for %s: %v", file, err)
	}
	slices.Sort(paths)
	out := map[string][]string{}
	for _, p := range paths {
		_, role, _ := strings.Cut(strings.TrimSuffix(filepath.Base(p), ".txt"), "-")
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if role == "developer" && strings.TrimSpace(string(data)) == "[]" {
			continue
		}
		// The covers entries a tester's answer predates (ADR-0065,
		// change 4), kept beside it, given after it.
		if covers, err := os.ReadFile(p + ".covers"); err == nil {
			data = append(append(data, '\n'), covers...)
		}
		out[role] = append(out[role], string(data))
	}
	return out
}

func (w *world) state(t *testing.T) forge.FakeState {
	var s forge.FakeState
	data, err := os.ReadFile(w.forge)
	if err == nil {
		err = json.Unmarshal(data, &s)
	}
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// run is one reconcile, on the simulator's clock.
func (w *world) run(at time.Time) *line.Result {
	// The clock, for the engine and the role scripts it runs alike.
	_ = os.Setenv(clock.Fixed, at.Format(time.RFC3339))
	defer func() { _ = os.Unsetenv(clock.Fixed) }()
	defer agent.Hook("sim", w.agent)()
	before, err := w.read()
	n := len(w.agent.calls)
	res := line.RunWith(line.ReconcileEvent, engine.Options{Repo: w.repo, RolesDir: w.roles, AI: "hook:sim",
		Forge: "fake:" + w.forge, OpenMergeRequest: true}, &line.Gates{})
	w.agent.attribute(n, res.Woke)
	if err == nil {
		w.dateLabels(before, at)
	}
	return res
}

// requests are the forge's requests so far: its reads, its writes.
func (w *world) requests(t *testing.T) int {
	st, err := os.Stat(forge.ReadsLog(w.forge))
	if err != nil {
		t.Fatal(err)
	}
	return int(st.Size()) + w.state(t).Writes
}

func (w *world) read() (forge.FakeState, error) {
	var s forge.FakeState
	data, err := os.ReadFile(w.forge)
	if err == nil {
		err = json.Unmarshal(data, &s)
	}
	return s, err
}

// dateLabels dates the labels a run put on or took off, as a real forge
// keeps them and the simulated one does not: the line reads when an
// issue entered its state from them.
func (w *world) dateLabels(before forge.FakeState, at time.Time) {
	after, err := w.read()
	if err != nil {
		return
	}
	for i := range after.Issues {
		is := &after.Issues[i]
		var was []string
		if b := issueOf(before, is.ID); b != nil {
			was = b.Labels
		}
		note := forge.Note{Author: "workline-bot", Bot: true, Created: at.Format(time.RFC3339)}
		for _, l := range is.Labels {
			if !slices.Contains(was, l) {
				is.LabelEvents = append(is.LabelEvents, forge.LabelEvent{Label: l, Added: true, Note: note})
				if is.Labeled == nil {
					is.Labeled = map[string]string{}
				}
				is.Labeled[l] = note.Created
			}
		}
		for _, l := range was {
			if !slices.Contains(is.Labels, l) {
				is.LabelEvents = append(is.LabelEvents, forge.LabelEvent{Label: l, Note: note})
			}
		}
	}
	if data, err := json.Marshal(after); err == nil {
		_ = os.WriteFile(w.forge, data, 0o644)
	}
}

// k is the ticks the line has to be quiet in, from the config:
// S × (B + 1) + 2R + 2, S the states on the line (ADR-0063).
func (w *world) k() int { return len(w.cfg.States)*(budget+1) + 2*rounds + 2 }
