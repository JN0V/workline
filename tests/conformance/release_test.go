package conformance

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// A release's dry run each night (ADR-0050): what would hold the release
// if it were cut now, run in a job holding no write token and asking no
// agent, then posted by a job holding the token, one writer of the
// default branch's issue at a time; and at a tag on GitLab, the release's
// line with no agent nor token, red when it holds the release.

// ghWorkflow is what these tests read of a GitHub workflow.
type ghWorkflow struct {
	On struct {
		Schedule []struct {
			Cron string `yaml:"cron"`
		} `yaml:"schedule"`
	} `yaml:"on"`
	Jobs map[string]struct {
		Needs       any               `yaml:"needs"`
		Permissions map[string]string `yaml:"permissions"`
		Concurrency map[string]any    `yaml:"concurrency"`
		Env         map[string]string `yaml:"env"`
		Steps       []struct {
			Uses string            `yaml:"uses"`
			Run  string            `yaml:"run"`
			Env  map[string]string `yaml:"env"`
		} `yaml:"steps"`
	} `yaml:"jobs"`
}

func readWorkflow(t *testing.T, f string) ghWorkflow {
	t.Helper()
	data, err := os.ReadFile(f)
	if err != nil {
		t.Fatal(err)
	}
	var wf ghWorkflow
	if err := yaml.Unmarshal(data, &wf); err != nil {
		t.Fatal(err)
	}
	return wf
}

// nightsOf are the days of the week a workflow's schedules run on, 0 to 6.
func nightsOf(t *testing.T, wf ghWorkflow) map[int]bool {
	t.Helper()
	nights := map[int]bool{}
	for _, s := range wf.On.Schedule {
		f := strings.Fields(s.Cron)
		if len(f) != 5 {
			t.Fatalf("cron %q: not five fields", s.Cron)
		}
		if f[4] == "*" {
			for d := range 7 {
				nights[d] = true
			}
			continue
		}
		for _, part := range strings.Split(f[4], ",") {
			a, b, ok := strings.Cut(part, "-")
			x, _ := strconv.Atoi(a)
			y := x
			if ok {
				y, _ = strconv.Atoi(b)
			}
			for d := x; d <= y; d++ {
				nights[d%7] = true
			}
		}
	}
	return nights
}

func TestReleaseDryRunEachNight(t *testing.T) {
	// The templates: the weekly gardening and the release's docs the other
	// nights, every night between them; workline's own, its nightly gardening.
	sets := [][]string{
		{"../../ci/github/workline-gardening.yml", "../../ci/github/workline-release-docs.yml"},
		{"../../.github/workflows/workline-gardening.yml"},
	}
	for _, files := range sets {
		nights := map[int]bool{}
		for _, f := range files {
			wf := readWorkflow(t, f)
			ran, posted := "", ""
			for name, job := range wf.Jobs {
				for _, s := range job.Steps {
					if strings.Contains(s.Run, "workline route release --dry-run") {
						ran = name
						if job.Permissions["contents"] != "read" || job.Permissions["pull-requests"] != "read" || len(job.Permissions) != 2 {
							t.Errorf("%s, job %s: the dry run holds more than contents and pull-requests read: %v", f, name, job.Permissions)
						}
						if !strings.Contains(s.Run, "--forge github") {
							t.Errorf("%s, job %s: the dry run lists no pull request (--forge github): the release's own is never judged", f, name)
						}
						for k, v := range s.Env {
							if strings.Contains(v, "secrets.") || strings.Contains(k, "CLAUDE") {
								t.Errorf("%s, job %s: the dry run is given %s: it asks no agent and writes nothing", f, name, k)
							}
						}
					}
					if strings.Contains(s.Run, "workline apply --release-dry-run") {
						posted = name
						if job.Permissions["issues"] != "write" || job.Permissions["pull-requests"] != "write" {
							t.Errorf("%s, job %s: posts with no issues and pull-requests write: %v", f, name, job.Permissions)
						}
						if fmt.Sprint(job.Concurrency["group"]) != "workline-main-issue" {
							t.Errorf("%s, job %s: not in main's issue concurrency group: two writers of one issue", f, name)
						}
						for _, s := range job.Steps {
							if strings.Contains(s.Run, "npm install") || strings.Contains(s.Run, "workline route") {
								t.Errorf("%s, job %s: the job holding the token runs %q", f, name, s.Run)
							}
						}
					}
				}
			}
			if ran == "" || posted == "" {
				t.Errorf("%s: no job runs the release's dry run, or none posts it", f)
				continue
			}
			if !strings.Contains(fmt.Sprint(wf.Jobs[posted].Needs), ran) {
				t.Errorf("%s: %s posts before %s ran", f, posted, ran)
			}
			for d := range nightsOf(t, wf) {
				nights[d] = true
			}
		}
		if len(nights) != 7 {
			t.Errorf("%v: the dry run runs %d nights a week, not every night", files, len(nights))
		}
	}

	top := readGitLab(t)
	run, _ := top["workline:release:dry-run"].(map[string]any)
	post, _ := top["workline:release:dry-run:post"].(map[string]any)
	if run == nil || post == nil {
		t.Fatal("GitLab: no workline:release:dry-run, or no workline:release:dry-run:post")
	}
	rules := fmt.Sprint(run["rules"])
	if !strings.Contains(rules, `$CI_PIPELINE_SOURCE == "schedule"`) || !strings.Contains(rules, `$WORKLINE_TASK != "sample"`) || strings.Contains(rules, "release-docs") {
		t.Errorf("GitLab: the dry run does not run on every night's schedule, the gardening's and the release docs': %s", rules)
	}
	script := fmt.Sprint(run["script"])
	if !strings.Contains(script, "workline route release --dry-run") || !strings.Contains(script, "unset CLAUDE_CODE_OAUTH_TOKEN") || strings.Contains(script, "npm install") {
		t.Errorf("GitLab: the dry run is not run with no agent: %s", script)
	}
	if fmt.Sprint(post["resource_group"]) != "workline-main-issue" || !strings.Contains(fmt.Sprint(post["needs"]), "workline:release:dry-run") ||
		!strings.Contains(fmt.Sprint(post["script"]), "workline apply --release-dry-run") {
		t.Errorf("GitLab: the dry run is not posted after it ran, one writer of main's issue at a time: %v", post)
	}
	if issue, _ := top["workline:main:issue"].(map[string]any); fmt.Sprint(issue["resource_group"]) != "workline-main-issue" {
		t.Errorf("GitLab: workline:main:issue is not in the resource group the dry run's post is in")
	}
}

