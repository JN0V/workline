package conformance

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
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

// Every lens of the reviewer once when a merge request becomes ready
// (ADR-0020, #129): the templates pass `--input lenses=all` then, and only
// then. GitHub says so by the event; GitLab starts no pipeline on draft →
// ready, so its next pipeline reads the merge request's draft state and its
// notes, a ready note newer than the reviewer's comment.

func TestGitHubTemplatesAskEveryLensWhenReady(t *testing.T) {
	n := 0
	for _, f := range []string{"../../ci/github/workline.yml", "../../.github/workflows/workline.yml"} {
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
				if !strings.Contains(st.Run, "workline route merge-request") {
					continue
				}
				n++
				if st.Env["ACTION"] != "${{ github.event.action }}" || st.Env["DRAFT"] != "${{ github.event.pull_request.draft }}" {
					t.Errorf("%s, job %s: ACTION and DRAFT are not the event's action and the pull request's draft: %v", filepath.Base(f), name, st.Env)
				}
				if !strings.Contains(st.Run, "${ready:+--input lenses=all}") {
					t.Errorf("%s, job %s: the line is not given lenses=all when ready", filepath.Base(f), name)
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
	if n < 2 {
		t.Fatalf("found %d steps running the merge request's line, want the template's and workline's own", n)
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
	if script := fmt.Sprint(judge["script"]); !strings.Contains(script, "--input range=$CI_MERGE_REQUEST_DIFF_BASE_SHA..$CI_COMMIT_SHA $lenses") ||
		!strings.Contains(script, "lenses=;") {
		t.Errorf("workline:judge does not run .workline:ready and pass $lenses: %s", script)
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
		cmd := exec.Command("sh", "-ec", fmt.Sprint(ready[0])+"\necho \"lenses[$lenses] page[$page]\"")
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
				if !strings.Contains(s.Run, "workline apply") {
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
