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
}

// traces are the trial's issues as round 3 left them on 2026-11-04 (#757),
// and the five it merged taken back to how they were opened, the path
// that works. Each blocked one was blocked by the machine, not by a
// person's question (pending.txt says how): an agent that progresses
// takes every one to Done.
var traces = func() []trace {
	var ts []trace
	for _, n := range []int{3, 4, 5, 16, 18, 19, 20, 33, 34, 35, 36, 37} {
		f := fmt.Sprintf("trial-%d", n)
		ts = append(ts, trace{name: f, file: f, end: "done"})
	}
	for _, n := range []int{3, 4, 5, 16, 18} {
		f := fmt.Sprintf("trial-%d", n)
		ts = append(ts, trace{name: f + "-fresh", file: f, fresh: true, end: "done"})
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
	n := len(w.agent.calls)
	res := line.RunWith(line.ReconcileEvent, engine.Options{Repo: w.repo, RolesDir: w.roles, AI: "hook:sim",
		Forge: "fake:" + w.forge, OpenMergeRequest: true}, &line.Gates{})
	w.agent.attribute(n, res.Woke)
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
