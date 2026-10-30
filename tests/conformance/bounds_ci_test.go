package conformance

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// ci-github-bounds-check-from-base (ADR-0058, step 2): the developer's
// bounds are a check on its pull request, `workline bounds`, which
// branch protection can require. GitHub runs it from the default
// branch's definition (pull_request_target), so a pull request cannot
// edit its own check; its code never runs there: the base checked out,
// the head fetched as data, no agent, a read token. The verdict is kept
// on the pull request by a job apart, the App's key in its environment,
// at the head the check read. In the template and in workline's own.
func TestCIGitHubBoundsCheckFromBase(t *testing.T) {
	for _, f := range []string{"../../ci/github/workline-bounds.yml", "../../.github/workflows/workline-bounds.yml"} {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		type step struct {
			Uses string            `yaml:"uses"`
			With map[string]any    `yaml:"with"`
			Env  map[string]string `yaml:"env"`
			Run  string            `yaml:"run"`
		}
		var wf struct {
			On   map[string]any `yaml:"on"`
			Jobs map[string]struct {
				Name        string            `yaml:"name"`
				If          string            `yaml:"if"`
				Needs       any               `yaml:"needs"`
				Environment any               `yaml:"environment"`
				Permissions map[string]string `yaml:"permissions"`
				Steps       []step            `yaml:"steps"`
			} `yaml:"jobs"`
		}
		if err := yaml.Unmarshal(data, &wf); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		if _, ok := wf.On["pull_request_target"]; !ok || len(wf.On) != 1 {
			t.Errorf("%s: not run from the default branch's definition alone (pull_request_target): %v", f, wf.On)
		}
		b := wf.Jobs["bounds"]
		if b.Name != "workline bounds" || !strings.Contains(b.If, "'workline/developer/'") {
			t.Errorf("%s, bounds: not the check `workline bounds`, or not on the developer's branches: %q %q", f, b.Name, b.If)
		}
		for k, v := range b.Permissions {
			if v != "read" {
				t.Errorf("%s, bounds: holds %s: %s, a write", f, k, v)
			}
		}
		checked := false
		for _, s := range b.Steps {
			if strings.HasPrefix(s.Uses, "actions/checkout@") {
				checked = true
				if ref := fmt.Sprint(s.With["ref"]); s.With["ref"] != nil {
					t.Errorf("%s, bounds: checks out %s, not the base", f, ref)
				}
				if fmt.Sprint(s.With["persist-credentials"]) != "false" || fmt.Sprint(s.With["fetch-depth"]) != "0" {
					t.Errorf("%s, bounds: its checkout keeps the token, or not every branch: %v", f, s.With)
				}
			}
			for k := range s.Env {
				if strings.Contains(k, "CLAUDE") || strings.Contains(k, "ANTHROPIC") || strings.Contains(s.Run, "WORKLINE_APP_PRIVATE_KEY") {
					t.Errorf("%s, bounds: an agent's token or the App's key in reach (%s)", f, k)
				}
			}
			if strings.Contains(s.Run, "git checkout") || strings.Contains(s.Run, "git switch") || strings.Contains(s.Run, "${{ github.event.pull_request.head.ref }}") {
				t.Errorf("%s, bounds: checks the head out, or writes its branch's name into the script: %s", f, s.Run)
			}
			if strings.Contains(s.Run, "workline develop --bounds") {
				for _, want := range []string{`--merge-request "$PR"`, "--ai none", "--no-apply", `--input range="$GITHUB_SHA..$HEAD_SHA"`} {
					if !strings.Contains(s.Run, want) {
						t.Errorf("%s, bounds: the check lacks %s: %s", f, want, s.Run)
					}
				}
				if !strings.Contains(s.Run, `fetch -q origin "+$HEAD_SHA:refs/remotes/origin/$HEAD_REF"`) {
					t.Errorf("%s, bounds: the head is not fetched as data before the check", f)
				}
			}
		}
		if !checked {
			t.Errorf("%s, bounds: no checkout of the base", f)
		}
		a := wf.Jobs["bounds-apply"]
		if fmt.Sprint(a.Needs) != "bounds" || environmentOf(a.Environment) != "workline-app" {
			t.Errorf("%s, bounds-apply: not after the check, or the App's key outside workline-app: %v %v", f, a.Needs, a.Environment)
		}
		applied := false
		for _, s := range a.Steps {
			if strings.HasPrefix(s.Uses, "actions/checkout@") && s.With["ref"] != nil {
				t.Errorf("%s, bounds-apply: checks out %v beside the App's key", f, s.With["ref"])
			}
			applied = applied || strings.Contains(s.Run, `workline apply "$RUN" --from-merge-request "$PR" --head "$HEAD_SHA"`)
		}
		if !applied {
			t.Errorf("%s, bounds-apply: does not keep the verdict on the pull request, at the head checked", f)
		}
	}
}

// ci-gitlab-bounds-guard-by-tier (ADR-0058, step 2): on GitLab, the job
// workline:bounds runs on a merge request from a developer's branch, no
// agent, the bot's token or a failure, never a pass. GitLab runs a merge
// request's pipeline from its own branch, so the job is a file of its
// own, with no anchor nor `extends`: a pipeline execution policy can
// include it on Ultimate, CODEOWNERS own it on Premium. The doctor reads
// the tier (cases setup/doctor-bounds-guard-*).
func TestCIGitLabBoundsGuardByTier(t *testing.T) {
	data, err := os.ReadFile("../../ci/gitlab/workline-bounds.gitlab-ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "<<: *") || strings.Contains(string(data), ": *") {
		t.Error("workline-bounds.gitlab-ci.yml: merges an anchor, which a policy's file does not have")
	}
	var top map[string]any
	if err := yaml.Unmarshal(data, &top); err != nil {
		t.Fatal(err)
	}
	j, ok := top["workline:bounds"].(map[string]any)
	if !ok || len(top) != 1 {
		t.Fatalf("workline-bounds.gitlab-ci.yml: not the one job workline:bounds: %v", top)
	}
	if j["extends"] != nil || j["image"] == nil {
		t.Errorf("workline:bounds: extends a job of workline.gitlab-ci.yml, or names no image: %v %v", j["extends"], j["image"])
	}
	rules := fmt.Sprint(j["rules"])
	if !strings.Contains(rules, `$CI_PIPELINE_SOURCE == "merge_request_event"`) || !strings.Contains(rules, `$CI_MERGE_REQUEST_SOURCE_BRANCH_NAME =~ /^workline\/developer\/[0-9]+$/`) {
		t.Errorf("workline:bounds: not on a developer's merge request: %s", rules)
	}
	script := fmt.Sprint(j["script"])
	for _, want := range []string{`workline develop --bounds --merge-request "$CI_MERGE_REQUEST_IID" --ai none --forge gitlab --input range="$base..$head"`, "exit 1; }", "CI_MERGE_REQUEST_SOURCE_BRANCH_SHA"} {
		if !strings.Contains(script, want) {
			t.Errorf("workline:bounds: its script lacks %s: %s", want, script)
		}
	}
	if strings.Contains(script, "exit 0") || strings.Contains(script, "CLAUDE") {
		t.Errorf("workline:bounds: passes without checking, or holds an agent: %s", script)
	}
}
