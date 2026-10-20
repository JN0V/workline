package conformance

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// Main after a merge (ADR-0050): on each push to the default branch, the
// line of `main` runs once, in the job holding no secret: the gates every
// finding kept by the commit where main alone writes it, the whole repository's
// checks after them, no agent. A pull request based on that commit reads
// the kept findings as its merge base's. One job that writes keeps what
// main's run found in one issue, and only main's run: never a pull
// request's, nor another branch's push.

// A push to main as GitHub gives it to a workflow.
var mainPush = map[string]any{
	"github.event_name":   "push",
	"github.repository":   "owner/repo",
	"github.sha":          "a0a0",
	"github.event.before": "9f9f",
}

func TestGitHubMainAfterAMergeRunsOnce(t *testing.T) {
	type step struct {
		Name string         `yaml:"name"`
		Uses string         `yaml:"uses"`
		With map[string]any `yaml:"with"`
		Env  map[string]any `yaml:"env"`
		If   string         `yaml:"if"`
		Run  string         `yaml:"run"`
	}
	type job struct {
		If          string            `yaml:"if"`
		Permissions map[string]string `yaml:"permissions"`
		Steps       []step            `yaml:"steps"`
	}
	type workflow struct {
		On   map[string]any `yaml:"on"`
		Jobs map[string]job `yaml:"jobs"`
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
		wf := read(dir + "workline.yml")
		if fmt.Sprint(wf.On["push"]) != "map[branches:[main]]" {
			t.Errorf("%sworkline.yml: no run on a push to main (%v)", dir, wf.On["push"])
		}
		ctx := map[string]any{"steps.any.outputs.any": "true", "steps.base.outputs.sha": "9f9f", "steps.main.outputs.kept": "true"}
		for k, v := range mainPush {
			ctx[k] = v
		}
		if j := wf.Jobs["judge"]; truthy(eval(t, j.If, ctx)) {
			t.Errorf("%sworkline.yml: judge runs on main, beside the agent's token: %q", dir, j.If)
		}
		gates := wf.Jobs["gates"]
		if len(gates.Permissions) != 1 || gates.Permissions["contents"] != "read" {
			t.Errorf("%sworkline.yml: main's run holds %v, want contents: read alone", dir, gates.Permissions)
		}
		var restored, saved map[string]any
		mains := 0
		for _, s := range gates.Steps {
			if s.If != "" && !truthy(eval(t, s.If, ctx)) {
				continue
			}
			if env, _ := yaml.Marshal(s.Env); strings.Contains(string(env), "secrets.") {
				t.Errorf("%sworkline.yml: main's step %q holds a secret", dir, s.Name)
			}
			run := expand(t, s.Run, ctx)
			switch {
			case strings.HasPrefix(s.Uses, "actions/checkout@"):
				if ref := expand(t, fmt.Sprint(s.With["ref"]), ctx); ref != "a0a0" {
					t.Errorf("%sworkline.yml: main's run checks out %q, want the pushed commit", dir, ref)
				}
			case strings.HasPrefix(s.Uses, "actions/cache/restore@"):
				restored = s.With
			case strings.HasPrefix(s.Uses, "actions/cache/save@"):
				saved = s.With
			case strings.Contains(run, "--gates-only"):
				t.Errorf("%sworkline.yml: the merge request's gates run on main: %s", dir, s.Name)
			case strings.Contains(run, "workline route main"):
				mains++
				for _, want := range []string{"--ai none", "--forge none", `--gates-kept "$RUNNER_TEMP/workline-kept"`, "--input range=9f9f..a0a0"} {
					if !strings.Contains(run, want) {
						t.Errorf("%sworkline.yml: main's run without %s:\n%s", dir, want, run)
					}
				}
			}
		}
		if mains != 1 {
			t.Errorf("%sworkline.yml: %d steps run main's line on a push, want one", dir, mains)
		}
		// Read at main's tip before the merge, kept by the commit pushed,
		// in the same folder; never by a prefix, an older commit's.
		if restored == nil || saved == nil || restored["path"] != saved["path"] || fmt.Sprint(restored["path"]) != "${{ runner.temp }}/workline-kept" {
			t.Errorf("%sworkline.yml: main's findings not read and kept in one folder: %v, %v", dir, restored, saved)
		} else {
			if got := expand(t, fmt.Sprint(restored["key"]), ctx); got != "workline-gates-9f9f" {
				t.Errorf("%sworkline.yml: main's run reads the findings kept at %q, want main's tip before the merge", dir, got)
			}
			if got := expand(t, fmt.Sprint(saved["key"]), ctx); got != "workline-gates-a0a0" {
				t.Errorf("%sworkline.yml: main's run keeps its findings as %q, want the commit pushed", dir, got)
			}
			if _, ok := restored["restore-keys"]; ok {
				t.Errorf("%sworkline.yml: the findings are read by a prefix: another commit's could come back", dir)
			}
		}
		// A pull request reads them at its merge base.
		pr := map[string]any{"steps.any.outputs.any": "true", "steps.base.outputs.sha": "b0b0"}
		for k, v := range pullRequestPush {
			pr[k] = v
		}
		reads := false
		for _, s := range gates.Steps {
			if s.If != "" && !truthy(eval(t, s.If, pr)) {
				continue
			}
			if strings.HasPrefix(s.Uses, "actions/cache/save@") {
				t.Errorf("%sworkline.yml: a pull request's run keeps findings as main's", dir)
			}
			if run := expand(t, s.Run, pr); strings.Contains(run, "--gates-only") {
				reads = strings.Contains(run, `--gates-kept "$RUNNER_TEMP/workline-kept"`)
			}
			if s.Name == "The base's findings, kept by main" && (!strings.Contains(s.Run, `git merge-base "$BASE" "$HEAD"`) || fmt.Sprint(s.Env["HEAD"]) != "${{ github.event.pull_request.head.sha }}") {
				t.Errorf("%sworkline.yml: a pull request does not read the findings kept at its merge base: %s", dir, s.Run)
			}
		}
		if !reads {
			t.Errorf("%sworkline.yml: a pull request's gates do not read the findings main kept", dir)
		}

		// After main's run, one job keeps what it found in an issue; after a
		// push to another branch, none.
		apply := read(dir + "workline-apply.yml")
		for _, c := range []struct {
			branch string
			want   []string
		}{{"main", []string{"main"}}, {"feature", nil}} {
			run := map[string]any{"github.event_name": "workflow_run", "github.repository": "owner/repo",
				"github.event.workflow_run.name": "workline", "github.event.workflow_run.event": "push",
				"github.event.workflow_run.head_branch": c.branch, "github.event.repository.default_branch": "main",
				"github.event.workflow_run.conclusion": "failure", "github.event.workflow_run.head_repository.full_name": "owner/repo"}
			var ran []string
			for name, j := range apply.Jobs {
				if j.If != "" && truthy(eval(t, j.If, run)) && name != "follow" {
					ran = append(ran, name)
				}
			}
			slices.Sort(ran)
			if !slices.Equal(ran, c.want) {
				t.Errorf("%sworkline-apply.yml: after a push to %s, runs %v, want %v", dir, c.branch, ran, c.want)
			}
		}
		m := apply.Jobs["main"]
		if m.Permissions["issues"] != "write" || m.Permissions["contents"] != "read" || m.Permissions["pull-requests"] != "" {
			t.Errorf("%sworkline-apply.yml, job main: permissions %v, want issues: write and contents: read", dir, m.Permissions)
		}
		keeps := 0
		for _, s := range m.Steps {
			if strings.Contains(s.Run, "workline apply --main") {
				keeps++
				if !strings.Contains(s.Run, `-n workline-main -D "$RUNNER_TEMP/main"`) || !strings.Contains(s.Run, `--main "$RUNNER_TEMP/main/main.json" --forge github`) {
					t.Errorf("%sworkline-apply.yml, job main: does not keep main's own result, read outside the checkout:\n%s", dir, s.Run)
				}
			}
		}
		if keeps != 1 {
			t.Errorf("%sworkline-apply.yml, job main: %d steps keep main's findings, want one", dir, keeps)
		}
	}
}

