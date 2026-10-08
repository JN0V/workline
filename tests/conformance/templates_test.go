package conformance

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// The documentalist compares each doc's sources since the commit its
// `checked` names, and the release looks for its last tag: both need the
// whole history, which a CI's default shallow clone lacks. Every job of the
// CI templates fetches it.

func TestGitLabTemplateFetchesWholeHistory(t *testing.T) {
	data, err := os.ReadFile("../../ci/gitlab/workline.gitlab-ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	var top map[string]any
	if err := yaml.Unmarshal(data, &top); err != nil {
		t.Fatal(err)
	}
	jobs := map[string]map[string]any{}
	for name, v := range top {
		if job, ok := v.(map[string]any); ok && name != "variables" {
			jobs[name] = job
		}
	}
	depth := func(name string) string {
		for seen := 0; name != "" && seen < 10; seen++ {
			job := jobs[name]
			if vars, ok := job["variables"].(map[string]any); ok {
				if d, ok := vars["GIT_DEPTH"]; ok {
					return fmt.Sprint(d)
				}
			}
			name, _ = job["extends"].(string)
		}
		return ""
	}
	n := 0
	for name, job := range jobs {
		if strings.HasPrefix(name, ".") || job["script"] == nil {
			continue
		}
		n++
		if d := depth(name); d != "0" {
			t.Errorf("job %s: GIT_DEPTH is %q, want 0 (the whole history)", name, d)
		}
	}
	if n == 0 {
		t.Fatal("no job found in the GitLab template")
	}
}

func TestGitHubTemplatesFetchWholeHistory(t *testing.T) {
	files, _ := filepath.Glob("../../ci/github/*.yml")
	n := 0
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var wf struct {
			Jobs map[string]struct {
				Steps []struct {
					Uses string         `yaml:"uses"`
					With map[string]any `yaml:"with"`
				} `yaml:"steps"`
			} `yaml:"jobs"`
		}
		if err := yaml.Unmarshal(data, &wf); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		for name, job := range wf.Jobs {
			for _, s := range job.Steps {
				if !strings.HasPrefix(s.Uses, "actions/checkout@") {
					continue
				}
				n++
				if d := fmt.Sprint(s.With["fetch-depth"]); d != "0" {
					t.Errorf("%s, job %s: checkout's fetch-depth is %s, want 0 (the whole history)", filepath.Base(f), name, d)
				}
			}
		}
	}
	if n == 0 {
		t.Fatal("no checkout found in the GitHub templates")
	}
}

// A person opening a workline job reads what ran, each step's verdict, the
// findings and what waits to be applied in one place: the engine writes it
// (--summary), the templates show it, none rebuilds it from the JSON.

