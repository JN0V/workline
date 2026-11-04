// Package simulator runs the line over many ticks, in process, on frozen
// real traces and a seeded fake agent, and checks after each tick the
// properties of ADR-0063 (step 1); docs/spec/conformance.md says how.
package simulator

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
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

// trace is one frozen trace: a fake forge's file in traces/, and fresh,
// its issue taken back to how it was opened: no state, no comment.
type trace struct {
	name  string
	file  string
	fresh bool
}

// traces are the trial's issues as round 3 left them on 2026-11-04 (#757),
// and the five it merged, taken back to how they were opened, the path
// that works.
var traces = func() []trace {
	var ts []trace
	for _, n := range []int{3, 4, 5, 16, 18, 19, 20, 33, 34, 35, 36, 37} {
		f := fmt.Sprintf("trial-%d", n)
		ts = append(ts, trace{name: f, file: f})
	}
	for _, n := range []int{3, 4, 5, 16, 18} {
		f := fmt.Sprintf("trial-%d", n)
		ts = append(ts, trace{name: f + "-fresh", file: f, fresh: true})
	}
	return ts
}()

// config is the trial's line: the product owner, the tester and the
// developer, each woken by an issue's state.
const config = `routing:
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
	} {
		t.Setenv(k, v)
	}
}

// repoTemplate is the shell project every trace runs on, built once.
var repoTemplate string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "workline-sim-")
	if err != nil {
		panic(err)
	}
	repoTemplate = filepath.Join(dir, "repo")
	if err := os.MkdirAll(repoTemplate, 0o755); err != nil {
		panic(err)
	}
	script, _ := filepath.Abs(filepath.Join("..", "conformance", "fixtures", "repos", "tested.sh"))
	cmd := exec.Command("sh", script)
	cmd.Dir = repoTemplate
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		panic(fmt.Sprintf("the repository: %v\n%s", err, out))
	}
	// The role scripts run the engine's binary: one built from this tree,
	// not this test's.
	bin := filepath.Join(dir, "workline")
	if out, err := exec.Command("go", "build", "-o", bin, "../../cmd/workline").CombinedOutput(); err != nil {
		panic(fmt.Sprintf("the engine: %v\n%s", err, out))
	}
	engine.Self = func() string { return bin }
	if err := os.MkdirAll(filepath.Join(repoTemplate, ".workline"), 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(repoTemplate, ".workline", "config.yaml"), []byte(config), 0o644); err != nil {
		panic(err)
	}
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
	if out, err := exec.Command("cp", "-a", repoTemplate, repo).CombinedOutput(); err != nil {
		t.Fatalf("the repository: %v\n%s", err, out)
	}
	data, err := os.ReadFile(filepath.Join("traces", tr.file+".json"))
	if err != nil {
		t.Fatal(err)
	}
	if tr.fresh {
		var s forge.FakeState
		if err := json.Unmarshal(data, &s); err != nil {
			t.Fatal(err)
		}
		for i := range s.Issues {
			is := &s.Issues[i]
			is.Closed, is.Reason, is.Comments, is.LabelEvents, is.Labeled = false, "", nil, nil, nil
			// As a person puts it on the line: its kind, and refine.
			is.Labels = append(slices.DeleteFunc(is.Labels, func(l string) bool { return strings.HasPrefix(l, "workline:") }), "workline:to-refine")
			is.LabelEvents = []forge.LabelEvent{{Label: "workline:to-refine", Added: true, Writer: true,
				Note: forge.Note{Author: "JN0V", Insider: true, Created: start.Add(-time.Hour).Format(time.RFC3339)}}}
		}
		s.MergeRequests = nil
		data, _ = json.Marshal(s)
	}
	f := filepath.Join(work, "forge.json")
	if err := os.WriteFile(f, data, 0o644); err != nil {
		t.Fatal(err)
	}
	roles, _ := filepath.Abs(filepath.Join("..", "..", "roles"))
	cfg, err := routing.Load(repo)
	if err != nil {
		t.Fatal(err)
	}
	a.work = work
	return &world{repo: repo, forge: f, roles: roles, cfg: cfg, agent: a}
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
	res := line.RunWith(line.ReconcileEvent, engine.Options{Repo: w.repo, RolesDir: w.roles, AI: "hook:sim",
		Forge: "fake:" + w.forge, OpenMergeRequest: true}, &line.Gates{})
	if err == nil {
		w.dateLabels(before, at)
	}
	return res
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

// ---- the fake agent ----

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

type call struct {
	Tick   int
	Role   string
	Issue  int
	Digest string
	Kind   outcome
}

// tick is the tick the agent answers in, set by the driver.
var tick int

var issueRef = regexp.MustCompile(`#(\d+)`)

