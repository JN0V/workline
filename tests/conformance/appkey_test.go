package conformance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// The GitHub App's key never reaches a pull request's own run, whose
// workflow its branch can change (ADR-0044, #331): no job of a workflow a
// pull request runs names it, but to say whether it is there. Each job using
// it runs the default branch's workflow, in the environment workline-app,
// which only the default branch may use. One reading a pull request's run
// takes the event's head, reads the run's artifacts outside the checkout,
// and checks the base out, never the head.

const appKey = "secrets.WORKLINE_APP_PRIVATE_KEY"

// prEvents are the events whose run is the pull request's own workflow.
var prEvents = []string{"pull_request", "pull_request_review", "pull_request_review_comment"}

// baseEvents are the events whose run is the default branch's workflow
// (pull_request_target's since 2025-12-08), or one a writer starts on a
// branch, which the environment's branch rule refuses (workflow_dispatch).
var baseEvents = []string{"push", "schedule", "workflow_dispatch", "workflow_run", "pull_request_target"}

type keyStep struct {
	Uses string            `yaml:"uses"`
	With map[string]any    `yaml:"with"`
	Env  map[string]string `yaml:"env"`
	Run  string            `yaml:"run"`
}

type keyJob struct {
	Needs       any       `yaml:"needs"`
	If          string    `yaml:"if"`
	Environment any       `yaml:"environment"`
	Env         yaml.Node `yaml:"env"`
	Steps       []keyStep `yaml:"steps"`
}

func TestGitHubAppKeyOutOfPullRequestRuns(t *testing.T) {
	files, _ := filepath.Glob("../../ci/github/*.yml")
	own, _ := filepath.Glob("../../.github/workflows/*.yml")
	users, warned := 0, 0
	for _, f := range append(files, own...) {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var wf struct {
			On   map[string]any       `yaml:"on"`
			Jobs map[string]yaml.Node `yaml:"jobs"`
		}
		if err := yaml.Unmarshal(data, &wf); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		name := strings.TrimPrefix(f, "../../")
		var events []string
		for e := range wf.On {
			events = append(events, e)
		}
		ofPR := slices.ContainsFunc(events, func(e string) bool { return slices.Contains(prEvents, e) })
		for jobName, node := range wf.Jobs {
			var job keyJob
			if err := node.Decode(&job); err != nil {
				t.Fatalf("%s, job %s: %v", name, jobName, err)
			}
			text, _ := yaml.Marshal(&node)
			// Whether it is there, never its value: a pull request's run
			// warns when the key reaches it.
			if strings.Contains(string(text), appKey+" != ''") {
				if !ofPR || !warns(job) {
					t.Errorf("%s, job %s: asks whether the App's key is there, without warning a pull request's run it is in its reach", name, jobName)
				}
				warned++
			}
			if !strings.Contains(strings.ReplaceAll(string(text), appKey+" != ''", ""), appKey) {
				continue
			}
			users++
			if ofPR {
				t.Errorf("%s, job %s: uses the App's key in a workflow a pull request runs (on %v)", name, jobName, events)
			}
			for _, e := range events {
				if !slices.Contains(baseEvents, e) {
					t.Errorf("%s, job %s: uses the App's key on %s, not the default branch's workflow", name, jobName, e)
				}
			}
			if p, present := wf.On["push"]; present {
				push, _ := p.(map[string]any)
				if b := fmt.Sprint(push["branches"]); b != "[main]" {
					t.Errorf("%s, job %s: uses the App's key on a push to %s, not to the default branch alone", name, jobName, b)
				}
			}
			if env := environmentOf(job.Environment); env != "workline-app" {
				t.Errorf("%s, job %s: uses the App's key outside the environment workline-app (%q)", name, jobName, env)
			}
			if _, ok := wf.On["workflow_run"]; ok {
				for _, p := range readsRunAsData(job) {
					t.Errorf("%s, job %s: %s", name, jobName, p)
				}
			}
			_, target := wf.On["pull_request_target"]
			if _, after := wf.On["workflow_run"]; target || after {
				for _, s := range job.Steps {
					if ref := fmt.Sprint(s.With["ref"]); strings.HasPrefix(s.Uses, "actions/checkout@") && strings.Contains(ref, "head") {
						t.Errorf("%s, job %s: checks out %s, the pull request's head, beside the App's key", name, jobName, ref)
					}
				}
			}
		}
	}
	// apply, verdict and follow; merge, gardening, the sample: in the
	// templates and here; release-please, here.
	if users < 13 {
		t.Errorf("found %d jobs using the App's key, want 13", users)
	}
	if warned < 2 {
		t.Errorf("found %d judges warning the App's key is in a pull request's reach, want the template's and workline's own", warned)
	}
}

