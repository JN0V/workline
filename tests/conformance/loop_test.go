package conformance

import (
	enc "encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// The GitHub loop (ADR-0069's 2026-11-17 amendment, #1665): one long job
// runs `workline tick --json` every minute and, in its dry run, logs what
// it would dispatch, exactly the list tick prints, and starts nothing.
// Both copies are read: the template a project installs, and this
// repository's own.

var loopFiles = []string{"../../defaults/ci/github/workline-loop.yml", "../../.github/workflows/workline-loop.yml"}

type loopStep struct {
	Name string            `yaml:"name"`
	ID   string            `yaml:"id"`
	If   string            `yaml:"if"`
	Uses string            `yaml:"uses"`
	Run  string            `yaml:"run"`
	Env  map[string]string `yaml:"env"`
}

type loopWorkflow struct {
	On   map[string]any `yaml:"on"`
	Jobs map[string]struct {
		Needs       any               `yaml:"needs"`
		If          string            `yaml:"if"`
		Timeout     int               `yaml:"timeout-minutes"`
		Concurrency map[string]any    `yaml:"concurrency"`
		Permissions map[string]string `yaml:"permissions"`
		Steps       []loopStep        `yaml:"steps"`
	} `yaml:"jobs"`
}

func readLoop(t *testing.T, f string) loopWorkflow {
	t.Helper()
	data, err := os.ReadFile(f)
	if err != nil {
		t.Fatal(err)
	}
	var wf loopWorkflow
	if err := yaml.Unmarshal(data, &wf); err != nil {
		t.Fatalf("%s: %v", f, err)
	}
	return wf
}

func loopStepNamed(t *testing.T, wf loopWorkflow, f, name string) loopStep {
	t.Helper()
	for _, s := range wf.Jobs["loop"].Steps {
		if s.Name == name {
			return s
		}
	}
	t.Fatalf("%s: no step %q in the job loop", f, name)
	return loopStep{}
}

// loopStates: fixed states on a fake forge, at loopNow, each with the
// number of jobs tick lists on it at least.
type loopState struct {
	name  string
	forge map[string]any
	jobs  int
}

const loopNow = "2026-11-15T19:30:00Z"

func loopStates() []loopState {
	ready := func(n int) []map[string]any {
		var is []map[string]any
		for i := 1; i <= n; i++ {
			is = append(is, map[string]any{"id": i, "title": "Item", "body": "x", "labels": []string{"workline:ready"}})
		}
		return is
	}
	return []loopState{
		{"three jobs due", map[string]any{"issues": ready(6), "runs": []map[string]any{
			{"name": "workline merge", "workflow": "workline-merge.yml", "status": "completed", "success": true, "started": "2026-11-15T19:00:00Z"},
			{"name": "workline sample", "workflow": "workline-sample.yml", "status": "completed", "success": true, "started": "2026-11-09T05:45:00Z"},
		}}, 3},
		{"nothing due", map[string]any{"issues": ready(10), "runs": []map[string]any{
			{"name": "workline develop reconcile", "workflow": "workline-develop.yml", "status": "in_progress", "started": "2026-11-15T19:29:00Z"},
			{"name": "workline merge", "workflow": "workline-merge.yml", "status": "completed", "success": true, "started": "2026-11-15T19:00:00Z"},
			{"name": "workline gardening docs", "workflow": "workline-gardening.yml", "status": "completed", "success": true, "started": "2026-11-15T18:30:00Z"},
			{"name": "workline sample", "workflow": "workline-sample.yml", "status": "completed", "success": true, "started": "2026-11-09T05:45:00Z"},
		}}, 0},
		{"no run yet", map[string]any{"issues": ready(6)}, 2},
	}
}

// loopFixture: a clone of a repository with a config, the fake forge
// holding st, and a bin where workline runs the engine on that forge at
// loopNow, gh logs each call in calls; tick's list on that state.
func loopFixture(t *testing.T, st loopState) (repo, bin, tmp, calls string, want []map[string]any) {
	t.Helper()
	for _, tool := range []string{"jq", "bash"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skip(tool + " is not installed")
		}
	}
	dir := t.TempDir()
	origin := filepath.Join(dir, "origin")
	repo, bin, tmp = filepath.Join(dir, "repo"), filepath.Join(dir, "bin"), filepath.Join(dir, "tmp")
	for _, d := range []string{filepath.Join(origin, ".workline"), bin, tmp} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	cfg := "routing: {events: {schedule: [product-owner], issue: [tester, developer]}, wip: {ready: 10}}\n"
	if err := os.WriteFile(filepath.Join(origin, ".workline", "config.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, origin, "init", "-q", "-b", "main")
	gitIn(t, origin, "add", ".")
	gitIn(t, origin, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-qm", "config")
	gitIn(t, dir, "clone", "-q", origin, repo)
	forgeFile := filepath.Join(dir, "forge.json")
	data, _ := enc.Marshal(st.forge)
	if err := os.WriteFile(forgeFile, data, 0o644); err != nil {
		t.Fatal(err)
	}
	calls = filepath.Join(dir, "gh-calls")
	writeExec(t, filepath.Join(bin, "workline"), `#!/bin/bash
args=(); for a in "$@"; do [ "$a" = github ] && a="fake:`+forgeFile+`"; args+=("$a"); done
[ "$1" = tick ] && [ "$2" != --help ] && args+=(--now `+loopNow+`)
exec "`+engineBin+`" "${args[@]}"
`)
	writeExec(t, filepath.Join(bin, "gh"), "#!/bin/bash\necho \"$*\" >> \""+calls+"\"\n")
	want = tickJobs(t, repo, forgeFile, loopNow)
	if len(want) < st.jobs {
		t.Fatalf("%s: tick lists %d jobs, the state wants %d at least: %v", st.name, len(want), st.jobs, want)
	}
	return repo, bin, tmp, calls, want
}

// loggedItems: tick's items the beat logged after prefix, in order.
func loggedItems(t *testing.T, out, prefix string) []map[string]any {
	t.Helper()
	var got []map[string]any
	for _, line := range strings.Split(out, "\n") {
		js, ok := strings.CutPrefix(line, prefix)
		if !ok {
			continue
		}
		var it map[string]any
		if err := enc.Unmarshal([]byte(js), &it); err != nil {
			t.Fatalf("a line not in the format `%s<tick's JSON item>`: %q", prefix, line)
		}
		got = append(got, it)
	}
	return got
}

// A dry run dispatches nothing, and logs exactly what tick lists
// Given a fixed state on a fake forge and a fixed clock,
// When the loop's beat runs once, dispatching unset or false,
// Then it logs one `would dispatch` line per job `workline tick --json`
// prints, the same jobs in the same order, and calls no dispatch.
func TestLoopDryRunLogsExactlyWhatTickLists(t *testing.T) {
	for _, f := range loopFiles {
		beat := loopStepNamed(t, readLoop(t, f), f, "The beat")
		for _, st := range loopStates() {
			for _, dispatch := range []string{"", "false"} {
				repo, bin, tmp, calls, want := loopFixture(t, st)
				out := runBeat(t, beat.Run, repo, bin, tmp, "DISPATCH="+dispatch)
				got := loggedItems(t, out, "would dispatch ")
				if !reflect.DeepEqual(got, want) && (len(got) > 0 || len(want) > 0) {
					t.Errorf("%s, %s: the dry run logs %v, tick lists %v", f, st.name, got, want)
				}
				if b, _ := os.ReadFile(filepath.Join(tmp, "output")); !strings.Contains(string(b), "handover=true") {
					t.Errorf("%s, %s: a beat ended by its time hands over to no successor", f, st.name)
				}
				if b, _ := os.ReadFile(calls); len(b) > 0 {
					t.Errorf("%s, %s, DISPATCH=%q: the dry run called gh: %s", f, st.name, dispatch, b)
				}
				if len(want) == 0 && !strings.Contains(out, ": nothing due") {
					t.Errorf("%s, %s: a beat with nothing due does not say so:\n%s", f, st.name, out)
				}
				for _, it := range want {
					if !strings.Contains(out, it["key"].(string)+" ×1") || !strings.Contains(out, "would have dispatched") {
						t.Errorf("%s, %s: the run's notice does not count %q:\n%s", f, st.name, it["key"], out)
					}
				}
			}
		}
	}
}

// dispatchOf: each job tick lists without --issues, the workflow the loop
// dispatches it in, its inputs, and the part of the workflow's run-name
// (or its name) that names a dispatched run as tick's key.
var dispatchOf = map[string]struct {
	file   string
	inputs map[string]string
	named  string
}{
	"reconcile": {"workline-develop.yml", map[string]string{"reconcile": "true"}, "inputs.reconcile == 'true' && 'workline develop reconcile'"},
	"backlog":   {"workline-gardening.yml", map[string]string{"job": "backlog"}, "format('workline gardening {0}', inputs.job)"},
	"nightly":   {"workline-gardening.yml", map[string]string{"job": "nightly"}, "format('workline gardening {0}', inputs.job)"},
	"docs":      {"workline-gardening.yml", map[string]string{"job": "docs"}, "format('workline gardening {0}', inputs.job)"},
	"merge":     {"workline-merge.yml", nil, "name: workline merge"},
	"sample":    {"workline-sample.yml", nil, "name: workline sample"},
	"seeds":     {"test.yml", map[string]string{"job": "seeds"}, "inputs.job == 'seeds') && 'workline test seeds'"},
}

// ghCall: the `gh workflow run` the loop sends for a job.
func ghCall(job string) string {
	d := dispatchOf[job]
	c := "workflow run " + d.file
	keys := make([]string, 0, len(d.inputs))
	for k := range d.inputs {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		c += " -f " + k + "=" + d.inputs[k]
	}
	return c + " --ref main"
}

// The loop dispatches exactly what tick lists, once
// Given a fixed state on a fake forge, a fixed clock, and
// WORKLINE_LOOP_DISPATCH=true,
// When the loop's beat runs once, then once more at once,
// Then the first beat sends one `gh workflow run` per job tick lists, the
// same jobs in the same order, none for a job whose run is going, and
// logs each `dispatched`; the second sends none: each key is held while
// its run may not be listed yet.
func TestLoopDispatchesExactlyWhatTickLists(t *testing.T) {
	for _, f := range loopFiles {
		beat := loopStepNamed(t, readLoop(t, f), f, "The beat")
		if beat.Env["HOLD_SECONDS"] == "" {
			t.Fatalf("%s: the beat sets no HOLD_SECONDS", f)
		}
		for _, st := range loopStates() {
			repo, bin, tmp, calls, want := loopFixture(t, st)
			out := runBeat(t, beat.Run, repo, bin, tmp, "DISPATCH=true", "HOLD_SECONDS=120")
			var wantCalls []string
			for _, it := range want {
				wantCalls = append(wantCalls, ghCall(it["job"].(string)))
			}
			b, _ := os.ReadFile(calls)
			var gotCalls []string
			if c := strings.TrimSpace(string(b)); c != "" {
				gotCalls = strings.Split(c, "\n")
			}
			if !slices.Equal(gotCalls, wantCalls) {
				t.Errorf("%s, %s: the loop sent %q; tick lists %v, so %q", f, st.name, gotCalls, want, wantCalls)
			}
			if got := loggedItems(t, out, "dispatched "); !reflect.DeepEqual(got, want) && (len(got) > 0 || len(want) > 0) {
				t.Errorf("%s, %s: the loop logs %v dispatched, tick lists %v", f, st.name, got, want)
			}
			if strings.Contains(out, "would dispatch") {
				t.Errorf("%s, %s: dispatching on, the beat still logs a dry run:\n%s", f, st.name, out)
			}
			for _, it := range want {
				if !strings.Contains(out, "dispatched: ") || !strings.Contains(out, it["key"].(string)+" ×1") {
					t.Errorf("%s, %s: the run's notice does not count %q dispatched:\n%s", f, st.name, it["key"], out)
				}
			}
			// The same job, the beat after: held, not dispatched again.
			if err := os.Remove(calls); err != nil && len(want) > 0 {
				t.Fatal(err)
			}
			again := runBeatKeeping(t, beat.Run, repo, bin, tmp, "DISPATCH=true", "HOLD_SECONDS=120")
			if b, _ := os.ReadFile(calls); len(b) > 0 {
				t.Errorf("%s, %s: a job dispatched a beat before is dispatched again: %s", f, st.name, b)
			}
			if got := loggedItems(t, again, "held "); !reflect.DeepEqual(got, want) && (len(got) > 0 || len(want) > 0) {
				t.Errorf("%s, %s: the beat after holds %v, tick lists %v", f, st.name, got, want)
			}
		}
	}
}

// Each job tick lists starts in its workflow, under its key
// Given a tick that lists every timed job, and one the loop does not know,
// When the loop's beat runs with WORKLINE_LOOP_DISPATCH=true,
// Then each job is dispatched in its workflow with its inputs, the job
// it does not know is said and not dispatched; and each such workflow
// can be dispatched (workflow_dispatch, those inputs), its run named by
// tick's key.
func TestLoopDispatchesEachJobInItsWorkflow(t *testing.T) {
	var jobs []string
	for j := range dispatchOf {
		jobs = append(jobs, j)
	}
	slices.Sort(jobs)
	var items []string
	for _, j := range jobs {
		items = append(items, `{"job":"`+j+`","why":"due","key":"k-`+j+`"}`)
	}
	items = append(items, `{"job":"unheard","why":"due","key":"k-unheard"}`)
	for _, f := range loopFiles {
		beat := loopStepNamed(t, readLoop(t, f), f, "The beat")
		dir := t.TempDir()
		bin, tmp := filepath.Join(dir, "bin"), filepath.Join(dir, "tmp")
		for _, d := range []string{bin, tmp} {
			if err := os.MkdirAll(d, 0o755); err != nil {
				t.Fatal(err)
			}
		}
		calls := filepath.Join(dir, "gh-calls")
		writeExec(t, filepath.Join(bin, "workline"), "#!/bin/bash\necho '["+strings.Join(items, ",")+"]'\n")
		writeExec(t, filepath.Join(bin, "gh"), "#!/bin/bash\necho \"$*\" >> \""+calls+"\"\n")
		out := runBeat(t, beat.Run, dir, bin, tmp, "DISPATCH=true", "HOLD_SECONDS=120")
		var want []string
		for _, j := range jobs {
			want = append(want, ghCall(j))
		}
		b, _ := os.ReadFile(calls)
		if got := strings.Split(strings.TrimSpace(string(b)), "\n"); !slices.Equal(got, want) {
			t.Errorf("%s: the loop sent %q; want %q", f, got, want)
		}
		if !strings.Contains(out, "unheard: no workflow known to start it") || !strings.Contains(out, "k-unheard not dispatched") {
			t.Errorf("%s: a job the loop does not know is not said:\n%s", f, out)
		}
	}
	// Each workflow, in this repository and the template where it is one.
	for _, j := range jobs {
		d := dispatchOf[j]
		for _, dir := range []string{"../../.github/workflows", "../../defaults/ci/github"} {
			p := filepath.Join(dir, d.file)
			data, err := os.ReadFile(p)
			if os.IsNotExist(err) && strings.Contains(dir, "defaults") && d.file == "test.yml" {
				continue // a project's own tests: workline's seeds are its own
			} else if err != nil {
				t.Fatal(err)
			}
			var wf struct {
				On struct {
					Dispatch *struct {
						Inputs map[string]any `yaml:"inputs"`
					} `yaml:"workflow_dispatch"`
				} `yaml:"on"`
			}
			if err := yaml.Unmarshal(data, &wf); err != nil {
				t.Fatalf("%s: %v", p, err)
			}
			wd := wf.On.Dispatch
			if !strings.Contains(string(data), "\n  workflow_dispatch:") {
				t.Errorf("%s: no workflow_dispatch; tick lists %s, the loop dispatches it there", p, j)
				continue
			}
			for in := range d.inputs {
				if _, ok := wd.Inputs[in]; wd == nil || !ok {
					t.Errorf("%s: workflow_dispatch has no input %q, which the loop passes for %s", p, in, j)
				}
			}
			if !strings.Contains(string(data), d.named) {
				t.Errorf("%s: its runs are not named %s's key (want %q in its run-name or name)", p, j, d.named)
			} else if strings.HasPrefix(d.named, "name: ") && strings.Contains(string(data), "\nrun-name:") {
				t.Errorf("%s: a run-name hides its name, %s's key", p, j)
			}
		}
	}
}

// A release with no tick skips
// Given a WORKLINE_VERSION with no tick (the line's pin before v0.79.0),
// When the loop's beat runs,
// Then it says so, exits 0, and starts no successor.
func TestLoopSkipsAReleaseWithNoTick(t *testing.T) {
	for _, f := range loopFiles {
		wf := readLoop(t, f)
		beat := loopStepNamed(t, wf, f, "The beat")
		dir := t.TempDir()
		bin, tmp := filepath.Join(dir, "bin"), filepath.Join(dir, "tmp")
		for _, d := range []string{bin, tmp} {
			if err := os.MkdirAll(d, 0o755); err != nil {
				t.Fatal(err)
			}
		}
		writeExec(t, filepath.Join(bin, "workline"), "#!/bin/bash\necho 'workline: unknown command' >&2; exit 64\n")
		out := runBeat(t, beat.Run, dir, bin, tmp)
		if !strings.Contains(out, "has no tick") {
			t.Errorf("%s: a release with no tick is not said:\n%s", f, out)
		}
		if b, _ := os.ReadFile(filepath.Join(tmp, "output")); strings.Contains(string(b), "handover=true") {
			t.Errorf("%s: a release with no tick still hands over to a successor", f)
		}
	}
}

// A crash hands over to no successor
// Given a beat that fails (a wait that is not a number),
// When the loop's beat runs,
// Then the step fails and hands over to no successor: no chain of
// crashing runs; the hourly schedule restarts it.
func TestLoopCrashHandsOverToNoSuccessor(t *testing.T) {
	for _, f := range loopFiles {
		wf := readLoop(t, f)
		beat := loopStepNamed(t, wf, f, "The beat")
		dir := t.TempDir()
		bin, tmp := filepath.Join(dir, "bin"), filepath.Join(dir, "tmp")
		for _, d := range []string{bin, tmp} {
			if err := os.MkdirAll(d, 0o755); err != nil {
				t.Fatal(err)
			}
		}
		writeExec(t, filepath.Join(bin, "workline"), "#!/bin/bash\necho '[]'\n")
		cmd := exec.Command("bash", "-e", "-c", beat.Run)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"), "RUNNER_TEMP="+tmp, "GITHUB_OUTPUT="+filepath.Join(tmp, "output"),
			"GIT_CONFIG_GLOBAL=/dev/null", "BRANCH=main", "LOOP_MINUTES=1", "BEAT_SECONDS=x", "WORKLINE_VERSION=v0.0.0")
		if out, err := cmd.CombinedOutput(); err == nil {
			t.Errorf("%s: a beat whose wait fails still succeeds:\n%s", f, out)
		}
		if b, _ := os.ReadFile(filepath.Join(tmp, "output")); strings.Contains(string(b), "handover=true") {
			t.Errorf("%s: a crashed beat hands over to a successor", f)
		}
		next := loopStepNamed(t, wf, f, "The next loop")
		if next.If != "steps.beat.outputs.handover == 'true'" {
			t.Errorf("%s: the next loop's if is %q; want the beat's handover alone, so a crash or a cancel starts none", f, next.If)
		}
	}
}

// Tick failing beat after beat stops the loop
// Given a tick that fails on every beat,
// When the loop's beat runs,
// Then it stops after FAILS_MAX beats in a row, fails, and hands over to
// no successor: no endless loop of warnings.
func TestLoopStopsWhenTickKeepsFailing(t *testing.T) {
	for _, f := range loopFiles {
		wf := readLoop(t, f)
		beat := loopStepNamed(t, wf, f, "The beat")
		if beat.Env["FAILS_MAX"] == "" {
			t.Fatalf("%s: the beat sets no FAILS_MAX", f)
		}
		dir := t.TempDir()
		bin, tmp := filepath.Join(dir, "bin"), filepath.Join(dir, "tmp")
		for _, d := range []string{bin, tmp} {
			if err := os.MkdirAll(d, 0o755); err != nil {
				t.Fatal(err)
			}
		}
		writeExec(t, filepath.Join(bin, "workline"), "#!/bin/bash\n[ \"$2\" = --help ] && exit 0\necho 'forge down' >&2; exit 1\n")
		cmd := exec.Command("bash", "-e", "-c", beat.Run)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"), "RUNNER_TEMP="+tmp, "GITHUB_OUTPUT="+filepath.Join(tmp, "output"),
			"GIT_CONFIG_GLOBAL=/dev/null", "BRANCH=main", "LOOP_MINUTES=60", "BEAT_SECONDS=0", "FAILS_MAX=3", "WORKLINE_VERSION=v0.0.0")
		out, err := cmd.CombinedOutput()
		if err == nil || !strings.Contains(string(out), "failed 3 beats in a row: the loop stops") {
			t.Errorf("%s: a tick failing beat after beat does not stop the loop (err %v):\n%s", f, err, out)
		}
		if b, _ := os.ReadFile(filepath.Join(tmp, "output")); strings.Contains(string(b), "handover=true") {
			t.Errorf("%s: a loop stopped by failures hands over to a successor", f)
		}
	}
}

// One loop at a time, handed over before 6 h
// Given the loop's workflow,
// When it is read,
// Then only a dispatch and a schedule start it; its job is in one
// concurrency group that cancels none; it ends its beats before its
// timeout, under GitHub's 6 h, and dispatches its successor.
func TestLoopOneAtATimeHandedOverBeforeSixHours(t *testing.T) {
	for _, f := range loopFiles {
		wf := readLoop(t, f)
		var on []string
		for k := range wf.On {
			on = append(on, k)
		}
		slices.Sort(on)
		if !slices.Equal(on, []string{"schedule", "workflow_dispatch"}) {
			t.Errorf("%s: started by %v; want a dispatch and a schedule only", f, on)
		}
		loop := wf.Jobs["loop"]
		if loop.Concurrency["group"] != "workline-loop" || loop.Concurrency["cancel-in-progress"] != false {
			t.Errorf("%s: the loop's concurrency is %v; want one group, cancelling none", f, loop.Concurrency)
		}
		if loop.Timeout <= 0 || loop.Timeout >= 360 {
			t.Errorf("%s: the loop's timeout-minutes is %d; want under GitHub's 360", f, loop.Timeout)
		}
		beat := loopStepNamed(t, wf, f, "The beat")
		var minutes int
		if err := yaml.Unmarshal([]byte(beat.Env["LOOP_MINUTES"]), &minutes); err != nil || minutes <= 0 || minutes+5 > loop.Timeout {
			t.Errorf("%s: LOOP_MINUTES %q; want its beats to end 5 minutes before the timeout, %d", f, beat.Env["LOOP_MINUTES"], loop.Timeout)
		}
		if loop.Permissions["actions"] != "write" || loop.Permissions["contents"] != "read" {
			t.Errorf("%s: the loop's permissions %v; want actions: write for its successor, contents read", f, loop.Permissions)
		}
		next := loopStepNamed(t, wf, f, "The next loop")
		if !strings.Contains(next.Run, "gh workflow run workline-loop.yml") ||
			!strings.Contains(next.Run, `after="$RUN"`) {
			t.Errorf("%s: the next loop does not dispatch the loop, handing over: %q", f, next.Run)
		}
		checkouts := 0
		for _, s := range loop.Steps {
			if strings.HasPrefix(s.Uses, "actions/checkout@") {
				checkouts++
			}
		}
		if checkouts != 1 {
			t.Errorf("%s: %d checkouts in the loop; want one, kept between beats with its .git/workline", f, checkouts)
		}
	}
}

// tickJobs is what the engine's tick lists on that state.
func tickJobs(t *testing.T, repo, forgeFile, now string) []map[string]any {
	t.Helper()
	cmd := exec.Command(engineBin, "tick", "--forge", "fake:"+forgeFile, "--now", now, "--json")
	cmd.Dir = repo
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("workline tick: %v", err)
	}
	var items []map[string]any
	if err := enc.Unmarshal(out, &items); err != nil {
		t.Fatalf("workline tick --json: %v: %s", err, out)
	}
	return items
}