func (a *simAgent) Propose(r agent.Request) (agent.Call, error) {
	c := agent.Call{Agent: "fake", Model: "sim", Tier: r.Role.Model.Tier}
	task, _ := os.ReadFile(filepath.Join(r.RunDir, "in", "task.md"))
	issue := 0
	if m := issueRef.FindSubmatch(task); m != nil {
		issue, _ = strconv.Atoi(string(m[1]))
	}
	kind := progress
	if n := len(a.calls); n < len(a.script) {
		kind = a.script[n]
	}
	a.calls = append(a.calls, call{Tick: tick, Role: r.Role.Name, Issue: issue, Digest: a.digest(task), Kind: kind})
	var answer string
	switch kind {
	case machineError:
		return c, fmt.Errorf("sim: the agent's run failed")
	case malformed:
		answer = "- ready: {issue: [not, a, number\n"
	case waiting:
		answer = fmt.Sprintf("- ask: {issue: %d, questions: \"Which one is meant?\", why: \"two readings\"}\n", issue)
	default:
		answer = progressOf(r.Role.Name, issue)
	}
	return c, os.WriteFile(filepath.Join(r.RunDir, "out", "intentions.yaml"), []byte(answer), 0o644)
}

// progressOf is a role's answer that moves the issue on.
func progressOf(role string, issue int) string {
	switch role {
	case "tester":
		return fmt.Sprintf("- patch:\n    file: tests/test_issue_%d.sh\n    content: |\n      # Scenario: the fix\n      . ./lib.sh\n      [ \"$(mul 2 3)\" = 6 ]\n", issue)
	case "developer":
		return "- patch:\n    file: lib.sh\n    content: |\n      add() { echo $(($1 + $2)); }\n      mul() { echo $(($1 * $2)); }\n"
	}
	return fmt.Sprintf("- ready: {issue: %d, why: \"its sections are there\"}\n", issue)
}

// volatile are the parts of a task the simulator's digest leaves out,
// each with why: a part that changes on every run makes every input new.
var volatile = []*regexp.Regexp{
	regexp.MustCompile(`\d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ`),     // a time stamp: the clock moved, not the input
	regexp.MustCompile(`\d{8}T\d{6}\.\d+-[a-z-]+`),            // a run folder's name
	regexp.MustCompile(`(?m)^.*(?:minutes?|hours?|days?) ago`), // an age, read from the clock
}

// digest is the simulator's own digest of what the fake received: the
// oracle I2 is checked against, independent of the engine's.
func (a *simAgent) digest(task []byte) string {
	s := strings.ReplaceAll(string(task), a.work, "<work>") // the run's own folder
	for _, v := range volatile {
		s = v.ReplaceAllString(s, "")
	}
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:6])
}

// ---- the properties ----

// failure is the first property a run broke.
type failure struct {
	Property string
	Tick     int
	Detail   string
}

func (f *failure) String() string {
	if f == nil {
		return "none"
	}
	return fmt.Sprintf("%s at tick %d: %s", f.Property, f.Tick, f.Detail)
}

// The statuses I1 accepts, read from today's findings: a step started or
// queued (Due), a wait said with what it waits on (Waiting). Anything
// else on an open issue, "acted, nothing to run" first, is no status.
var (
	dueRules     = []string{"state-wakes", "reconcile-left"}
	waitingRules = []string{"state-blocked-quiet", "state-blocked-seen", "state-waits", "state-room", "repro-waits", "state-review-missing"}
)

