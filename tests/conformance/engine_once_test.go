package conformance

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// workline's own workflows run the engine of the commit they check out,
// built once a commit (ADR-0050, step 1): every job takes it from the action
// .github/actions/engine, none builds it on its own. The action reads the
// cache by its exact key, keeps only an engine it checked, and runs nothing
// on a binary that does not name the commit checked out, unmodified: never
// an older engine, whatever the cache holds.

const engineAction = "./.github/actions/engine"

type engineStep struct {
	Name string            `yaml:"name"`
	ID   string            `yaml:"id"`
	If   string            `yaml:"if"`
	Uses string            `yaml:"uses"`
	With map[string]string `yaml:"with"`
	Env  map[string]string `yaml:"env"`
	Run  string            `yaml:"run"`
}

func readEngineAction(t *testing.T) []engineStep {
	t.Helper()
	data, err := os.ReadFile("../../.github/actions/engine/action.yml")
	if err != nil {
		t.Fatal(err)
	}
	var a struct {
		Runs struct {
			Using string       `yaml:"using"`
			Steps []engineStep `yaml:"steps"`
		} `yaml:"runs"`
	}
	if err := yaml.Unmarshal(data, &a); err != nil {
		t.Fatal(err)
	}
	if a.Runs.Using != "composite" || len(a.Runs.Steps) == 0 {
		t.Fatalf("the engine's action is not a composite one: %+v", a.Runs)
	}
	return a.Runs.Steps
}

func TestOwnWorkflowsTakeTheEngineOnceACommit(t *testing.T) {
	own, _ := filepath.Glob("../../.github/workflows/workline*.yml")
	users := 0
	for _, f := range own {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var wf struct {
			Jobs map[string]struct {
				Steps []engineStep `yaml:"steps"`
			} `yaml:"jobs"`
		}
		if err := yaml.Unmarshal(data, &wf); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		for name, job := range wf.Jobs {
			checkout, engine, gates := -1, -1, -1
			for i, s := range job.Steps {
				switch {
				case strings.HasPrefix(s.Uses, "actions/checkout@") && checkout < 0:
					checkout = i
				case s.Uses == engineAction:
					engine = i
				case strings.HasPrefix(s.Uses, "actions/setup-go@"):
					t.Errorf("%s, job %s: sets Go up itself; the engine's action does, only when it builds", filepath.Base(f), name)
				}
				if strings.Contains(s.Run, "go build") || strings.Contains(s.Run, "go install") {
					t.Errorf("%s, job %s: builds the engine itself, not once a commit", filepath.Base(f), name)
				}
				if strings.Contains(s.Run, "--gates-only") {
					gates = i
				}
			}
			if engine < 0 {
				continue
			}
			users++
			// The action is the checkout's, and the commit it builds is the
			// one checked out.
			if checkout < 0 || checkout > engine {
				t.Errorf("%s, job %s: takes the engine before any checkout", filepath.Base(f), name)
			}
			// Kept before the gates' tools, the change's code, run.
			if gates >= 0 && gates < engine {
				t.Errorf("%s, job %s: keeps the engine after the gates ran", filepath.Base(f), name)
			}
		}
	}
	// gates and judge; apply, verdict and follow; merge's follow; the
	// gardening's two jobs and the sample's two.
	if users < 10 {
		t.Errorf("found %d jobs taking the engine from %s, want 10", users, engineAction)
	}

	steps := readEngineAction(t)
	var restore, save engineStep
	check, saved := -1, -1
	for i, s := range steps {
		switch {
		case strings.HasPrefix(s.Uses, "actions/cache/restore@"):
			restore = s
		case strings.HasPrefix(s.Uses, "actions/cache/save@"):
			save, saved = s, i
		case strings.HasPrefix(s.Uses, "actions/cache@"):
			t.Errorf("the engine's action uses actions/cache, which saves whatever the job leaves at its end")
		case s.Name == "The engine is the commit's":
			check = i
		}
	}
	if restore.Uses == "" || save.Uses == "" || check < 0 {
		t.Fatalf("the engine's action does not restore, check, then save the engine: %+v", steps)
	}
	if _, ok := restore.With["restore-keys"]; ok {
		t.Errorf("the engine's action reads the cache by a prefix: an older commit's engine could come back")
	}
	if restore.With["key"] != save.With["key"] || restore.With["path"] != save.With["path"] {
		t.Errorf("the engine is read and kept under different keys or paths: %v, %v", restore.With, save.With)
	}
	if saved < check || !strings.Contains(save.If, "cache-hit != 'true'") {
		t.Errorf("the engine is kept before it is checked, or kept again when taken (if %q)", save.If)
	}
}

// The check runs as the action holds it, on stand-ins for the engine: an
// executable holding Go's build information as a build writes it.
func TestOwnEngineIsTheCommitsOrNothingRuns(t *testing.T) {
	var run string
	for _, s := range readEngineAction(t) {
		if s.Name == "The engine is the commit's" {
			run = s.Run
			if s.Env["SHA"] != "${{ steps.key.outputs.sha }}" {
				t.Errorf("the check is given %q, not the commit checked out", s.Env["SHA"])
			}
		}
	}
	if run == "" {
		t.Fatal("the engine's action holds no check of the engine's commit")
	}
	const head, other = "0123456789abcdef0123456789abcdef01234567", "89abcdef0123456789abcdef0123456789abcdef"
	for _, c := range []struct {
		name, sha, info string
		missing, runs   bool
	}{
		{name: "the commit's, unmodified", sha: head, info: "vcs.revision=" + head + "\nvcs.modified=false", runs: true},
		{name: "an older commit's", sha: head, info: "vcs.revision=" + other + "\nvcs.modified=false"},
		{name: "the commit's, modified", sha: head, info: "vcs.revision=" + head + "\nvcs.modified=true"},
		{name: "built with no commit", sha: head, info: "-buildvcs=false"},
		{name: "no engine", sha: head, missing: true},
		{name: "no commit to check", sha: "", info: "vcs.revision=\nvcs.modified=false"},
	} {
		dir := t.TempDir()
		bin := filepath.Join(dir, "workline-engine")
		if err := os.MkdirAll(bin, 0o755); err != nil {
			t.Fatal(err)
		}
		if !c.missing {
			stand := "#!/bin/sh\n: '\n" + c.info + "\n'\necho v0.0.0-stand-in\n"
			if err := os.WriteFile(filepath.Join(bin, "workline"), []byte(stand), 0o755); err != nil {
				t.Fatal(err)
			}
		}
		path := filepath.Join(dir, "path")
		cmd := exec.Command("bash", "--noprofile", "--norc", "-eo", "pipefail", "-c", run)
		cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "SHA=" + c.sha, "RUNNER_TEMP=" + dir, "GITHUB_PATH=" + path}
		out, err := cmd.CombinedOutput()
		added, _ := os.ReadFile(path)
		ran := err == nil && slices.Contains(strings.Fields(string(added)), bin) && strings.Contains(string(out), "v0.0.0-stand-in")
		if ran != c.runs {
			t.Errorf("%s: runs %v (%v), want %v\n%s", c.name, ran, err, c.runs, out)
		}
		if !c.runs && (len(added) > 0 || !strings.Contains(string(out), "::error title=workline::")) {
			t.Errorf("%s: not refused loudly, or the engine put on PATH still:\n%s", c.name, out)
		}
	}
}