// warns says whether a job warns, in a step of its own, when the key is in
// its reach.
func warns(job keyJob) bool {
	var env map[string]string
	job.Env.Decode(&env)
	if env["APP_KEY_IN_REACH"] != "${{ "+appKey+" != '' }}" {
		return false
	}
	for _, s := range job.Steps {
		if strings.Contains(s.Run, "::warning") && strings.Contains(s.Run, "environment workline-app") {
			return true
		}
	}
	return false
}

// environmentOf is the environment a job names: `environment: name`, or
// `environment: {name: …}`.
func environmentOf(v any) string {
	switch e := v.(type) {
	case string:
		return e
	case map[string]any:
		return fmt.Sprint(e["name"])
	}
	return ""
}

// readsRunAsData says what a job reading another job's run after it
// completed (workflow_run) does wrong: that run may have run a branch's
// code, the tester's answer too. Its artifacts outside the checkout, the
// head the event names, only what that pull request's line may do
// applied.
func readsRunAsData(job keyJob) (problems []string) {
	const head = "${{ github.event.workflow_run.head_sha }}"
	for _, s := range job.Steps {
		if strings.HasPrefix(s.Uses, "actions/download-artifact@") && !strings.Contains(fmt.Sprint(s.With["path"]), "runner.temp") {
			problems = append(problems, "downloads the run's artifacts into the checkout")
		}
		if strings.Contains(s.Run, "gh run download") && !strings.Contains(s.Run, `-D "$RUNNER_TEMP/`) {
			problems = append(problems, "downloads the run's artifacts into the checkout")
		}
		// Main's run after a merge (ADR-0050): only a push to the default
		// branch, whose workflow is main's, never another branch's.
		if strings.Contains(s.Run, "workline apply --main") {
			if !strings.Contains(job.If, "github.event.workflow_run.event == 'push'") || !strings.Contains(job.If, "github.event.workflow_run.head_branch == github.event.repository.default_branch") {
				problems = append(problems, fmt.Sprintf("keeps the findings of a run that may not be main's (if %q)", job.If))
			}
			continue
		}
		if strings.Contains(s.Run, "workline apply") {
			if !strings.Contains(s.Run, `--from-merge-request "$NUMBER" --head "$HEAD_SHA"`) || s.Env["HEAD_SHA"] != head {
				problems = append(problems, "applies a pull request's run without --from-merge-request and the event's head")
			}
			if s.Env["WORKLINE_APP"] != "${{ steps.app.outputs.app-slug }}" {
				problems = append(problems, "applies a pull request's run without naming the App, whose pushes alone let a release's fix be its merge request")
			}
		}
		if strings.Contains(s.Run, "--verdict-only") && s.Env["HEAD_SHA"] != head {
			problems = append(problems, "keeps a verdict for another head than the event's")
		}
	}
	return problems
}

// Every template pinning a release of workline is one release-please moves
// on each release (x-release-please-version): a template left out would
// stay on an old engine.
func TestReleasePleaseMovesEveryPin(t *testing.T) {
	data, err := os.ReadFile("../../release-please-config.json")
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Packages map[string]struct {
			ExtraFiles []struct {
				Path string `json:"path"`
			} `json:"extra-files"`
		} `json:"packages"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	var moved []string
	for _, p := range cfg.Packages {
		for _, e := range p.ExtraFiles {
			moved = append(moved, e.Path)
		}
	}
	files, _ := filepath.Glob("../../ci/*/*.yml")
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if p := strings.TrimPrefix(f, "../../"); strings.Contains(string(data), "x-release-please-version") && !slices.Contains(moved, p) {
			t.Errorf("%s pins a release, but release-please-config.json does not move it", p)
		}
	}
}

// A job holding the App's key after another run, the tester's answer as
// #656 first wrote it: the judge's folder downloaded into the checkout,
// applied whole, by the path it named. Each is caught.
func TestReadsRunAsDataRefusesAnAnswerAppliedWhole(t *testing.T) {
	job := keyJob{Needs: "judge", Steps: []keyStep{
		{Uses: "actions/download-artifact@v4", With: map[string]any{"name": "workline-answer", "path": ".workline-runs"}},
		{Run: `workline apply "$RUN" --summary workline-summary.md`, Env: map[string]string{"RUN": "${{ needs.judge.outputs.run }}", "WORKLINE_APP": "${{ steps.app.outputs.app-slug }}"}},
	}}
	got := readsRunAsData(job)
	for _, want := range []string{"downloads the run's artifacts into the checkout", "applies a pull request's run without --from-merge-request"} {
		if !slices.ContainsFunc(got, func(p string) bool { return strings.Contains(p, want) }) {
			t.Errorf("not caught: %s (found %q)", want, got)
		}
	}
}
