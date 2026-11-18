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

// A dry run dispatches exactly what tick lists
// Given a fixed state on a fake forge and a fixed clock,
// When the loop's beat runs once, in its dry run,
// Then it logs one `would dispatch` line per job `workline tick --json`
// prints, the same jobs in the same order, and calls no dispatch.
func TestLoopDryRunLogsExactlyWhatTickLists(t *testing.T) {
	for _, tool := range []string{"jq", "bash"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skip(tool + " is not installed")
		}
	}
	ready := func(n int) []map[string]any {
		var is []map[string]any
		for i := 1; i <= n; i++ {
			is = append(is, map[string]any{"id": i, "title": "Item", "body": "x", "labels": []string{"workline:ready"}})
		}
		return is
	}
	states := []struct {
		name     string
		forge    map[string]any
		dispatch string
		jobs     int // tick's jobs, at least: the state is what the case says
	}{
		{"three jobs due", map[string]any{"issues": ready(6), "runs": []map[string]any{
			{"name": "workline merge", "workflow": "workline-merge.yml", "status": "completed", "success": true, "started": "2026-11-15T19:00:00Z"},
			{"name": "workline sample", "workflow": "workline-sample.yml", "status": "completed", "success": true, "started": "2026-11-09T05:45:00Z"},
		}}, "", 3},
		{"nothing due", map[string]any{"issues": ready(10), "runs": []map[string]any{
			{"name": "workline develop reconcile", "workflow": "workline-develop.yml", "status": "in_progress", "started": "2026-11-15T19:29:00Z"},
			{"name": "workline merge", "workflow": "workline-merge.yml", "status": "completed", "success": true, "started": "2026-11-15T19:00:00Z"},
			{"name": "workline gardening docs", "workflow": "workline-gardening.yml", "status": "completed", "success": true, "started": "2026-11-15T18:30:00Z"},
			{"name": "workline sample", "workflow": "workline-sample.yml", "status": "completed", "success": true, "started": "2026-11-09T05:45:00Z"},
		}}, "", 0},
		{"dispatch asked, not built", map[string]any{"issues": ready(6)}, "true", 2},
	}
	const now = "2026-11-15T19:30:00Z"
	for _, f := range loopFiles {
		wf := readLoop(t, f)
		beat := loopStepNamed(t, wf, f, "The beat")
		for _, st := range states {
			dir := t.TempDir()
			origin, repo, bin, tmp := filepath.Join(dir, "origin"), filepath.Join(dir, "repo"), filepath.Join(dir, "bin"), filepath.Join(dir, "tmp")
			for _, d := range []string{origin, bin, tmp} {
				if err := os.MkdirAll(d, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			cfg := "routing: {events: {schedule: [product-owner], issue: [tester, developer]}, wip: {ready: 10}}\n"
			if err := os.MkdirAll(filepath.Join(origin, ".workline"), 0o755); err != nil {
				t.Fatal(err)
			}
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
			// workline: the engine on the fake forge and the fixed clock,
			// for the forge GitHub the step names; gh: each call logged.
			calls := filepath.Join(dir, "gh-calls")
			writeExec(t, filepath.Join(bin, "workline"), `#!/bin/bash
args=(); for a in "$@"; do [ "$a" = github ] && a="fake:`+forgeFile+`"; args+=("$a"); done
[ "$1" = tick ] && [ "$2" != --help ] && args+=(--now `+now+`)
exec "`+engineBin+`" "${args[@]}"
`)
			writeExec(t, filepath.Join(bin, "gh"), "#!/bin/bash\necho \"$*\" >> \""+calls+"\"\n")
			want := tickJobs(t, repo, forgeFile, now)
			if len(want) < st.jobs {
				t.Fatalf("%s: tick lists %d jobs, the state wants %d at least: %v", st.name, len(want), st.jobs, want)
			}
			out := runBeat(t, beat.Run, repo, bin, tmp, st.dispatch)
			var got []map[string]any
			for _, line := range strings.Split(out, "\n") {
				js, ok := strings.CutPrefix(line, "would dispatch ")
				if !ok {
					continue
				}
				var it map[string]any
				if err := enc.Unmarshal([]byte(js), &it); err != nil {
					t.Fatalf("%s, %s: a line not in the dry run's format, `would dispatch <tick's JSON item>`: %q", f, st.name, line)
				}
				got = append(got, it)
			}
			if !reflect.DeepEqual(got, want) && (len(got) > 0 || len(want) > 0) {
				t.Errorf("%s, %s: the dry run logs %v, tick lists %v", f, st.name, got, want)
			}
			if b, _ := os.ReadFile(filepath.Join(tmp, "output")); !strings.Contains(string(b), "handover=true") {
				t.Errorf("%s, %s: a beat ended by its time hands over to no successor", f, st.name)
			}
			if b, _ := os.ReadFile(calls); len(b) > 0 {
				t.Errorf("%s, %s: the dry run called gh: %s", f, st.name, b)
			}
			if len(want) == 0 && !strings.Contains(out, ": nothing due") {
				t.Errorf("%s, %s: a beat with nothing due does not say so:\n%s", f, st.name, out)
			}
			if st.dispatch == "true" && !strings.Contains(out, "dispatching is not built in this release; a dry run") {
				t.Errorf("%s, %s: WORKLINE_LOOP_DISPATCH=true is not said a dry run still:\n%s", f, st.name, out)
			}
			for _, it := range want {
				if !strings.Contains(out, it["key"].(string)+" ×1") {
					t.Errorf("%s, %s: the run's notice does not count %q:\n%s", f, st.name, it["key"], out)
				}
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
		out := runBeat(t, beat.Run, dir, bin, tmp, "")
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
// minute left, no wait between beats.
func runBeat(t *testing.T, script, dir, bin, tmp, dispatch string) string {
	t.Helper()
	cmd := exec.Command("bash", "-e", "-c", script)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"), "RUNNER_TEMP="+tmp,
		"GITHUB_OUTPUT="+filepath.Join(tmp, "output"),
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "BRANCH=main", "DISPATCH="+dispatch, "LOOP_MINUTES=0", "BEAT_SECONDS=0", "WORKLINE_VERSION=v0.0.0", "GH_TOKEN=x")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("the beat failed: %v\n%s", err, out)
	}
	return string(out)
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