func readGitLab(t *testing.T) map[string]any {
	t.Helper()
	data, err := os.ReadFile("../../ci/gitlab/workline.gitlab-ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	var top map[string]any
	if err := yaml.Unmarshal(data, &top); err != nil {
		t.Fatal(err)
	}
	return top
}

// On GitLab, a tag pipeline runs workline:release alone: the release's
// line, the whole history and every tag fetched, no agent, the tokens
// unset before it, red when it holds the release, never allowed to fail.
func TestGitLabReleaseJobAtATag(t *testing.T) {
	top := readGitLab(t)
	job, _ := top["workline:release"].(map[string]any)
	if job == nil {
		t.Fatal("GitLab: no workline:release")
	}
	if !strings.Contains(fmt.Sprint(job["rules"]), "$CI_COMMIT_TAG") {
		t.Errorf("workline:release does not run on a tag pipeline: %v", job["rules"])
	}
	if v, _ := job["variables"].(map[string]any); fmt.Sprint(v["GIT_DEPTH"]) != "0" {
		t.Errorf("workline:release: not the whole history: the release before this one may be missing")
	}
	if _, allowed := job["allow_failure"]; allowed {
		t.Errorf("workline:release may fail: a held release would publish")
	}
	script, _ := job["script"].([]any)
	unset, route := -1, -1
	for i, l := range script {
		s := fmt.Sprint(l)
		switch {
		case strings.Contains(s, "unset") && strings.Contains(s, "WORKLINE_GITLAB_TOKEN") && strings.Contains(s, "CLAUDE_CODE_OAUTH_TOKEN"):
			unset = i
		case strings.Contains(s, "workline route release"):
			route = i
			if !strings.Contains(s, "--ai none") || strings.Contains(s, "--dry-run") || strings.Contains(s, "--no-apply") {
				t.Errorf("workline:release: %q is not the release's own line, no agent", s)
			}
		case strings.Contains(s, "git fetch") && strings.Contains(s, "--tags"):
		}
	}
	if route < 0 || unset < 0 || unset > route {
		t.Errorf("workline:release: the tokens are not unset before the release's line: %v", script)
	}
	if !strings.Contains(fmt.Sprint(script), "exit $status") {
		t.Errorf("workline:release does not end with the line's status: a held release goes green")
	}
	// No other job of workline's runs at a tag: every other one asks for a
	// merge request, a schedule, or a branch, which a tag pipeline has not.
	for name, v := range top {
		j, ok := v.(map[string]any)
		if !ok || name == "workline:release" || strings.HasPrefix(name, ".") || !strings.HasPrefix(name, "workline:") {
			continue
		}
		rules := fmt.Sprint(j["rules"])
		if j["rules"] == nil {
			if ext, _ := j["extends"].(string); ext != "" {
				parent, _ := top[ext].(map[string]any)
				rules = fmt.Sprint(parent["rules"])
			}
		}
		if rules == "<nil>" || strings.Contains(rules, "$CI_COMMIT_TAG") ||
			!(strings.Contains(rules, "merge_request_event") || strings.Contains(rules, `"schedule"`) || strings.Contains(rules, "$CI_COMMIT_BRANCH ==") || strings.Contains(rules, `"web"`) || strings.Contains(rules, `"api"`) || strings.Contains(rules, `"trigger"`)) {
			t.Errorf("GitLab: %s may run on a tag pipeline: %s", name, rules)
		}
	}
}