// status is an open issue's status after a tick, "" for none, and what
// was said instead.
func status(res *line.Result, id int) (string, string) {
	where := fmt.Sprintf("#%d", id)
	if res.Woke[id] != "" {
		return "due(" + res.Woke[id] + ")", ""
	}
	var said []string
	for _, f := range res.Findings {
		if f.Where != where {
			continue
		}
		switch {
		case slices.Contains(dueRules, f.Rule):
			return "due", ""
		case slices.Contains(waitingRules, f.Rule):
			return "waiting", ""
		}
		said = append(said, f.Rule+": "+f.Message)
	}
	if len(said) == 0 {
		return "", "nothing said"
	}
	return "", strings.Join(said, "; ")
}

var recordRe = regexp.MustCompile(`workline:line-record (\{[^\n]*\}) -->`)

// trail is what a machine error must leave as it was: an issue's labels,
// and its record's state and hops.
func trail(is forge.FakeItem) string {
	rec := ""
	for _, c := range is.Comments {
		if m := recordRe.FindStringSubmatch(c.Body); m != nil {
			var r struct {
				State string `json:"state"`
				Hops  int    `json:"hops"`
			}
			_ = json.Unmarshal([]byte(m[1]), &r) // unread: no state, no hop
			rec = fmt.Sprintf("state %s, hops %d", r.State, r.Hops)
		}
	}
	return fmt.Sprintf("labels %v, %s", slices.Sorted(slices.Values(is.Labels)), rec)
}

func issueOf(s forge.FakeState, id int) *forge.FakeItem {
	for i := range s.Issues {
		if s.Issues[i].ID == id {
			return &s.Issues[i]
		}
	}
	return nil
}

// tickChecks are the properties checked after each tick, in order.
func (w *world) tickChecks(k int, before, after forge.FakeState, res, again *line.Result, calls []call) *failure {
	fail := func(p, format string, args ...any) *failure {
		return &failure{Property: p, Tick: k, Detail: fmt.Sprintf(format, args...)}
	}
	states := map[string]bool{}
	for _, s := range w.cfg.States {
		states[s.Label] = true
	}
	// One state label at a time (ADR-0059 §2).
	for _, is := range after.Issues {
		var on []string
		for _, l := range is.Labels {
			if states[l] {
				on = append(on, l)
			}
		}
		if !is.Closed && len(on) > 1 {
			return fail("one-state-label", "#%d carries %v", is.ID, on)
		}
	}
	// I3: a machine error moves nothing.
	for _, c := range calls {
		if !c.Kind.machine() || c.Issue == 0 {
			continue
		}
		b, a := issueOf(before, c.Issue), issueOf(after, c.Issue)
		if b != nil && a != nil && trail(*b) != trail(*a) {
			return fail("i3-machine-error-moves-nothing", "#%d after the %s's %s: %s, was %s", c.Issue, c.Role, c.Kind, trail(*a), trail(*b))
		}
	}
	// No Waiting(person) while a machine-error budget remains.
	for _, c := range calls {
		a := issueOf(after, c.Issue)
		if !c.Kind.machine() || a == nil || !slices.Contains(a.Labels, "workline:blocked") || slices.Contains(issueOf(before, c.Issue).Labels, "workline:blocked") {
			continue
		}
		errs := 0
		for _, d := range w.agent.calls {
			if d.Issue == c.Issue && d.Role == c.Role && d.Kind.machine() {
				errs++
			}
		}
		if errs <= budget {
			return fail("no-person-wait-with-budget", "#%d blocked after the %s's %s %d of %d", c.Issue, c.Role, c.Kind, errs, budget)
		}
	}
	// I1: every open issue on the line has a status.
	for _, is := range after.Issues {
		if is.Closed || !slices.ContainsFunc(is.Labels, func(l string) bool { return states[l] }) {
			continue
		}
		if st, said := status(res, is.ID); st == "" {
			return fail("i1-status", "#%d has no status: %s", is.ID, said)
		}
	}
	// I2: never the same input twice, but after a machine error with
	// budget left; the simulator's digest is the oracle.
	for i, c := range w.agent.calls {
		if c.Tick != k {
			continue
		}
		errs, seen := 0, false
		for _, d := range w.agent.calls[:i] {
			if d.Role == c.Role && d.Issue == c.Issue && d.Digest == c.Digest {
				seen = true
				if d.Kind.machine() {
					errs++
				}
			}
		}
		if seen && (errs == 0 || errs > budget) {
			return fail("i2-same-input", "the %s ran on #%d's input %s again", c.Role, c.Issue, c.Digest)
		}
	}
	// The same tick twice starts no second step: the second run starts
	// only the next state's role, on an issue the first one moved.
	for id, role := range again.Woke {
		b, a := issueOf(before, id), issueOf(after, id)
		if role == res.Woke[id] || b == nil || a == nil || slices.Equal(b.Labels, a.Labels) {
			return fail("same-tick-twice", "the second run started the %s on #%d again", role, id)
		}
	}
	return nil
}

