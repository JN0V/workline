package conformance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/line"
	"github.com/JN0V/workline/internal/verdict"
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
		if vars, _ := job["variables"].(map[string]any); fmt.Sprint(vars["GIT_STRATEGY"]) == "none" {
			continue // checks nothing out (workline:base): no history to lack
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
		if !strings.Contains(script, "workline route") && !strings.Contains(script, "workline apply") && !strings.Contains(script, "workline sample") &&
			!strings.Contains(script, "workline follow") {
			continue
		}
		n++
		// The gates' job names its own: the judge, given its artifacts,
		// writes workline-summary.md afresh.
		sum := "workline-summary"
		if strings.Contains(script, "--gates-only") {
			sum = "gates-summary"
		}
		if !strings.Contains(script, "--summary "+sum+".md") || !strings.Contains(script, "--summary "+sum+".html") {
			t.Errorf("job %s: workline is not given --summary %s.md and .html", name, sum)
		}
		art, _ := job["artifacts"].(map[string]any)
		if !strings.Contains(fmt.Sprint(art["paths"]), sum+".html") || fmt.Sprint(art["when"]) != "always" {
			t.Errorf("job %s: %s.md is not kept as an artifact, always: %v", name, sum, art)
		}
		if !strings.Contains(fmt.Sprint(art["reports"]), "annotations:workline-annotations.json") ||
			!strings.Contains(fmt.Sprint(job["after_script"]), "artifacts/file/"+sum+".html") {
			t.Errorf("job %s: the job's page does not link the summary (annotations)", name)
		}
		// A merge request's pipeline; not a scheduled one, whose page is the job's.
		if strings.Contains(script, "--target merge-request:$") && art["expose_as"] == nil {
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
// failed or was left unapplied, on its page, the job failing still; the
// job following main too, which names each pull request it merged main
// into, and each it left (ADR-0041).
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
				if !strings.Contains(st.Run, "workline apply") && !strings.Contains(st.Run, "workline sample --apply") && !strings.Contains(st.Run, "workline follow") {
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
		case strings.Contains(script, "--warn-only"):
			// What only warns never fails the pipeline (ADR-0050).
			if job["allow_failure"] != true {
				t.Errorf("job %s: allow_failure %v: what only warns would fail the pipeline", name, job["allow_failure"])
			}
		case name == "workline:release":
			// At a tag, a release held, or whose checks could not run
			// (3), is never published: no failure allowed
			// (TestGitLabReleaseJobAtATag).
		case strings.Contains(script, "workline route") && !strings.Contains(script, "--gates-only"):
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
			On struct {
				WorkflowRun *struct {
					Types []string `yaml:"types"`
				} `yaml:"workflow_run"`
			} `yaml:"on"`
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
		// After the judge's workflow, run on its completion: whatever its
		// conclusion, unless the job's condition reads it — but for a run a
		// newer push cancelled (#359).
		afterAny := wf.On.WorkflowRun != nil && slices.Equal(wf.On.WorkflowRun.Types, []string{"completed"})
		for name, job := range wf.Jobs {
			for _, s := range job.Steps {
				if !strings.Contains(s.Run, "workline apply") {
					continue
				}
				n++
				reads := strings.Contains(strings.ReplaceAll(job.If, notCancelled, ""), "conclusion")
				if afterAny && reads || !afterAny && !strings.HasPrefix(job.If, "always()") {
					t.Errorf("%s, job %s: if %q: a failed judge would skip it", filepath.Base(f), name, job.If)
				}
			}
		}
	}
	if n < 4 {
		t.Fatalf("found %d jobs running workline apply, want the merge request's and the gardening's, here and in workline's own", n)
	}
}

// notCancelled is how a job after the judge's workflow skips a run a
// newer push cancelled.
const notCancelled = "github.event.workflow_run.conclusion != 'cancelled'"

// One line per merge request at a time (#359): a push cancels the run
// still judging the head before — on GitHub, the workflow's concurrency
// group per pull request; on GitLab, an interruptible judge —, and nothing of a cancelled
// run is uploaded after, applied, kept, followed or commented. The jobs that
// write on a merge request's pipeline are never interruptible: one begun
// goes to its end.
// ownGroup is a concurrency group as a pull request's run reads it: a
// merge queue's group takes one of its own (TestGitHubMergeQueueRunsTheGatesAlone).
func ownGroup(group string) string {
	for _, other := range []string{" || github.event.merge_group.head_ref || github.sha }}", " || github.event.merge_group.head_ref }}", " || github.sha }}"} {
		group = strings.ReplaceAll(group, other, " }}")
	}
	return group
}

func TestOneLinePerMergeRequestAtATime(t *testing.T) {
	type workflow struct {
		Concurrency map[string]any `yaml:"concurrency"`
		Jobs        map[string]struct {
			If          string         `yaml:"if"`
			Concurrency map[string]any `yaml:"concurrency"`
			Steps       []struct {
				If  string `yaml:"if"`
				Run string `yaml:"run"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	read := func(f string) workflow {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var wf workflow
		if err := yaml.Unmarshal(data, &wf); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		return wf
	}
	for _, dir := range []string{"../../ci/github/", "../../.github/workflows/"} {
		judges := 0
		wf := read(dir + "workline.yml")
		if group := ownGroup(fmt.Sprint(wf.Concurrency["group"])); !strings.Contains(group, "${{ github.event.pull_request.number }}") || wf.Concurrency["cancel-in-progress"] != true {
			t.Errorf("%sworkline.yml: no concurrency group per pull request cancelling the run in progress: %v", dir, wf.Concurrency)
		}
		// The run apart from the verdict (ADR-0050): one per pull request
		// too, in a group of its own, never cancelling the judge's.
		warn := read(dir + "workline-warn.yml")
		if group := fmt.Sprint(warn.Concurrency["group"]); !strings.Contains(group, "${{ github.event.pull_request.number }}") || group == fmt.Sprint(wf.Concurrency["group"]) || warn.Concurrency["cancel-in-progress"] != true {
			t.Errorf("%sworkline-warn.yml: no concurrency group of its own per pull request cancelling the run in progress: %v", dir, warn.Concurrency)
		}
		for name, job := range warn.Jobs {
			if strings.Contains(job.If, "always()") {
				t.Errorf("%sworkline-warn.yml, job %s: runs on always(), a cancelled run's too: %q", dir, name, job.If)
			}
			for _, s := range job.Steps {
				if strings.Contains(s.If, "always()") {
					t.Errorf("%sworkline-warn.yml, job %s: a step runs on always(), a cancelled run's too: %q", dir, name, s.If)
				}
			}
		}
		for name, job := range wf.Jobs {
			route := false
			for _, s := range job.Steps {
				route = route || strings.Contains(s.Run, "workline route merge-request")
				if strings.Contains(s.If, "always()") {
					t.Errorf("%sworkline.yml, job %s: a step runs on always(), a cancelled run's too: %q", dir, name, s.If)
				}
			}
			if !route {
				continue
			}
			judges++
		}
		if judges == 0 {
			t.Errorf("%sworkline.yml: no job runs the merge request's line", dir)
		}
		for _, f := range []string{"workline-apply.yml", "workline-fork.yml"} {
			for name, job := range read(dir + f).Jobs {
				if !strings.Contains(job.If, notCancelled) {
					t.Errorf("%s%s, job %s: if %q: it runs after a cancelled run too", dir, f, name, job.If)
				}
			}
		}
	}
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
		rules := job["rules"]
		if ext, ok := job["extends"]; rules == nil && ok {
			base, _ := ext.(string)
			parent, found := top[base].(map[string]any)
			if !found {
				t.Fatalf("job %s extends %v, which the template does not hold as one job: the check reads one", name, ext)
			}
			rules = parent["rules"]
		}
		if !strings.Contains(fmt.Sprint(rules), "merge_request_event") {
			continue
		}
		n++
		// Every job that writes nothing (the judge, the gates, what reads
		// their base) is interruptible: one that is not, once started,
		// keeps GitLab from cancelling the whole pipeline.
		script := fmt.Sprint(job["script"])
		judge := !strings.Contains(script, "workline apply") && !strings.Contains(script, "workline follow")
		why := "a job that writes goes to its end"
		if judge {
			why = "a push must cancel the judge still on the head before"
		}
		if got, set := job["interruptible"]; !set || got != judge {
			t.Errorf("job %s: interruptible %v, want %v: %s", name, got, judge, why)
		}
	}
	if n < 3 {
		t.Fatalf("found %d jobs of a merge request's pipeline, want the judge and the two that write", n)
	}
}

// One review at a time per merge request (#375): the judge and the apply
// share a lock, neither cancelling the other, so a push is judged once
// the review before it is written, and reads its record. On GitHub, a
// concurrency group per pull request; on GitLab, a resource group per
// merge request.
func TestOneReviewAtATimePerMergeRequest(t *testing.T) {
	type workflow struct {
		Jobs map[string]struct {
			Concurrency map[string]any `yaml:"concurrency"`
			Steps       []struct {
				Run string `yaml:"run"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	// The review apart from the verdict (ADR-0050) holds a lock of its own,
	// with the apply of its runs: its record is its own comment's.
	const (
		review = "workline-review-${{ github.event.pull_request.number }}"
		apart  = "workline-warn-review-${{ github.event.pull_request.number }}"
		apply  = "workline-${{ github.event.workflow_run.name == 'workline warn' && 'warn-review' || 'review' }}-${{ github.event.workflow_run.pull_requests[0].number }}"
	)
	locks := func(file, command, want string) {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		var wf workflow
		if err := yaml.Unmarshal(data, &wf); err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		n := 0
		for name, job := range wf.Jobs {
			runs := false
			for _, s := range job.Steps {
				runs = runs || strings.Contains(s.Run, command) && !strings.Contains(s.Run, "--gates-only")
			}
			if !runs {
				continue
			}
			n++
			if group := ownGroup(fmt.Sprint(job.Concurrency["group"])); group != want || job.Concurrency["cancel-in-progress"] != false {
				t.Errorf("%s, job %s: not in the review's lock per pull request %q, never cancelling: %v", file, name, want, job.Concurrency)
			}
		}
		if n == 0 {
			t.Errorf("%s: no job runs %q", file, command)
		}
	}
	for _, dir := range []string{"../../ci/github/", "../../.github/workflows/"} {
		locks(dir+"workline.yml", "workline route merge-request", review)
		locks(dir+"workline-warn.yml", "workline route merge-request --warn-only", apart)
		locks(dir+"workline-apply.yml", "--from-merge-request", apply)
	}
	data, err := os.ReadFile("../../ci/gitlab/workline.gitlab-ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	var top map[string]any
	if err := yaml.Unmarshal(data, &top); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{"workline:judge": "workline-review-$CI_MERGE_REQUEST_IID", "workline:apply": "workline-review-$CI_MERGE_REQUEST_IID",
		"workline:warn": "workline-warn-$CI_MERGE_REQUEST_IID", "workline:warn:apply": "workline-warn-$CI_MERGE_REQUEST_IID"} {
		job, _ := top[name].(map[string]any)
		if got := job["resource_group"]; got != want {
			t.Errorf("job %s: resource_group %v, want the review's lock per merge request, %s", name, got, want)
		}
	}
}

// The judge says when no workflow reads the lenses it left apart, and the
// run apart when no workflow applies what it proposed: each check runs as
// its workflow holds it, in a folder of workflows with and without the
// other, and never takes its own text for the other (ADR-0050).
func TestWarnApartSaysWhatIsMissing(t *testing.T) {
	type workflow struct {
		Jobs map[string]struct {
			Steps []struct {
				Name string `yaml:"name"`
				Run  string `yaml:"run"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	script := func(file, step string) string {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		var wf workflow
		if err := yaml.Unmarshal(data, &wf); err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		for _, job := range wf.Jobs {
			for _, s := range job.Steps {
				if s.Name == step {
					return s.Run
				}
			}
		}
		t.Fatalf("%s: no step %q", file, step)
		return ""
	}
	// run runs a step's script in a folder holding these workflows and, with
	// reviewer, a config routing the reviewer (routed, when set, the roles
	// it routes instead); it says whether the step warned, and whether it
	// said the run apart reads (any=true), and triage (triage=true).
	routed, triage := "", false
	run := func(script string, files map[string]string, reviewer bool) (warned, reads bool) {
		dir := t.TempDir()
		os.MkdirAll(filepath.Join(dir, ".github", "workflows"), 0o755)
		os.MkdirAll(filepath.Join(dir, ".workline"), 0o755)
		roles := "reviewer"
		if routed != "" {
			roles = routed
		}
		if reviewer {
			os.WriteFile(filepath.Join(dir, ".workline", "config.yaml"), []byte("routing: {events: {merge-request: [committer, "+roles+"]}}\n"), 0o644)
		}
		for name, from := range files {
			data, err := os.ReadFile(from)
			if err != nil {
				t.Fatal(err)
			}
			if name == "old-apply" { // an apply copied before the run apart
				old := strings.Replace(string(data), "workflows: [workline, workline warn]", "workflows: [workline]", 1)
				if old == string(data) {
					t.Fatalf("%s: no `workflows: [workline, workline warn]` to make an older apply of", from)
				}
				name, data = "workline-apply.yml", []byte(old)
			}
			os.WriteFile(filepath.Join(dir, ".github", "workflows", name), data, 0o644)
		}
		output := filepath.Join(dir, "output")
		cmd := exec.Command("sh", "-ec", script)
		cmd.Dir, cmd.Env = dir, append(os.Environ(), "GITHUB_OUTPUT="+output)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s: %v\n%s", script, err, out)
		}
		said, _ := os.ReadFile(output)
		triage = strings.Contains(string(said), "triage=true")
		return strings.Contains(string(out), "::warning"), strings.Contains(string(said), "any=true")
	}
	warns := func(script string, files map[string]string) bool {
		warned, _ := run(script, files, true)
		return warned
	}
	for _, dir := range []string{"../../ci/github/", "../../.github/workflows/"} {
		judge := script(dir+"workline.yml", "Whether anything reads the lenses that warn")
		if !warns(judge, map[string]string{"workline.yml": dir + "workline.yml", "workline-apply.yml": dir + "workline-apply.yml"}) {
			t.Errorf("%sworkline.yml: no warning with no workline-warn.yml beside it", dir)
		}
		if warns(judge, map[string]string{"workline.yml": dir + "workline.yml", "workline-warn.yml": dir + "workline-warn.yml"}) {
			t.Errorf("%sworkline.yml: warns with workline-warn.yml beside it", dir)
		}
		// The run apart reads only where something applies what it
		// proposes, said when nothing does; with no reviewer, nothing.
		apart := script(dir+"workline-warn.yml", "Whether anything reads apart")
		if warned, reads := run(apart, map[string]string{"workline-warn.yml": dir + "workline-warn.yml", "old-apply": dir + "workline-apply.yml"}, true); !warned || reads {
			t.Errorf("%sworkline-warn.yml: with an apply that does not name it, warned %v, reads %v; want a warning, no read", dir, warned, reads)
		}
		if warned, reads := run(apart, map[string]string{"workline-warn.yml": dir + "workline-warn.yml", "workline-apply.yml": dir + "workline-apply.yml"}, true); warned || !reads {
			t.Errorf("%sworkline-warn.yml: with this release's apply, warned %v, reads %v; want a read, no warning", dir, warned, reads)
		}
		if warned, reads := run(apart, map[string]string{"workline-warn.yml": dir + "workline-warn.yml", "workline-apply.yml": dir + "workline-apply.yml"}, false); warned || reads {
			t.Errorf("%sworkline-warn.yml: with no reviewer, warned %v, reads %v; want neither", dir, warned, reads)
		}
		if _, reads := run(apart, map[string]string{"workline-warn.yml": dir + "workline-warn.yml", "workline-apply.yml": dir + "workline-apply.yml"}, true); triage {
			t.Errorf("%sworkline-warn.yml: the reviewer alone routed, reads %v, and waits for the gates' result: no inspector triages it", dir, reads)
		}
		// The inspector alone reads apart too, waiting for the gates'
		// result it triages (ADR-0049).
		routed = "\"gate:lint\", inspector"
		if warned, reads := run(apart, map[string]string{"workline-warn.yml": dir + "workline-warn.yml", "workline-apply.yml": dir + "workline-apply.yml"}, true); warned || !reads || !triage {
			t.Errorf("%sworkline-warn.yml: with the inspector alone, warned %v, reads %v, triage %v; want a read and the gates' result, no warning", dir, warned, reads, triage)
		}
		routed = ""
		triageJudge := script(dir+"workline.yml", "The line for this pull request (proposals only)")
		if !strings.Contains(triageJudge, `select(.name == "reviewer" or .name == "inspector")`) {
			t.Errorf("%sworkline.yml: the judge says not that nothing reads the inspector's triage it left apart", dir)
		}
		if line := script(dir+"workline-warn.yml", "What only warns, apart from the verdict (proposals only)"); !strings.Contains(line, `${gates:+--gates-from "$gates"}`) {
			t.Errorf("%sworkline-warn.yml: the run apart is not given the gates' result for the inspector", dir)
		}
		applies := script(dir+"workline.yml", "Whether anything applies what it proposed")
		if warns(applies, map[string]string{"workline.yml": dir + "workline.yml", "workline-apply.yml": dir + "workline-apply.yml"}) {
			t.Errorf("%sworkline.yml: says nothing applies, with this release's apply beside it", dir)
		}
	}
}

// What only warns runs apart from the verdict, never holding it (ADR-0050):
// the judge leaves the reviewer's lenses that warn (--warn-apart), a run of
// their own reads them (--warn-only), writing no verdict and uploading no
// SARIF, and fails no check by what it finds; the head's verdict is kept
// from the judge's run alone. On GitLab, that job and its apply never fail
// the pipeline, and the merge policy reads the judge alone.
func TestWarnApartNeverHoldsTheVerdict(t *testing.T) {
	type workflow struct {
		On struct {
			WorkflowRun *struct {
				Workflows []string `yaml:"workflows"`
			} `yaml:"workflow_run"`
		} `yaml:"on"`
		Jobs map[string]struct {
			If    string `yaml:"if"`
			Steps []struct {
				Uses string         `yaml:"uses"`
				With map[string]any `yaml:"with"`
				Run  string         `yaml:"run"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	read := func(f string) (workflow, string) {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var wf workflow
		if err := yaml.Unmarshal(data, &wf); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		return wf, string(data)
	}
	for _, dir := range []string{"../../ci/github/", "../../.github/workflows/"} {
		judge, _ := read(dir + "workline.yml")
		left := false
		for _, job := range judge.Jobs {
			for _, s := range job.Steps {
				left = left || strings.Contains(s.Run, "workline route merge-request") && strings.Contains(s.Run, "--warn-apart")
			}
		}
		if !left {
			t.Errorf("%sworkline.yml: the judge's line does not leave what only warns apart (--warn-apart)", dir)
		}
		warn, text := read(dir + "workline-warn.yml")
		if !strings.Contains(text, "workline route merge-request --warn-only") {
			t.Errorf("%sworkline-warn.yml: does not run what only warns alone (--warn-only)", dir)
		}
		for _, bad := range []string{"--verdict", "--sarif", "workline-verdict", "upload-sarif", "secrets.WORKLINE_APP"} {
			if strings.Contains(text, bad) {
				t.Errorf("%sworkline-warn.yml: holds %q: the run apart gives no verdict, nor findings to code scanning", dir, bad)
			}
		}
		reads, says := false, false
		for name, job := range warn.Jobs {
			for _, s := range job.Steps {
				reads = reads || strings.Contains(s.Run, "workline route merge-request --warn-only")
				if !strings.Contains(s.Run, "exit") {
					continue
				}
				// The run's status: 0 passes, 3 (the agent out of reach) is
				// said, anything else is the engine's own error.
				says = strings.Contains(s.Run, `3) echo "::warning`) && strings.Contains(s.Run, "0) ;;")
				if !says {
					t.Errorf("%sworkline-warn.yml, job %s: a step exits by the run's status without passing 0 and saying 3", dir, name)
				}
			}
		}
		if !reads || !says {
			t.Errorf("%sworkline-warn.yml: no step runs the line apart (%v), or none ends the job by its status (%v)", dir, reads, says)
		}
		apply, _ := read(dir + "workline-apply.yml")
		if r := apply.On.WorkflowRun; r == nil || !slices.Equal(r.Workflows, []string{"workline", "workline warn"}) {
			t.Errorf("%sworkline-apply.yml: does not apply both runs, the judge's and the one apart: %+v", dir, r)
		}
		for name, job := range apply.Jobs {
			keeps := false
			for _, s := range job.Steps {
				keeps = keeps || strings.Contains(s.Run, "--verdict-only") || strings.Contains(s.Run, "workline follow")
			}
			if keeps && !strings.Contains(job.If, "github.event.workflow_run.name == 'workline'") {
				t.Errorf("%sworkline-apply.yml, job %s: keeps or follows a verdict after any run, the one apart's too: if %q", dir, name, job.If)
			}
		}
	}
	data, err := os.ReadFile("../../ci/gitlab/workline.gitlab-ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	var top map[string]any
	if err := yaml.Unmarshal(data, &top); err != nil {
		t.Fatal(err)
	}
	judge, _ := top["workline:judge"].(map[string]any)
	// The judge leaves the lenses that warn only where workline:warn runs,
	// the bot's token set; without it they read in the judge, said in its
	// log and its summary: never lost in silence.
	steps, _ := judge["script"].([]any)
	var route string
	var says []string
	for _, s := range steps {
		line := fmt.Sprint(s)
		switch {
		case strings.Contains(line, "workline route merge-request"):
			route = line
		case strings.Contains(line, "WORKLINE_GITLAB_TOKEN"):
			says = append(says, line)
		}
	}
	if !strings.Contains(route, "${WORKLINE_GITLAB_TOKEN:+--warn-apart}") || strings.Contains(strings.ReplaceAll(route, "${WORKLINE_GITLAB_TOKEN:+--warn-apart}", ""), "--warn-apart") {
		t.Errorf("workline:judge: leaves what only warns apart whether or not workline:warn can run: %s", route)
	}
	for _, token := range []string{"", "bot"} {
		dir := t.TempDir()
		cmd := exec.Command("sh", "-ec", strings.Join(says, "\n")+"\necho \"route[${WORKLINE_GITLAB_TOKEN:+--warn-apart}]\"")
		cmd.Dir, cmd.Env = dir, []string{"PATH=" + os.Getenv("PATH"), "WORKLINE_GITLAB_TOKEN=" + token}
		out, err := cmd.CombinedOutput()
		summary, _ := os.ReadFile(filepath.Join(dir, "workline-summary.md"))
		said := strings.Contains(string(out), "WORKLINE_GITLAB_TOKEN not set") && strings.Contains(string(summary), "WORKLINE_GITLAB_TOKEN not set")
		apart := strings.Contains(string(out), "route[--warn-apart]")
		if err != nil || said != (token == "") || apart != (token != "") {
			t.Errorf("workline:judge, WORKLINE_GITLAB_TOKEN %q: said in log and summary %v, left apart %v (%v)\n%s", token, said, apart, err, out)
		}
	}
	warn, _ := top["workline:warn"].(map[string]any)
	script := fmt.Sprint(warn["script"])
	if !strings.Contains(script, "--warn-only") || strings.Contains(script, "--verdict") || strings.Contains(script, "--code-quality") {
		t.Errorf("workline:warn: does not read what only warns alone, with no verdict nor report: %s", script)
	}
	// In a child pipeline the merge request's does not wait for (ADR-0051):
	// GitLab's "merge when the pipeline succeeds" waits for allow_failure
	// jobs too. Started after the gates, whatever their verdict, from this
	// same file, with no strategy.
	child, _ := top["workline:warn:child"].(map[string]any)
	trigger, _ := child["trigger"].(map[string]any)
	rules, _ := child["rules"].([]any)
	var rule map[string]any
	if len(rules) == 1 {
		rule, _ = rules[0].(map[string]any)
	}
	if trigger["include"] != "ci/workline.gitlab-ci.yml" || trigger["strategy"] != nil || fmt.Sprint(child["needs"]) != "[workline:gates]" ||
		rule["when"] != "always" || child["allow_failure"] != true || fmt.Sprint(child["variables"]) != "map[WORKLINE_PARENT_PIPELINE_ID:$CI_PIPELINE_ID]" {
		t.Errorf("workline:warn:child: does not start what only warns in a child pipeline the merge does not wait for, after the gates whatever their verdict: %v", child)
	}
	// The inspector triages the gates' findings (ADR-0049): read from their
	// job in the merge request's pipeline, never run here. GitLab's
	// needs:pipeline:job fetches only a job that passed: the bot's token
	// reads it, as given or as missing, said.
	if !strings.Contains(script, "--gates-from") || strings.Contains(script, "--gates-only") || fmt.Sprint(warn["needs"]) != "[]" {
		t.Errorf("workline:warn: does not read the gates' result from the merge request's pipeline (needs %v): %s", warn["needs"], script)
	}
	if _, err := exec.LookPath("jq"); err == nil {
		var fetch string
		for _, s := range warn["script"].([]any) {
			if line := fmt.Sprint(s); strings.Contains(line, "WORKLINE_PARENT_PIPELINE_ID") {
				fetch = line
			}
		}
		dir := t.TempDir()
		curl := "#!/bin/sh\nout=; for a; do [ \"$prev\" = -o ] && out=$a; prev=$a; url=$a; done\n" +
			"case $url in */pipelines/7/jobs*) echo '[{\"id\":41,\"name\":\"workline:judge\"},{\"id\":42,\"name\":\"workline:gates\"}]' ;;\n" +
			"*/jobs/42/artifacts/gates.json) echo '{\"steps\":[]}' > \"$out\" ;;\n*) exit 22 ;; esac\n"
		if err := os.WriteFile(filepath.Join(dir, "curl"), []byte(curl), 0o755); err != nil {
			t.Fatal(err)
		}
		for _, c := range []struct{ parent, gates string }{{"7", "gates.json"}, {"8", ""}, {"", ""}} {
			work := t.TempDir()
			cmd := exec.Command("sh", "-ec", fetch+"\necho \"gates[$gates]\"")
			cmd.Dir, cmd.Env = work, []string{"PATH=" + dir + string(os.PathListSeparator) + os.Getenv("PATH"), "WORKLINE_PARENT_PIPELINE_ID=" + c.parent,
				"WORKLINE_GITLAB_TOKEN=glpat", "CI_API_V4_URL=https://gitlab.example/api/v4", "CI_PROJECT_ID=1"}
			out, err := cmd.CombinedOutput()
			said := strings.Contains(string(out), "left no gates.json")
			if err != nil || !strings.Contains(string(out), "gates["+c.gates+"]") || said != (c.gates == "") {
				t.Errorf("workline:warn, the parent pipeline %q: %s (%v); want gates %q, a missing result said", c.parent, out, err, c.gates)
			}
		}
	} else if os.Getenv("GITHUB_ACTIONS") != "" {
		t.Fatal("jq is not on PATH: the GitLab template's check runs it, in its image")
	}
	for _, name := range []string{"workline:warn", "workline:warn:apply"} {
		if job, _ := top[name].(map[string]any); job["allow_failure"] != true {
			t.Errorf("job %s: allow_failure %v: what only warns would fail the pipeline the merge waits on", name, job["allow_failure"])
		}
	}
	if merge, _ := top["workline:merge"].(map[string]any); fmt.Sprint(merge["needs"]) != "[workline:judge]" {
		t.Errorf("workline:merge: needs %v: the head's verdict is the judge's alone", merge["needs"])
	}
}

// Every lens of the reviewer once when a merge request becomes ready
// (ADR-0020, #129): the templates pass `--input lenses=all` then, and only
// then. GitHub says so by the event; GitLab starts no pipeline on draft →
// ready, so its next pipeline reads the merge request's draft state and its
// notes, a ready note newer than the reviewer's comment.

func TestGitHubTemplatesAskEveryLensWhenReady(t *testing.T) {
	n := 0
	for _, f := range []string{"../../ci/github/workline.yml", "../../.github/workflows/workline.yml", "../../ci/github/workline-warn.yml", "../../.github/workflows/workline-warn.yml"} {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var wf struct {
			On struct {
				PullRequest struct {
					Types []string `yaml:"types"`
				} `yaml:"pull_request"`
			} `yaml:"on"`
			Jobs map[string]struct {
				Steps []struct {
					Env map[string]string `yaml:"env"`
					Run string            `yaml:"run"`
				} `yaml:"steps"`
			} `yaml:"jobs"`
		}
		if err := yaml.Unmarshal(data, &wf); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		for _, ty := range []string{"opened", "synchronize", "reopened", "ready_for_review"} {
			if !slices.Contains(wf.On.PullRequest.Types, ty) {
				t.Errorf("%s: pull_request types %v lack %s", filepath.Base(f), wf.On.PullRequest.Types, ty)
			}
		}
		for name, job := range wf.Jobs {
			for _, st := range job.Steps {
				if !strings.Contains(st.Run, "workline route merge-request") || strings.Contains(st.Run, "--gates-only") || strings.Contains(st.Run, "--input queue=true") {
					continue
				}
				n++
				if st.Env["ACTION"] != "${{ github.event.action }}" || st.Env["DRAFT"] != "${{ github.event.pull_request.draft }}" {
					t.Errorf("%s, job %s: ACTION and DRAFT are not the event's action and the pull request's draft: %v", filepath.Base(f), name, st.Env)
				}
				if !strings.Contains(st.Run, "${ready:+--input lenses=all}") {
					t.Errorf("%s, job %s: the line is not given lenses=all when ready", filepath.Base(f), name)
				}
				// The draft state on every run: a run on ready a push cancelled
				// leaves the next one every lens (#359).
				if !strings.Contains(st.Run, `--input draft="$DRAFT"`) {
					t.Errorf("%s, job %s: the line is not given the pull request's draft state", filepath.Base(f), name)
				}
				var decide string
				for _, l := range strings.Split(st.Run, "\n") {
					if strings.HasPrefix(strings.TrimSpace(l), "ready=;") {
						decide = l
					}
				}
				for _, c := range []struct {
					action, draft string
					ready         bool
				}{
					{"opened", "false", true}, {"opened", "true", false}, {"ready_for_review", "false", true},
					{"synchronize", "false", false}, {"reopened", "false", false},
				} {
					cmd := exec.Command("sh", "-ec", decide+"\necho \"[$ready]\"")
					cmd.Env = []string{"ACTION=" + c.action, "DRAFT=" + c.draft}
					out, err := cmd.CombinedOutput()
					if got := strings.TrimSpace(string(out)) == "[1]"; err != nil || got != c.ready {
						t.Errorf("%s, job %s: %s, draft %s: ready %v (%s, %v), want %v", filepath.Base(f), name, c.action, c.draft, got, out, err, c.ready)
					}
				}
			}
		}
	}
	if n < 4 {
		t.Fatalf("found %d steps running the merge request's line, want the template's and workline's own, the judge's and the run apart's", n)
	}
}

// A draft asks no agent (ADR-0050): the engine sets it aside, given the
// draft state, so the jobs that run the merge request's line install none
// on a draft, and still name the agent, for the engine to say the AI
// review waits for ready.
func TestTemplatesInstallNoAgentOnADraft(t *testing.T) {
	for _, f := range []string{"../../ci/github/workline.yml", "../../.github/workflows/workline.yml", "../../ci/github/workline-warn.yml", "../../.github/workflows/workline-warn.yml"} {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var wf struct {
			Jobs map[string]struct {
				Steps []struct {
					If  string            `yaml:"if"`
					Env map[string]string `yaml:"env"`
					Run string            `yaml:"run"`
				} `yaml:"steps"`
			} `yaml:"jobs"`
		}
		if err := yaml.Unmarshal(data, &wf); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		installs := 0
		for name, job := range wf.Jobs {
			for _, st := range job.Steps {
				if !strings.Contains(st.Run, "npm install -g @anthropic-ai/claude-code") {
					continue
				}
				installs++
				skipped := strings.Contains(st.If, "!github.event.pull_request.draft") ||
					st.Env["DRAFT"] == "${{ github.event.pull_request.draft }}" && strings.Contains(st.Run, `[ "$DRAFT" = true ] || npm install -g @anthropic-ai/claude-code`)
				if !skipped {
					t.Errorf("%s, job %s: the agent is installed on a draft, which asks none", filepath.Base(f), name)
				}
			}
		}
		if installs == 0 {
			t.Errorf("%s: no step installs the agent", filepath.Base(f))
		}
	}
	data, err := os.ReadFile("../../ci/gitlab/workline.gitlab-ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	var top map[string]any
	if err := yaml.Unmarshal(data, &top); err != nil {
		t.Fatal(err)
	}
	claude, _ := top[".workline:claude"].([]any)
	dir := t.TempDir()
	npm := "#!/bin/sh\necho installed > \"$NPM_SEEN\"\n"
	if err := os.WriteFile(filepath.Join(dir, "npm"), []byte(npm), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		draft, token string
		ai           string
		installed    bool
	}{
		{"true", "t", "claude", false}, {"false", "t", "claude", true}, {"", "t", "claude", true}, {"true", "", "none", false},
	} {
		seen := filepath.Join(dir, "seen-"+c.draft+c.token)
		var script strings.Builder
		for _, l := range claude {
			fmt.Fprintln(&script, l)
		}
		cmd := exec.Command("sh", "-ec", script.String()+"echo \"ai[$ai]\"")
		cmd.Env = []string{"PATH=" + dir + string(os.PathListSeparator) + os.Getenv("PATH"), "NPM_SEEN=" + seen, "draft=" + c.draft, "CLAUDE_CODE_OAUTH_TOKEN=" + c.token}
		out, err := cmd.CombinedOutput()
		_, statErr := os.Stat(seen)
		if err != nil || !strings.Contains(string(out), "ai["+c.ai+"]") || (statErr == nil) != c.installed {
			t.Errorf(".workline:claude, draft %q, token %q: %s (%v), installed %v; want ai %s, installed %v", c.draft, c.token, out, err, statErr == nil, c.ai, c.installed)
		}
	}
	// The draft state is read before the agent is installed.
	for _, job := range []string{"workline:judge", "workline:warn"} {
		j, _ := top[job].(map[string]any)
		script := fmt.Sprint(j["script"])
		at, install := strings.Index(script, "lenses=;"), strings.Index(script, "npm install")
		if at < 0 || install < 0 || at > install {
			t.Errorf("%s: the draft state (.workline:ready) is not read before the agent is installed: %s", job, script)
		}
	}
}

func TestGitLabTemplateAsksEveryLensWhenReady(t *testing.T) {
	if _, err := exec.LookPath("jq"); err != nil {
		if os.Getenv("GITHUB_ACTIONS") != "" {
			t.Fatal("jq is not on PATH: the GitLab template's check runs it, in its image")
		}
		t.Skip("jq is not on PATH: the GitLab template's check for ready is not tried here")
	}
	data, err := os.ReadFile("../../ci/gitlab/workline.gitlab-ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	var top map[string]any
	if err := yaml.Unmarshal(data, &top); err != nil {
		t.Fatal(err)
	}
	ready, _ := top[".workline:ready"].([]any)
	if len(ready) != 1 {
		t.Fatalf(".workline:ready: %v, want one script", top[".workline:ready"])
	}
	judge, _ := top["workline:judge"].(map[string]any)
	if script := fmt.Sprint(judge["script"]); !strings.Contains(script, "--input range=$CI_MERGE_REQUEST_DIFF_BASE_SHA..$CI_COMMIT_SHA $lenses ${draft:+--input draft=$draft}") ||
		!strings.Contains(script, "lenses=;") {
		t.Errorf("workline:judge does not run .workline:ready and pass $lenses and the draft state (#359): %s", script)
	}
	// The API as GitLab answers it: the merge request, then its notes, the
	// most recently updated first, a hundred a page (NOTES_<page>, the
	// pages not named NOTES_REST, else none); read with the token the
	// engine reads with. A query asking anything else is refused.
	dir := t.TempDir()
	curl := `#!/bin/sh
case "$*" in *"$AUTH"*) ;; *) exit 22 ;; esac
for a; do url=$a; done
case "$url" in
  *'/notes?order_by=updated_at&sort=desc&per_page=100&page='*)
    [ -z "$NOTES_UNREAD" ] || exit 22
    n=${url##*page=}; eval "f=\${NOTES_$n:-}"; [ -n "$f" ] || f=${NOTES_REST:-[]} ;;
  */notes*) exit 22 ;;
  *) f=$MR ;;
esac
[ -n "$f" ] || exit 22
printf '%s' "$f"
`
	if err := os.WriteFile(filepath.Join(dir, "curl"), []byte(curl), 0o755); err != nil {
		t.Fatal(err)
	}
	const (
		summary = `{"system": false, "body": "Reviewed.\n\n<!-- workline:sticky=reviewer/summary -->"}`
		marked  = `{"system": true, "body": "marked this merge request as **ready**"}`
		other   = `{"system": false, "body": "Looks good."}`
	)
	full := "[" + strings.Repeat(other+",", 99) + other + "]" // a full page, neither note on it
	for _, c := range []struct {
		name, draft, mr, token string
		pages                  []string // the notes, a page each
		rest                   string   // every page after them
		unread, all            bool
	}{
		{name: "opened ready: no comment yet", draft: "false", pages: []string{"[" + other + "]"}, all: true},
		{name: "marked ready since the last review", draft: "false", pages: []string{"[" + marked + "," + summary + "]"}, all: true},
		{name: "a push after the review on ready", draft: "false", pages: []string{"[" + other + "," + summary + "," + marked + "]"}},
		{name: "a draft", draft: "true", pages: []string{"[" + marked + "]"}},
		{name: "before GitLab 17.10: ready, by the API", mr: `{"draft": false}`, pages: []string{"[" + marked + "," + summary + "]"}, all: true},
		{name: "before GitLab 17.10: a draft, by the API", mr: `{"draft": true}`, pages: []string{"[" + marked + "]"}},
		{name: "read with GITLAB_TOKEN when the job has one", draft: "false", token: "read", pages: []string{"[" + marked + "]"}, all: true},
		{name: "the comment past a full page", draft: "false", pages: []string{full, "[" + summary + "," + marked + "]"}},
		{name: "the ready note past a full page", draft: "false", pages: []string{full, "[" + marked + "," + summary + "]"}, all: true},
		{name: "the comment past two full pages", draft: "false", pages: []string{full, full, "[" + summary + "]"}},
		{name: "every page full, neither note: 50 pages read, then one lens", draft: "false", rest: full},
		{name: "the notes unread", draft: "false", unread: true},
	} {
		env := []string{"PATH=" + dir + string(os.PathListSeparator) + os.Getenv("PATH"), "CI_API_V4_URL=https://gitlab.invalid/api/v4",
			"CI_PROJECT_ID=1", "CI_MERGE_REQUEST_IID=5", "CI_JOB_TOKEN=job", "MR=" + c.mr, "NOTES_REST=" + c.rest, "AUTH=JOB-TOKEN: job"}
		for i, p := range c.pages {
			env = append(env, fmt.Sprintf("NOTES_%d=%s", i+1, p))
		}
		if c.unread {
			env = append(env, "NOTES_UNREAD=1")
		}
		if c.token != "" {
			env = append(env, "GITLAB_TOKEN="+c.token, "AUTH=PRIVATE-TOKEN: "+c.token)
		}
		if c.draft != "" {
			env = append(env, "CI_MERGE_REQUEST_DRAFT="+c.draft)
		}
		cmd := exec.Command("sh", "-ec", fmt.Sprint(ready[0])+"\necho \"lenses[$lenses] page[$page] draft[$draft]\"")
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		if got := strings.Contains(string(out), "lenses[--input lenses=all]"); err != nil || got != c.all {
			t.Errorf("%s: every lens %v, want %v (%v)\n%s", c.name, got, c.all, err, out)
		}
		if (c.unread || c.rest != "") && !strings.Contains(string(out), "could not be read") {
			t.Errorf("%s: not said in the log:\n%s", c.name, out)
		}
		if c.rest != "" && !strings.Contains(string(out), "page[50]") {
			t.Errorf("%s: not stopped at page 50:\n%s", c.name, out)
		}
		// The draft state the judge passes on (#359), as the job or the API says it.
		want := c.draft
		if c.mr != "" {
			want = map[bool]string{true: "true", false: "false"}[strings.Contains(c.mr, "true")]
		}
		if !strings.Contains(string(out), "draft["+want+"]") {
			t.Errorf("%s: draft not %q for the judge:\n%s", c.name, want, out)
		}
	}
}

// Gardening tends the merge requests it opened before, every night, after
// applying what it proposed (ADR-0034, amended; #303): the job that
// applies runs whether or not anything was proposed, and its step runs
// `workline follow --gardening` after `workline apply`, its summary on
// the job's page.
func TestGardeningTendsItsMergeRequests(t *testing.T) {
	for _, f := range []string{"../../ci/github/workline-gardening.yml", "../../.github/workflows/workline-gardening.yml"} {
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
		found := false
		for name, job := range wf.Jobs {
			for _, s := range job.Steps {
				if !strings.Contains(s.Run, "workline apply") || strings.Contains(s.Run, "--release-dry-run") {
					continue
				}
				found = true
				if job.If != "always()" {
					t.Errorf("%s, job %s: if %q: it runs only when something was proposed, and tends nothing otherwise", f, name, job.If)
				}
				apply, follow := strings.Index(s.Run, "workline apply"), strings.Index(s.Run, "workline follow --gardening")
				if follow < apply || !strings.Contains(s.Run[follow:], "--summary workline-summary.md || status=$?") {
					t.Errorf("%s, job %s: does not run `workline follow --gardening` after applying, with its summary", f, name)
				}
			}
		}
		if !found {
			t.Errorf("%s: no job applies", f)
		}
	}
	data, err := os.ReadFile("../../ci/gitlab/workline.gitlab-ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	var top map[string]any
	if err := yaml.Unmarshal(data, &top); err != nil {
		t.Fatal(err)
	}
	job, _ := top["workline:garden:apply"].(map[string]any)
	script := fmt.Sprint(job["script"])
	apply, follow := strings.Index(script, "workline apply"), strings.Index(script, "workline follow --gardening")
	if apply < 0 || follow < apply || !strings.Contains(script[follow:], "|| status=$?") {
		t.Errorf("workline:garden:apply does not run `workline follow --gardening` after applying: %s", script)
	}
}

// The release pull request's docs are judged at night, with the gardening
// (ADR-0050, §7): its judging job runs the schedule line with a forge that
// lists the open merge requests and --open-merge-request, which is when
// the line takes up the release; on GitHub, the job that applies names
// the App, so that the release's fix is the App's pull request only when
// the App alone pushed to the release's branch, as for a pull request's
// own run (workline-apply.yml).
func TestGardeningJudgesTheReleaseAtNight(t *testing.T) {
	for _, f := range []string{"../../ci/github/workline-gardening.yml", "../../.github/workflows/workline-gardening.yml"} {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var wf struct {
			Jobs map[string]struct {
				Permissions map[string]string `yaml:"permissions"`
				Steps       []struct {
					Run string            `yaml:"run"`
					Env map[string]string `yaml:"env"`
				} `yaml:"steps"`
			} `yaml:"jobs"`
		}
		if err := yaml.Unmarshal(data, &wf); err != nil {
			t.Fatal(err)
		}
		judged, applied := false, false
		for name, job := range wf.Jobs {
			for _, s := range job.Steps {
				if strings.Contains(s.Run, "workline route schedule") {
					judged = true
					if !strings.Contains(s.Run, "--forge github") || !strings.Contains(s.Run, "--open-merge-request") || job.Permissions["pull-requests"] != "read" {
						t.Errorf("%s, job %s: the schedule line does not list the open pull requests (--forge github, --open-merge-request, pull-requests: read): no release is judged at night", f, name)
					}
				}
				if strings.Contains(s.Run, "workline apply") && !strings.Contains(s.Run, "--release-dry-run") {
					applied = true
					if !strings.Contains(s.Env["WORKLINE_APP"], "steps.app.outputs.app-slug") {
						t.Errorf("%s, job %s: applies with no WORKLINE_APP: who pushed to the release's branch is not checked before its fix is opened as the App's", f, name)
					}
				}
			}
		}
		if !judged || !applied {
			t.Errorf("%s: no job runs the schedule line, or none applies it", f)
		}
	}
	data, err := os.ReadFile("../../ci/gitlab/workline.gitlab-ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	var top map[string]any
	if err := yaml.Unmarshal(data, &top); err != nil {
		t.Fatal(err)
	}
	garden, _ := top[".workline:garden"].(map[string]any)
	job, _ := top["workline:garden:judge"].(map[string]any)
	script := fmt.Sprint(job["script"])
	if !strings.Contains(fmt.Sprint(garden["before_script"]), "export GITLAB_TOKEN=$WORKLINE_GITLAB_TOKEN") ||
		!strings.Contains(script, "--forge gitlab") || !strings.Contains(script, "--open-merge-request") {
		t.Errorf("workline:garden:judge does not list the open merge requests with the project's token: no release is judged at night: %s", script)
	}
}

// The merge policy (ADR-0042) acts after each judge, its verdict kept by a
// job no newer run cancels — in no concurrency group, where `follow` is in
// one —, and when a label is put or taken off — never in the workflow
// holding `judge`, whose run on a label would show the check skipped, which
// GitHub takes as passed for a required check. Both from the default
// branch's workflow, never the pull request's own (ADR-0044): the verdict is
// kept, and follow run, when the judge's workflow completes.
func TestMergePolicyRunsAfterTheJudgeAndOnLabels(t *testing.T) {
	type workflow struct {
		On struct {
			PullRequest struct {
				Types []string `yaml:"types"`
			} `yaml:"pull_request"`
			Target *struct {
				Types []string `yaml:"types"`
			} `yaml:"pull_request_target"`
			WorkflowRun *struct {
				Workflows []string `yaml:"workflows"`
				Types     []string `yaml:"types"`
			} `yaml:"workflow_run"`
		} `yaml:"on"`
		Jobs map[string]struct {
			Needs       any `yaml:"needs"`
			Concurrency any `yaml:"concurrency"`
			Steps       []struct {
				Run string `yaml:"run"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	read := func(f string) workflow {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var wf workflow
		if err := yaml.Unmarshal(data, &wf); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		return wf
	}
	for _, dir := range []string{"../../ci/github/", "../../.github/workflows/"} {
		judged := read(dir + "workline.yml")
		for _, ty := range []string{"labeled", "unlabeled", "converted_to_draft"} {
			if slices.Contains(judged.On.PullRequest.Types, ty) || judged.On.Target != nil {
				t.Errorf("%sworkline.yml: runs on %s, where judge would show skipped", dir, ty)
			}
		}
		var route string
		for name, job := range judged.Jobs {
			for _, s := range job.Steps {
				switch {
				case strings.Contains(s.Run, "--gates-only"), strings.Contains(s.Run, "--input queue=true"):
				case strings.Contains(s.Run, "workline route merge-request"):
					route = s.Run
				case strings.Contains(s.Run, "workline follow"):
					t.Errorf("%sworkline.yml, job %s: follows in the pull request's own run", dir, name)
				}
			}
		}
		if !strings.Contains(route, "--verdict workline-verdict.json") {
			t.Errorf("%sworkline.yml: judge does not write the head's verdict", dir)
		}
		after := read(dir + "workline-apply.yml")
		if r := after.On.WorkflowRun; r == nil || !slices.Equal(r.Workflows, []string{"workline", "workline warn"}) || !slices.Equal(r.Types, []string{"completed"}) {
			t.Errorf("%sworkline-apply.yml: does not run when workline.yml completes: %+v", dir, r)
		}
		var keep string
		for name, job := range after.Jobs {
			for _, s := range job.Steps {
				switch {
				case strings.Contains(s.Run, "--verdict-only"):
					keep = s.Run
					if job.Needs != nil || job.Concurrency != nil {
						t.Errorf("%sworkline-apply.yml, job %s: the verdict is not kept at once, in no concurrency group (needs %v, concurrency %v)", dir, name, job.Needs, job.Concurrency)
					}
					if !strings.Contains(s.Run, `[ "$head" = "$HEAD_SHA" ]`) {
						t.Errorf("%sworkline-apply.yml, job %s: the verdict is kept without checking it is the event's head's", dir, name)
					}
				case strings.Contains(s.Run, "--merge-group"):
					// A merge queue's group (ADR-0050): its yes set at once,
					// the queue waiting for it, no pull request followed.
					if job.Needs != nil || job.Concurrency != nil {
						t.Errorf("%sworkline-apply.yml, job %s: a queue's group waits for another job (needs %v, concurrency %v)", dir, name, job.Needs, job.Concurrency)
					}
				case strings.Contains(s.Run, "workline follow"):
					if fmt.Sprint(job.Needs) != "verdict" || job.Concurrency == nil {
						t.Errorf("%sworkline-apply.yml, job %s: follow does not run after the verdict is kept, one at a time (needs %v)", dir, name, job.Needs)
					}
				}
			}
		}
		if !strings.Contains(keep, "workline-verdict.json") {
			t.Errorf("%sworkline-apply.yml: no job keeps the head's verdict", dir)
		}
	}
	for _, f := range []string{"../../ci/github/workline-merge.yml", "../../.github/workflows/workline-merge.yml"} {
		wf := read(f)
		if wf.On.Target == nil || !slices.Contains(wf.On.Target.Types, "labeled") || !slices.Contains(wf.On.Target.Types, "unlabeled") {
			t.Errorf("%s: does not run on a label put and taken off", filepath.Base(f))
		}
		if _, ok := wf.Jobs["judge"]; ok {
			t.Errorf("%s: holds a judge job, which would show skipped on a label", filepath.Base(f))
		}
	}
}

// The documentalist's findings on a pull request reach the applying job,
// though the line proposed nothing, and that job may write the check
// holding them (#412): code scanning no longer gets them.
func TestGitHubFindingsReachTheApplyingJob(t *testing.T) {
	type workflow struct {
		Jobs map[string]struct {
			Permissions map[string]string `yaml:"permissions"`
			Steps       []struct {
				Uses string `yaml:"uses"`
				If   string `yaml:"if"`
				Run  string `yaml:"run"`
				With struct {
					Name string `yaml:"name"`
				} `yaml:"with"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	read := func(f string) workflow {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var wf workflow
		if err := yaml.Unmarshal(data, &wf); err != nil {
			t.Fatal(err)
		}
		return wf
	}
	for _, dir := range []string{"../../ci/github/", "../../.github/workflows/"} {
		judge := read(dir + "workline.yml").Jobs["judge"]
		found, uploaded := false, false
		for _, s := range judge.Steps {
			found = found || strings.Contains(s.Run, `echo "found=`)
			uploaded = uploaded || s.With.Name == "workline-runs" && strings.Contains(s.If, "steps.line.outputs.found == 'true'")
		}
		if !found || !uploaded {
			t.Errorf("%sworkline.yml: the documentalist's findings do not reach the applying job (found said: %v, uploaded on it: %v)", dir, found, uploaded)
		}
		if p := read(dir + "workline-apply.yml").Jobs["apply"].Permissions; p["checks"] != "write" {
			t.Errorf("%sworkline-apply.yml: apply may not write the check holding the findings (permissions %v)", dir, p)
		}
	}
}

// The judge's filter telling the documentalist found something reads the
// line's result as the engine writes it: a documentalist step's findings,
// handed off too, yes; another role's, or none, no.
func TestGitHubJudgeTellsTheDocumentalistFound(t *testing.T) {
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq is not installed: the judge's filter is not run here")
	}
	found := regexp.MustCompile(`echo "found=\$\(jq '([^']+)' line\.json\)"`)
	step := func(name string, findings ...verdict.Finding) line.Step {
		return line.Step{Name: name, Status: verdict.Pass, Result: &engine.Result{Status: verdict.Pass, Findings: findings}}
	}
	doc := verdict.Finding{Rule: "count-off", Where: "docs/a.md", Message: "`a.go` has 7 lines, not ~40"}
	for _, f := range []string{"../../ci/github/workline.yml", "../../.github/workflows/workline.yml"} {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		m := found.FindSubmatch(data)
		if m == nil {
			t.Fatalf("%s: no `found=` filter", filepath.Base(f))
		}
		for _, c := range []struct {
			name  string
			steps []line.Step
			want  string
		}{
			{"the documentalist's", []line.Step{step("committer"), step("documentalist", doc)}, "true"},
			{"handed off", []line.Step{step("documentalist (handoff)", doc)}, "true"},
			{"another role's", []line.Step{step("committer", verdict.Finding{Rule: "leak", Where: "config.ini:2"}), step("documentalist")}, "false"},
			{"a gate's step", []line.Step{{Name: "gate:secrets", Status: verdict.Pass, Gate: &verdict.Verdict{Status: verdict.Pass}}}, "false"},
		} {
			res, _ := json.Marshal(line.Result{Status: verdict.Pass, Steps: c.steps})
			cmd := exec.Command("jq", string(m[1]))
			cmd.Stdin = bytes.NewReader(res)
			out, err := cmd.Output()
			if got := strings.TrimSpace(string(out)); err != nil || got != c.want {
				t.Errorf("%s, %s: %q (%v), want %s", filepath.Base(f), c.name, got, err, c.want)
			}
		}
	}
}

// The gates run in a job of their own, their tools the change's code: on
// GitHub, one holding a read token alone and no secret, the agent's
// neither, its checkout keeping no token; the judge reads their verdicts
// and never runs a gate (ADR-0049). On GitLab, which gives every job the
// project's variables, the gates' job unsets the agent's token and the
// project bot's first, and its Code Quality report is what GitLab shows.
func TestGatesRunWithNoTokenNorAgent(t *testing.T) {
	type step struct {
		Uses string         `yaml:"uses"`
		With map[string]any `yaml:"with"`
		Env  map[string]any `yaml:"env"`
		If   string         `yaml:"if"`
		Run  string         `yaml:"run"`
	}
	type job struct {
		Needs       any               `yaml:"needs"`
		If          string            `yaml:"if"`
		Permissions map[string]string `yaml:"permissions"`
		Env         map[string]any    `yaml:"env"`
		Steps       []step            `yaml:"steps"`
	}
	for _, f := range []string{"../../ci/github/workline.yml", "../../.github/workflows/workline.yml"} {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var wf struct {
			Env  map[string]any `yaml:"env"`
			Jobs map[string]job `yaml:"jobs"`
		}
		if err := yaml.Unmarshal(data, &wf); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		for k, v := range wf.Env {
			if strings.Contains(fmt.Sprint(v), "secrets.") {
				t.Errorf("%s: the workflow's env %s gives every job a secret", f, k)
			}
		}
		running := 0
		for name, j := range wf.Jobs {
			text, _ := yaml.Marshal(j)
			gates := strings.Contains(string(text), "--gates-only")
			if !gates {
				continue
			}
			running++
			if len(j.Permissions) != 1 || j.Permissions["contents"] != "read" {
				t.Errorf("%s, job %s: runs the gates with permissions %v, want contents: read alone", f, name, j.Permissions)
			}
			if strings.Contains(string(text), "secrets.") {
				t.Errorf("%s, job %s: runs the gates with a secret in reach", f, name)
			}
			for _, s := range j.Steps {
				if strings.HasPrefix(s.Uses, "actions/checkout@") && fmt.Sprint(s.With["persist-credentials"]) != "false" {
					t.Errorf("%s, job %s: its checkout keeps the token in .git/config for the gates' tools to read", f, name)
				}
			}
		}
		if running != 1 {
			t.Errorf("%s: %d jobs run the gates alone, want one", f, running)
		}
		// Main's run after a merge is the gates' job's alone (ADR-0050).
		judge := wf.Jobs["judge"]
		if fmt.Sprint(judge.Needs) != "gates" || judge.If != "!cancelled() && github.event_name != 'push'" {
			t.Errorf("%s: judge does not run after the gates whatever their verdict (needs %v, if %q)", f, judge.Needs, judge.If)
		}
		read := false
		for _, s := range judge.Steps {
			if strings.Contains(s.Run, "workline route merge-request") {
				read = strings.Contains(s.Run, `--gates-from "$RUNNER_TEMP/gates/gates.json"`)
			}
		}
		if !read {
			t.Errorf("%s: judge's line runs the gates itself, beside the agent's token: no --gates-from", f)
		}
	}
	data, err := os.ReadFile("../../ci/gitlab/workline.gitlab-ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	var top map[string]any
	if err := yaml.Unmarshal(data, &top); err != nil {
		t.Fatal(err)
	}
	gates, _ := top["workline:gates"].(map[string]any)
	script, _ := gates["script"].([]any)
	if len(script) == 0 || fmt.Sprint(script[0]) != "unset CLAUDE_CODE_OAUTH_TOKEN WORKLINE_GITLAB_TOKEN GITLAB_TOKEN" {
		t.Errorf("workline:gates: its script does not unset the agent's and the bot's tokens first: %v", script)
	}
	if !strings.Contains(fmt.Sprint(script), "--gates-only") || !strings.Contains(fmt.Sprint(script), "--code-quality gl-code-quality-gates.json") {
		t.Errorf("workline:gates: does not run the gates alone into its Code Quality report: %v", script)
	}
	if reports, _ := gates["artifacts"].(map[string]any)["reports"].(map[string]any); fmt.Sprint(reports["codequality"]) != "gl-code-quality-gates.json" {
		t.Errorf("workline:gates: GitLab is not given its Code Quality report: %v", reports)
	}
	judge, _ := top["workline:judge"].(map[string]any)
	if !strings.Contains(fmt.Sprint(judge["script"]), "--gates-from gates.json") || fmt.Sprint(judge["needs"]) != "[workline:gates]" {
		t.Errorf("workline:judge: does not read the gates' verdicts from their job (needs %v)", judge["needs"])
	}
	for _, r := range judge["rules"].([]any) {
		if m, _ := r.(map[string]any); fmt.Sprint(m["when"]) != "always" {
			t.Errorf("workline:judge: rule %v is not `when: always`: gates that did not pass would skip the line", m)
		}
	}
}
