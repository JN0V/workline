package github

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// tallyStep is a step of a workflow that writes or uploads the job's
// tally: the step that runs `workline sample --tally`, or the upload
// of the artifact workline-tally-….
type tallyStep struct {
	where     string // file and job
	cond      string
	upload    bool
	name      string // the artifact's name, for an upload
	overwrite bool
}

// tallySteps reads every workflow template the project ships
// (ci/github/*.yml). It never reads the project's own workflows
// (.github/workflows/*.yml): those are not the developer's to change.
func tallySteps(t *testing.T) []tallyStep {
	t.Helper()
	files, _ := filepath.Glob("*.yml")
	var out []tallyStep
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		var w struct {
			Jobs map[string]struct {
				Steps []struct {
					If   string         `yaml:"if"`
					Uses string         `yaml:"uses"`
					Run  string         `yaml:"run"`
					With map[string]any `yaml:"with"`
				} `yaml:"steps"`
			} `yaml:"jobs"`
		}
		if err := yaml.Unmarshal(data, &w); err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		for job, j := range w.Jobs {
			for _, s := range j.Steps {
				name := fmt.Sprint(s.With["name"])
				switch {
				case strings.Contains(s.Uses, "upload-artifact") && strings.Contains(name, "workline-tally"):
					out = append(out, tallyStep{where: file + " " + job, cond: s.If, upload: true, name: name, overwrite: s.With["overwrite"] == true})
				case strings.Contains(s.Run, "sample --tally"):
					out = append(out, tallyStep{where: file + " " + job, cond: s.If})
				}
			}
		}
	}
	if len(out) == 0 {
		t.Fatal("no step writing or uploading a tally found")
	}
	return out
}

// Cancelled run kept
// Given a reviewer run cancelled by a newer push after its agent answered,
// When the weekly sample runs,
// Then that run's tokens are counted: the steps that write and upload the
// job's tally run in a cancelled run too. It reads the templates the
// project ships (ci/github/*.yml) only.
func TestTallyStepsRunInACancelledRun(t *testing.T) {
	for _, s := range tallySteps(t) {
		rest := strings.ReplaceAll(s.cond, "!cancelled()", "")
		if !strings.Contains(rest, "always()") && !strings.Contains(rest, "cancelled()") {
			t.Errorf("%s: the tally step runs `if: %s`, which never runs in a cancelled run", s.where, s.cond)
		}
	}
}

// Re-run kept
// Given a job re-run in the same workflow run,
// When the weekly sample runs,
// Then both attempts' tokens are counted: each attempt uploads its own
// artifact, none overwrites another's. It reads the templates the
// project ships (ci/github/*.yml) only.
func TestTallyArtifactsKeepEveryAttempt(t *testing.T) {
	uploads := 0
	for _, s := range tallySteps(t) {
		if !s.upload {
			continue
		}
		uploads++
		if s.overwrite {
			t.Errorf("%s: the tally artifact %q overwrites the one an earlier attempt uploaded", s.where, s.name)
		}
		if !strings.Contains(s.name, "run_attempt") {
			t.Errorf("%s: the tally artifact %q is named the same in every attempt of a run", s.where, s.name)
		}
	}
	if uploads == 0 {
		t.Error("no tally artifact upload found in the templates")
	}
}