func TestGitLabTemplateKeepsTheSummary(t *testing.T) {
	data, err := os.ReadFile("../../ci/gitlab/workline.gitlab-ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	var top map[string]any
	if err := yaml.Unmarshal(data, &top); err != nil {
		t.Fatal(err)
	}
	n := 0
	for name, v := range top {
		job, ok := v.(map[string]any)
		if !ok || strings.HasPrefix(name, ".") {
			continue
		}
		script := fmt.Sprint(job["script"])
		if !strings.Contains(script, "workline route") && !strings.Contains(script, "workline apply") && !strings.Contains(script, "workline sample") {
			continue
		}
		n++
		if !strings.Contains(script, "--summary workline-summary.md") || !strings.Contains(script, "--summary workline-summary.html") {
			t.Errorf("job %s: workline is not given --summary workline-summary.md and .html", name)
		}
		art, _ := job["artifacts"].(map[string]any)
		if !strings.Contains(fmt.Sprint(art["paths"]), "workline-summary.html") || fmt.Sprint(art["when"]) != "always" {
			t.Errorf("job %s: workline-summary.md is not kept as an artifact, always: %v", name, art)
		}
		if !strings.Contains(fmt.Sprint(art["reports"]), "annotations:workline-annotations.json") ||
			!strings.Contains(fmt.Sprint(job["after_script"]), "artifacts/file/workline-summary.html") {
			t.Errorf("job %s: the job's page does not link the summary (annotations)", name)
		}
		if strings.Contains(script, "--target merge-request") && art["expose_as"] == nil {
			t.Errorf("job %s: the merge request's page does not link the summary (expose_as)", name)
		}
	}
	if n < 6 {
		t.Fatalf("found %d jobs running workline, want the six judging and applying ones", n)
	}
}

func TestGitHubTemplatesUseTheSummary(t *testing.T) {
	files, _ := filepath.Glob("../../ci/github/workline*.yml")
	own, _ := filepath.Glob("../../.github/workflows/workline*.yml")
	n := 0
	for _, f := range append(files, own...) {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		if strings.Contains(text, `jq -r '"\(.status)`) {
			t.Errorf("%s: builds the summary with jq; the engine writes it (--summary)", filepath.Base(f))
		}
		if !strings.Contains(text, "GITHUB_STEP_SUMMARY") {
			continue
		}
		n++
		if !strings.Contains(text, "--summary workline-summary.md") || !strings.Contains(text, `cat workline-summary.md >> "$GITHUB_STEP_SUMMARY"`) {
			t.Errorf("%s: the job summary is not the engine's --summary", filepath.Base(f))
		}
	}
	if n < 6 {
		t.Fatalf("found %d workflows writing a job summary, want six", n)
	}
}

// The job that applies writes its own summary: what was done, and what
// failed or was left unapplied, on its page, the job failing still.
func TestGitHubApplyJobsWriteTheirSummary(t *testing.T) {
	files, _ := filepath.Glob("../../ci/github/workline*.yml")
	own, _ := filepath.Glob("../../.github/workflows/workline*.yml")
	n := 0
	for _, f := range append(files, own...) {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var wf struct {
			Jobs map[string]struct {
				Steps []struct {
					Run string `yaml:"run"`
				} `yaml:"steps"`
			} `yaml:"jobs"`
		}
		if err := yaml.Unmarshal(data, &wf); err != nil {
			t.Fatal(err)
		}
		for name, job := range wf.Jobs {
			for _, st := range job.Steps {
				if !strings.Contains(st.Run, "workline apply") && !strings.Contains(st.Run, "workline sample --apply") {
					continue
				}
				n++
				if !strings.Contains(st.Run, "--summary workline-summary.md || status=$?") ||
					!strings.Contains(st.Run, `cat workline-summary.md >> "$GITHUB_STEP_SUMMARY"`) ||
					!strings.Contains(st.Run, `exit "$status"`) {
					t.Errorf("%s, job %s: applies without its own summary on the job's page, or without failing by it", filepath.Base(f), name)
				}
			}
		}
	}
	if n < 6 {
		t.Fatalf("found %d steps applying, want six", n)
	}
}

// A judge that fails — a step blocked, the reviewer held by a finding —
// still leaves what it proposed to apply (#226, #142): the job that applies
// runs whatever the judge's outcome, and the pipeline still fails by the
// judge's. An agent out of reach (3) warns, as on GitHub, never fails.

func TestGitLabApplyRunsWhenTheJudgeFails(t *testing.T) {
	data, err := os.ReadFile("../../ci/gitlab/workline.gitlab-ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	var top map[string]any
	if err := yaml.Unmarshal(data, &top); err != nil {
		t.Fatal(err)
	}
	applies, judges := 0, 0
	for name, v := range top {
		job, ok := v.(map[string]any)
		if !ok || strings.HasPrefix(name, ".") {
			continue
		}
		script := fmt.Sprint(job["script"])
		switch {
		case strings.Contains(script, "workline apply"):
			applies++
			rules, _ := job["rules"].([]any)
			if len(rules) == 0 {
				t.Errorf("job %s: no rules of its own: GitLab skips it when the judge fails (on_success)", name)
			}
			for _, r := range rules {
				if m, _ := r.(map[string]any); fmt.Sprint(m["when"]) != "always" {
					t.Errorf("job %s: rule %v is not `when: always`: a failed judge would skip it", name, m)
				}
			}
		case strings.Contains(script, "workline route"):
			judges++
			af, _ := job["allow_failure"].(map[string]any)
			if fmt.Sprint(af["exit_codes"]) != "[3]" {
				t.Errorf("job %s: allow_failure %v: an agent out of reach (3) fails it, where GitHub warns", name, job["allow_failure"])
			}
		}
	}
	if applies < 2 || judges < 2 {
		t.Fatalf("found %d applying and %d judging jobs, want the merge request's and the gardening's", applies, judges)
	}
}

func TestGitHubApplyRunsWhenTheJudgeFails(t *testing.T) {
	files, _ := filepath.Glob("../../ci/github/workline*.yml")
	own, _ := filepath.Glob("../../.github/workflows/workline*.yml")
	n := 0
	for _, f := range append(files, own...) {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var wf struct {
			Jobs map[string]struct {
				If    string `yaml:"if"`
				Steps []struct {
					Run string `yaml:"run"`
				} `yaml:"steps"`
			} `yaml:"jobs"`
		}
		if err := yaml.Unmarshal(data, &wf); err != nil {
			t.Fatal(err)
		}
		for name, job := range wf.Jobs {
			for _, s := range job.Steps {
				if !strings.Contains(s.Run, "workline apply") {
					continue
				}
				n++
				if !strings.HasPrefix(job.If, "always()") {
					t.Errorf("%s, job %s: if %q: a failed judge would skip it", filepath.Base(f), name, job.If)
				}
			}
		}
	}
	if n < 4 {
		t.Fatalf("found %d jobs running workline apply, want the merge request's and the gardening's, here and in workline's own", n)
	}
}