// runBeat runs the beat's script as GitHub does (bash -e), once: no
// minute left, no wait between beats; env adds to or overrides the
// runner's.
func runBeat(t *testing.T, script, dir, bin, tmp string, env ...string) string {
	t.Helper()
	cmd := exec.Command("bash", "-e", "-c", script)
	cmd.Dir = dir
	cmd.Env = append(append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"), "RUNNER_TEMP="+tmp,
		"GITHUB_OUTPUT="+filepath.Join(tmp, "output"),
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "BRANCH=main", "DISPATCH=", "LOOP_MINUTES=0", "BEAT_SECONDS=0", "WORKLINE_VERSION=v0.0.0", "GH_TOKEN=x"), env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("the beat failed: %v\n%s", err, out)
	}
	return string(out)
}

// runBeatKeeping runs one more beat in the same job: what the beats
// before kept in RUNNER_TEMP is kept, as between two beats of one run.
func runBeatKeeping(t *testing.T, script, dir, bin, tmp string, env ...string) string {
	t.Helper()
	kept, err := os.ReadFile(filepath.Join(tmp, "dispatched"))
	if err != nil {
		t.Fatalf("the beat kept no dispatched list: %v", err)
	}
	// The script starts a job's list empty: a beat in the same job is
	// the loop's body run again, its list as the beat before left it.
	body := strings.Replace(script, `: > "$RUNNER_TEMP/dispatched"`, `cat "$RUNNER_TEMP/dispatched.kept" > "$RUNNER_TEMP/dispatched"`, 1)
	if body == script {
		t.Fatal("the beat no longer starts its dispatched list empty: this test runs it again as the next beat")
	}
	if err := os.WriteFile(filepath.Join(tmp, "dispatched.kept"), kept, 0o644); err != nil {
		t.Fatal(err)
	}
	return runBeat(t, body, dir, bin, tmp, env...)
}

func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
