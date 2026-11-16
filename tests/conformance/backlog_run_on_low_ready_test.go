package conformance

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// backlogDispatchRe is the command that starts the backlog run: the
// gardening workflow dispatched with `skip` naming the documentalist, the
// input the scheduled backlog run uses.
var backlogDispatchRe = regexp.MustCompile(`gh workflow run[^\n]*workline-gardening\.yml[^\n]*skip=["']?documentalist\b`)

// backlogDispatch reads a workflow and returns the job, the script (its
// line continuations joined) and the job's permissions of the step that
// dispatches the backlog run; it fails the test when no step does.
func backlogDispatch(t *testing.T, file string) (string, string, map[string]any) {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var wf struct {
		Jobs map[string]struct {
			Permissions map[string]any `yaml:"permissions"`
			Steps       []struct {
				Run string `yaml:"run"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(data, &wf); err != nil {
		t.Fatalf("%s: %v", file, err)
	}
	var names []string
	for name := range wf.Jobs {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		for _, s := range wf.Jobs[name].Steps {
			run := strings.ReplaceAll(s.Run, "\\\n", " ")
			if backlogDispatchRe.MatchString(run) {
				return name, run, wf.Jobs[name].Permissions
			}
		}
	}
	t.Fatalf("%s: no step dispatches workline-gardening.yml with skip=documentalist, the backlog run", file)
	return "", "", nil
}

// dispatchGuard is the condition the backlog dispatch runs under: the
// `if` that encloses it, or what comes before it on its own line.
func dispatchGuard(run string) string {
	lines := strings.Split(run, "\n")
	at := -1
	for i, l := range lines {
		if backlogDispatchRe.MatchString(l) {
			at = i
			break
		}
	}
	if at < 0 {
		return ""
	}
	if before, _, ok := strings.Cut(lines[at], "gh workflow run"); ok && strings.Contains(before, "&&") {
		return before
	}
	depth := 0
	for i := at - 1; i >= 0; i-- {
		l := strings.TrimSpace(lines[i])
		switch {
		case l == "fi" || strings.HasPrefix(l, "fi "):
			depth++
		case strings.HasPrefix(l, "if "):
			if depth == 0 {
				return l
			}
			depth--
		case strings.HasPrefix(l, "elif ") && depth == 0:
			return l
		}
	}
	return ""
}

// Ready drops below target
// Given a target of 10 and 10 ready issues,
// When the developer takes one and 9 remain,
// Then a backlog run starts within 5 minutes: the job that moves the issue
// out of Ready dispatches workline-gardening.yml with skip=documentalist
// (the roles of the scheduled backlog run), with the token that may start
// a workflow, and the count is the engine's: the shell holds no target and
// counts no issue.
func TestDevelopStartsTheBacklogRunWhenReadyDropsBelowTarget(t *testing.T) {
	job, run, perms := backlogDispatch(t, "../../.github/workflows/workline-develop.yml")
	if perms["actions"] != "write" {
		t.Errorf("workline-develop.yml, %s: dispatches the backlog run without `actions: write`: %v", job, perms)
	}
	for _, own := range []string{"wip", "gh issue list", "-lt ", "ready: 10"} {
		if strings.Contains(run, own) {
			t.Errorf("workline-develop.yml, %s: the shell counts Ready itself (%q): the engine says whether Ready is below target", job, own)
		}
	}
	if guard := dispatchGuard(run); !strings.Contains(guard, "jq") || !strings.Contains(guard, ".json") {
		t.Errorf("workline-develop.yml, %s: the backlog run is not dispatched on what the engine's --json says: guard %q", job, guard)
	}
}

// A run already queued or running
// Given a backlog run running,
// When Ready drops below target again,
// Then no second run runs beside it; at most one waits for it to end: the
// concurrency group lets one run at a time, none cancelled, and the job
// looks for a backlog run already waiting before it dispatches one, as
// GitHub would replace the waiting one with the newer.
func TestDevelopStartsNoSecondBacklogRunBesideOneWaiting(t *testing.T) {
	_, run, _ := backlogDispatch(t, "../../.github/workflows/workline-develop.yml")
	listed := regexp.MustCompile(`gh run list[^\n]*workline-gardening\.yml|actions/workflows/workline-gardening\.yml/runs`).FindStringIndex(run)
	if listed == nil {
		t.Fatal("workline-develop.yml: no look at the gardening runs already waiting before the backlog run is dispatched")
	}
	if !regexp.MustCompile(`queued|pending|waiting`).MatchString(run[listed[0]:]) {
		t.Error("workline-develop.yml: the gardening runs listed are not read for one queued or waiting")
	}
	if at := backlogDispatchRe.FindStringIndex(run); at == nil || listed[0] > at[0] {
		t.Error("workline-develop.yml: the waiting runs are looked at after the dispatch, not before")
	}

	data, err := os.ReadFile("../../.github/workflows/workline-gardening.yml")
	if err != nil {
		t.Fatal(err)
	}
	var garden struct {
		Concurrency map[string]any `yaml:"concurrency"`
	}
	if err := yaml.Unmarshal(data, &garden); err != nil {
		t.Fatal(err)
	}
	group := fmt.Sprint(garden.Concurrency["group"])
	if !strings.Contains(group, "'workline-gardening-backlog'") || !strings.Contains(group, "github.event.schedule == '7 18 * * 0'") || garden.Concurrency["cancel-in-progress"] != false {
		t.Errorf("workline-gardening.yml: a dispatched run does not land in the backlog group, one at a time, none cancelled: %v", garden.Concurrency)
	}
}

// Ready at or above target
// Given a target of 10 and 11 ready issues,
// When the developer takes one,
// Then no backlog run starts: the dispatch runs only under what the engine
// says (a jq condition on its --json output), never unconditionally, in the
// project's workflow and in the template's.
func TestDevelopStartsNoBacklogRunAtOrAboveTarget(t *testing.T) {
	for _, file := range []string{"../../.github/workflows/workline-develop.yml", "../../defaults/ci/github/workline-develop.yml"} {
		_, run, _ := backlogDispatch(t, file)
		guard := dispatchGuard(run)
		if guard == "" {
			t.Errorf("%s: the backlog run is dispatched whatever the engine says of Ready", file)
			continue
		}
		if !strings.Contains(guard, "jq") {
			t.Errorf("%s: the backlog run's condition does not read the engine's answer: %q", file, guard)
		}
	}
}

// A project from the template
// Given a project set up from the GitHub template with the product owner on
// its schedule line,
// When Ready drops below its target,
// Then its backlog run starts the same way: the template's develop workflow
// dispatches its gardening workflow with skip=documentalist under the
// engine's answer, with the token to do it, and looks for a run waiting; its
// gardening workflow takes the skip input, passes it to the schedule line,
// and holds one backlog run at a time in a group of its own, the weekly docs
// run apart.
func TestTemplateStartsItsBacklogRunWhenReadyDropsBelowTarget(t *testing.T) {
	job, run, perms := backlogDispatch(t, "../../defaults/ci/github/workline-develop.yml")
	if perms["actions"] != "write" {
		t.Errorf("defaults/ci/github/workline-develop.yml, %s: dispatches the backlog run without `actions: write`: %v", job, perms)
	}
	if guard := dispatchGuard(run); !strings.Contains(guard, "jq") {
		t.Errorf("defaults/ci/github/workline-develop.yml, %s: the backlog run is not dispatched on the engine's answer: %q", job, guard)
	}
	if !regexp.MustCompile(`gh run list[^\n]*workline-gardening\.yml|actions/workflows/workline-gardening\.yml/runs`).MatchString(run) {
		t.Errorf("defaults/ci/github/workline-develop.yml, %s: no look at a backlog run already waiting", job)
	}

	data, err := os.ReadFile("../../defaults/ci/github/workline-gardening.yml")
	if err != nil {
		t.Fatal(err)
	}
	var garden struct {
		On          map[string]any `yaml:"on"`
		Concurrency map[string]any `yaml:"concurrency"`
	}
	if err := yaml.Unmarshal(data, &garden); err != nil {
		t.Fatal(err)
	}
	group := fmt.Sprint(garden.Concurrency["group"])
	if !strings.Contains(group, "'workline-gardening-backlog'") || !strings.Contains(group, "github.run_id") || garden.Concurrency["cancel-in-progress"] != false {
		t.Errorf("defaults/ci/github/workline-gardening.yml: no backlog concurrency group, one run at a time and the weekly docs run apart: %v", garden.Concurrency)
	}
	dispatch, _ := garden.On["workflow_dispatch"].(map[string]any)
	inputs, _ := dispatch["inputs"].(map[string]any)
	if _, ok := inputs["skip"]; !ok {
		t.Errorf("defaults/ci/github/workline-gardening.yml: workflow_dispatch takes no `skip` input: %v", dispatch)
	}
	if !strings.Contains(string(data), `--input skip=`) {
		t.Error("defaults/ci/github/workline-gardening.yml: the schedule line is not given the skip input")
	}
}
