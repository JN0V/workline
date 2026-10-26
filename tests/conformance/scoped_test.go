package conformance

import (
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// GitLab gives a variable scoped to an environment only to the jobs that
// declare it (#520): the template's jobs holding a token declare
// `workline`, recording no deployment (`prepare`), and the jobs running
// the project's tools (the gates, main's run, the tag's) declare none, so
// a token scoped to it never reaches them. No `rules:` tests a scoped
// variable, which GitLab may not see when it creates the pipeline: a job
// that needs the bot's token and lacks it says so and stops, green.
func TestGitLabTokensKeptFromTheProjectsTools(t *testing.T) {
	data, err := os.ReadFile("../../ci/gitlab/workline.gitlab-ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	var top map[string]any
	if err := yaml.Unmarshal(data, &top); err != nil {
		t.Fatal(err)
	}
	scoped := []string{"WORKLINE_GITLAB_TOKEN", "CLAUDE_CODE_OAUTH_TOKEN", "GITLAB_TOKEN", "WORKLINE_SIGNING_KEY"}
	// field is a job's own, else the one it extends'.
	field := func(name, key string) any {
		for seen := 0; name != "" && seen < 10; seen++ {
			job, _ := top[name].(map[string]any)
			if v, ok := job[key]; ok {
				return v
			}
			name, _ = job["extends"].(string)
		}
		return nil
	}
	// lines are a job's commands, before_script and script, its anchors
	// flattened as GitLab does.
	lines := func(name string) []string {
		var out []string
		for _, part := range []string{"before_script", "script"} {
			items, _ := field(name, part).([]any)
			for _, l := range items {
				if anchored, ok := l.([]any); ok {
					for _, a := range anchored {
						out = append(out, fmt.Sprint(a))
					}
					continue
				}
				out = append(out, fmt.Sprint(l))
			}
		}
		return out
	}
	reads := func(text, key string) bool { return strings.Contains(text, "$"+key) || strings.Contains(text, "${"+key) }
	tools := []string{"workline:gates", "workline:main", "workline:release"}
	holding := 0
	for name, v := range top {
		if _, ok := v.(map[string]any); !ok || strings.HasPrefix(name, ".") || field(name, "script") == nil {
			continue
		}
		env := field(name, "environment")
		receives := fmt.Sprint(env) == "map[action:prepare name:workline]"
		if env != nil && !receives {
			t.Errorf("%s: environment %v, want {name: workline, action: prepare}: the one its variables are scoped to, no deployment recorded", name, env)
		}
		if slices.Contains(tools, name) && env != nil {
			t.Errorf("%s: runs the project's tools, and declares the environment %v: it receives the tokens scoped there", name, env)
		}
		var used []string
		for _, l := range lines(name) {
			if strings.HasPrefix(strings.TrimSpace(l), "unset ") {
				continue // keeps one left to every environment from the tools
			}
			for _, key := range scoped {
				if reads(l, key) && !slices.Contains(used, key) {
					used = append(used, key)
				}
			}
		}
		if len(used) > 0 && !receives {
			t.Errorf("%s: reads %v and declares no environment workline: scoped there, they never reach it", name, used)
		}
		if receives {
			holding++
		}
		rules, _ := yaml.Marshal(field(name, "rules"))
		for _, key := range scoped {
			if reads(string(rules), key) {
				t.Errorf("%s: its rules test $%s, an environment's variable GitLab may not see when it creates the pipeline: %s", name, key, rules)
			}
		}
		// What needs the bot's token says so when it lacks it, before
		// anything uses it; the judge, the dry run and the base's read do
		// without it.
		if slices.Contains(used, "WORKLINE_GITLAB_TOKEN") && name != "workline:judge" && name != "workline:base" && name != "workline:release:dry-run" {
			said := -1
			for i, l := range lines(name) {
				switch {
				case said < 0 && strings.HasPrefix(l, `[ -n "${WORKLINE_GITLAB_TOKEN:-}" ] ||`) && strings.Contains(l, "exit 0"):
					said = i
				case said < 0 && reads(l, "WORKLINE_GITLAB_TOKEN"):
					t.Errorf("%s: uses the bot's token before saying when it lacks it: %s", name, l)
				}
			}
			if said < 0 {
				t.Errorf("%s: needs the bot's token, and never says when it lacks it", name)
			}
		}
	}
	if holding < 14 {
		t.Errorf("%d jobs declare the environment workline, want every job holding a token", holding)
	}
	// A job holding the bot's token and lacking it: said, green, nothing
	// after it run. In a repository of its own: the script sets git's
	// remote and identity, which must never reach this one's.
	repo := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", repo).CombinedOutput(); err != nil {
		t.Fatalf("git init: %s", out)
	}
	var script []string
	for _, l := range top["workline:apply"].(map[string]any)["script"].([]any) {
		if anchored, ok := l.([]any); ok {
			for _, a := range anchored {
				script = append(script, fmt.Sprint(a))
			}
			continue
		}
		script = append(script, fmt.Sprint(l))
	}
	cmd := exec.Command("sh", "-ec", strings.Join(script, "\n"))
	cmd.Dir = repo
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + repo, "GIT_CONFIG_GLOBAL=" + os.DevNull, "GIT_CONFIG_NOSYSTEM=1", "GIT_CEILING_DIRECTORIES=" + repo}
	out, err := cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "WORKLINE_GITLAB_TOKEN is not set, or not scoped to the environment workline: nothing to do") {
		t.Errorf("workline:apply without the bot's token: %v\n%s", err, out)
	}
	if set, _ := exec.Command("git", "-C", repo, "config", "--local", "--list").Output(); strings.Contains(string(set), "remote.") || strings.Contains(string(set), "user.") {
		t.Errorf("workline:apply without the bot's token went on past saying so: git set to\n%s", set)
	}
}