func TestGitLabMainAfterAMergeRunsOnce(t *testing.T) {
	data, err := os.ReadFile("../../ci/gitlab/workline.gitlab-ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	var top map[string]any
	if err := yaml.Unmarshal(data, &top); err != nil {
		t.Fatal(err)
	}
	job := func(name string) map[string]any {
		j, _ := top[name].(map[string]any)
		if j == nil {
			t.Fatalf("no job %s", name)
		}
		return j
	}
	main := job("workline:main")
	script := fmt.Sprint(main["script"])
	if s, _ := main["script"].([]any); len(s) == 0 || fmt.Sprint(s[0]) != "unset CLAUDE_CODE_OAUTH_TOKEN WORKLINE_GITLAB_TOKEN GITLAB_TOKEN" {
		t.Errorf("workline:main: does not unset the agent's and the bot's tokens first: %v", main["script"])
	}
	for _, want := range []string{"workline route main --ai none --forge none --gates-kept .workline-kept", "--input range=$CI_COMMIT_BEFORE_SHA..$CI_COMMIT_SHA"} {
		if !strings.Contains(script, want) {
			t.Errorf("workline:main: runs without %s: %s", want, script)
		}
	}
	if reports, _ := main["artifacts"].(map[string]any)["reports"].(map[string]any); reports["codequality"] != nil {
		t.Errorf("workline:main: gives GitLab a Code Quality report, which the merge requests' widget would read old findings as fixed against")
	}
	// A base written by a branch is never read: the findings come from
	// main's own pipeline artifacts, by the default branch and its job,
	// with the job's token; no cache holds them, which a merge request's
	// pipeline could write (`unprotect`).
	const fromMain = `/jobs/artifacts/$CI_DEFAULT_BRANCH/raw/.workline-kept/$1?job=workline:main`
	for name, v := range top {
		j, _ := v.(map[string]any)
		if c, _ := yaml.Marshal(j["cache"]); strings.Contains(string(c), ".workline-kept") || strings.Contains(string(c), "unprotect") {
			t.Errorf("%s: keeps or reads the stored base in a cache, which a branch's pipeline could write: %s", name, c)
		}
		paths, _ := yaml.Marshal(j["artifacts"])
		if strings.Contains(string(paths), ".workline-kept/") != (name == "workline:main") {
			t.Errorf("%s: its artifacts hold the stored base, the default branch's job's alone to write: %s", name, paths)
		}
	}
	if !strings.Contains(fmt.Sprint(main["script"]), fromMain) || !strings.Contains(fmt.Sprint(main["script"]), "kept commits") {
		t.Errorf("workline:main: does not carry the findings main's last pipeline kept: %v", main["script"])
	}
	gates := job("workline:gates")
	if script := fmt.Sprint(gates["script"]); !strings.Contains(script, fromMain) || !strings.Contains(script, `JOB-TOKEN: $CI_JOB_TOKEN`) ||
		!strings.Contains(script, `kept "$base.json"`) || !strings.Contains(script, "--gates-kept .workline-kept") {
		t.Errorf("workline:gates: does not read its merge base's findings from main's own artifacts: %v", script)
	}
	issue := job("workline:main:issue")
	if !strings.Contains(fmt.Sprint(issue["script"]), "export GITLAB_TOKEN=$WORKLINE_GITLAB_TOKEN") ||
		!strings.Contains(fmt.Sprint(issue["script"]), "workline apply --main main.json --forge gitlab") || fmt.Sprint(issue["needs"]) != "[workline:main]" {
		t.Errorf("workline:main:issue: does not keep main's run's findings with the bot's token: %v", issue["script"])
	}
	for _, name := range []string{"workline:main", "workline:main:issue"} {
		rules, _ := job(name)["rules"].([]any)
		if len(rules) != 1 || !strings.Contains(fmt.Sprint(rules[0]), `$CI_PIPELINE_SOURCE == "push" && $CI_COMMIT_BRANCH == $CI_DEFAULT_BRANCH`) {
			t.Errorf("%s: runs on %v, want a push to the default branch alone", name, rules)
		}
	}
}