// outcomeOf runs a trace with an agent following script, and returns
// the first property broken.
func runTrace(t *testing.T, tr trace, script []outcome) (*failure, string) {
	a := &simAgent{script: script}
	w := newWorld(t, tr, a)
	var log strings.Builder
	k := w.k()
	quiet := false
	for i := range k {
		tick = i
		at := start.Add(time.Duration(i) * period)
		before := w.state(t)
		n := len(a.calls)
		res := w.run(at)
		mid := w.state(t)
		calls := slices.Clone(a.calls[n:])
		again := w.run(at)
		fmt.Fprintf(&log, "tick %d: %s; woke %v; calls %v\n", i, res.Summary, res.Woke, calls)
		if len(calls) > 0 || len(res.Woke) > 0 {
			for _, f := range res.Findings {
				if f.Level != "info" {
					fmt.Fprintf(&log, "  %s %s %s: %.200s\n", f.Level, f.Rule, f.Where, f.Message)
				}
			}
		}
		if f := w.tickChecks(i, before, mid, res, again, calls); f != nil {
			return f, log.String()
		}
		quiet = len(calls) == 0 && len(res.Woke) == 0 && mid.Writes == before.Writes
	}
	if !quiet {
		return &failure{Property: "quiet-within-k", Tick: k - 1, Detail: fmt.Sprintf("still acting after K = %d ticks", k)}, log.String()
	}
	end := w.state(t)
	if len(script) == 0 { // always progress
		for _, is := range end.Issues {
			if !is.Closed {
				st, _ := status(w.run(start.Add(time.Duration(k)*period)), is.ID)
				return &failure{Property: "always-progress-done", Tick: k, Detail: fmt.Sprintf("#%d not done: %s, %s", is.ID, trail(is), st)}, log.String()
			}
		}
	}
	// One byte of a body changed makes its issue due.
	for _, is := range end.Issues {
		if is.Closed {
			continue
		}
		f := forge.Fake{Path: w.forge}
		if err := f.SetBody(is.ID, is.Body+"."); err != nil {
			t.Fatal(err)
		}
		tick = k
		res := w.run(start.Add(time.Duration(k) * period))
		if res.Woke[is.ID] == "" {
			st, said := status(res, is.ID)
			return &failure{Property: "one-byte-due", Tick: k, Detail: fmt.Sprintf("#%d's body changed, not due: %s %s", is.ID, st, said)}, log.String()
		}
	}
	return nil, log.String()
}

// pending reads pending.txt: per trace, the first property it breaks today.
func pending(t *testing.T) map[string]string {
	out := map[string]string{}
	f, err := os.Open("pending.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, prop, _ := strings.Cut(line, " ")
		out[name] = strings.TrimSpace(prop)
	}
	return out
}

// TestTraces runs each trace twice with an agent that always progresses:
// the same run each time, and the first property broken the one
// pending.txt names, or none.
func TestTraces(t *testing.T) {
	want := pending(t)
	for name := range want {
		if !slices.ContainsFunc(traces, func(tr trace) bool { return tr.name == name }) {
			t.Errorf("pending.txt names %s, no such trace", name)
		}
	}
	for _, tr := range traces {
		t.Run(tr.name, func(t *testing.T) {
			f1, log1 := runTrace(t, tr, nil)
			f2, log2 := runTrace(t, tr, nil)
			if log1 != log2 || f1.String() != f2.String() {
				t.Fatalf("deterministic: the same trace ran twice differs\n--- first\n%s%s\n--- second\n%s%s", log1, f1, log2, f2)
			}
			if os.Getenv("SIM_LOG") != "" {
				t.Log(log1)
			}
			got := ""
			if f1 != nil {
				got = f1.Property
			}
			switch {
			case got == want[tr.name]:
				t.Logf("first property broken: %s", f1)
			case want[tr.name] == "":
				t.Errorf("%s\n%s", f1, log1)
			case got == "":
				t.Errorf("passes now: take %s off pending.txt", tr.name)
			default:
				t.Errorf("pending.txt says %s breaks %s first, it breaks %s\n%s", tr.name, want[tr.name], f1, log1)
			}
		})
	}
}

// seeds is how many seeded runs a trace that passes gets; the night sets
// SIM_SEEDS higher and SIM_SEED_BASE new.
func seeds() (base, n int64) {
	n, base = 200, 1
	if v, err := strconv.ParseInt(os.Getenv("SIM_SEEDS"), 10, 64); err == nil {
		n = v
	}
	if v, err := strconv.ParseInt(os.Getenv("SIM_SEED_BASE"), 10, 64); err == nil {
		base = v
	}
	return base, n
}

// scriptOf draws a run's outcomes from seed.
func scriptOf(seed int64) []outcome {
	rng := rand.New(rand.NewSource(seed))
	s := make([]outcome, 48)
	for i := range s {
		switch r := rng.Intn(10); {
		case r < 5:
			s[i] = progress
		case r < 7:
			s[i] = waiting
		case r < 9:
			s[i] = machineError
		default:
			s[i] = malformed
		}
	}
	return s
}

// shrink turns outcomes into progress one at a time while the run still
// breaks the same property: the smallest script that shows it.
func shrink(t *testing.T, tr trace, script []outcome, prop string) []outcome {
	s := slices.Clone(script)
	for i := range s {
		if s[i] == progress {
			continue
		}
		try := slices.Clone(s)
		try[i] = progress
		if f, _ := runTrace(t, tr, try); f != nil && f.Property == prop {
			s = try
		}
	}
	for len(s) > 0 && s[len(s)-1] == progress {
		s = s[:len(s)-1]
	}
	return s
}

func parseScript(v string) []outcome {
	var s []outcome
	for _, o := range strings.Split(v, ",") {
		s = append(s, outcome(strings.TrimSpace(o)))
	}
	return s
}

// TestSeeds runs the traces pending.txt does not name under seeded
// outcomes: progress, waiting, each machine error, malformed answers.
// A failure is never pending: it prints its seed and shrunk script.
func TestSeeds(t *testing.T) {
	want := pending(t)
	base, n := seeds()
	var green []trace
	for _, tr := range traces {
		// SIM_TRACE takes one trace, pending or not: to look at it.
		if v := os.Getenv("SIM_TRACE"); v == tr.name || v == "" && want[tr.name] == "" {
			green = append(green, tr)
		}
	}
	if len(green) == 0 {
		t.Skip("every trace is pending")
	}
	if v := os.Getenv("SIM_SCRIPT"); v != "" {
		for _, tr := range green {
			if f, log := runTrace(t, tr, parseScript(v)); f != nil {
				t.Errorf("%s: %s\n%s", tr.name, f, log)
			}
		}
		return
	}
	if v, err := strconv.ParseInt(os.Getenv("SIM_SEED"), 10, 64); err == nil {
		base, n = v, 1
	}
	for i := range n {
		seed := base + i
		tr := green[int(seed)%len(green)]
		script := scriptOf(seed)
		if f, _ := runTrace(t, tr, script); f != nil {
			small := shrink(t, tr, script, f.Property)
			_, log := runTrace(t, tr, small)
			var words []string
			for _, o := range small {
				words = append(words, string(o))
			}
			t.Errorf("%s, seed %d: %s\nreplay: SIM_SEED=%d, or shrunk: SIM_SCRIPT=%q\n%s", tr.name, seed, f, seed, strings.Join(words, ","), log)
		}
	}
}
