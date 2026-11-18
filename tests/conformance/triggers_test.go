package conformance

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// Each workflow's triggers, listed (#1665): check stays (a required
// check, the release); event moves the line, removed in a later slice or
// a proposed exception; time is a schedule or a dispatch. A trigger
// added, or removed without this list, fails here.
const (
	onCheck = "check"
	onEvent = "event"
	onTime  = "time"
	// onTemplate is a schedule of a job `workline tick` lists: this
	// repository's loop dispatches it, so its own workflow has none and
	// fails here if it comes back; the template keeps it, for a project
	// whose loop is a dry run (docs/tick.md#the-github-loop).
	onTemplate = "time, the template only"
)

// triggers is each workflow's, by file, the same for this repository's
// own (.github/workflows) and the template (defaults/ci/github) unless
// the file is in one only.
var triggers = map[string]map[string]string{
	"workline.yml":           {"pull_request": onCheck, "push": onCheck},
	"workline-warn.yml":      {"pull_request": onCheck},
	"workline-bounds.yml":    {"pull_request_target": onCheck},
	"test.yml":               {"pull_request": onCheck, "push": onCheck, "schedule": onTime, "workflow_dispatch": onTime},
	"release-please.yml":     {"push": onCheck},
	"release.yml":            {"workflow_call": onCheck},
	"release-candidate.yml":  {"push": onCheck},
	"workline-loop.yml":      {"schedule": onTime, "workflow_dispatch": onTime},
	"workline-gardening.yml": {"schedule": onTime, "workflow_dispatch": onTime},
	"workline-sample.yml":    {"schedule": onTemplate, "workflow_dispatch": onTime},
	"schedule-probe.yml":     {"schedule": onTime, "workflow_dispatch": onTime},
	"workline-develop.yml": {"issues": onEvent, "issue_comment": onEvent, "workflow_run": onEvent,
		"schedule": onTemplate, "workflow_dispatch": onTime},
	"workline-answer.yml": {"workflow_run": onEvent, "workflow_dispatch": onTime},
	"workline-merge.yml": {"pull_request_target": onEvent, "issues": onEvent, "schedule": onTemplate,
		"workflow_dispatch": onTime},
	"workline-objection.yml": {"pull_request_review_comment": onEvent},
	"workline-release.yml":   {"issues": onEvent, "workflow_run": onEvent},
	// The verdict the checks' results set (workline/verdict): a proposed
	// exception, kept until decided.
	"workline-apply.yml": {"workflow_run": onEvent, "push": onEvent},
	"workline-fork.yml":  {"workflow_run": onEvent},
}

// Only one copy differs: the template runs in a merge queue too.
var templateOnly = map[string]map[string]string{
	"workline.yml": {"merge_group": onCheck},
}

func TestEachWorkflowsTriggersListed(t *testing.T) {
	for _, dir := range []string{"../../.github/workflows", "../../defaults/ci/github"} {
		files, _ := filepath.Glob(filepath.Join(dir, "*.yml"))
		seen := 0
		for _, f := range files {
			data, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			var wf struct {
				On map[string]any `yaml:"on"`
			}
			if err := yaml.Unmarshal(data, &wf); err != nil {
				t.Fatalf("%s: %v", f, err)
			}
			if wf.On == nil {
				continue // not a workflow: an issue form
			}
			seen++
			base := filepath.Base(f)
			want, ok := triggers[base]
			if !ok {
				t.Errorf("%s: a workflow this list does not hold; add its triggers, each check, event or time", f)
				continue
			}
			all := map[string]string{}
			for k, v := range want {
				if v != onTemplate || strings.Contains(dir, "defaults") {
					all[k] = v
				}
			}
			if strings.Contains(dir, "defaults") {
				for k, v := range templateOnly[base] {
					all[k] = v
				}
			}
			var got, listed []string
			for k := range wf.On {
				got = append(got, k)
			}
			for k := range all {
				listed = append(listed, k)
			}
			slices.Sort(got)
			slices.Sort(listed)
			if !slices.Equal(got, listed) {
				t.Errorf("%s: triggered by %v, the list holds %v: a trigger added or removed is listed here, check, event or time", f, got, listed)
			}
		}
		if seen == 0 {
			t.Fatalf("no workflow read in %s", dir)
		}
	}
	var left []string
	for f, on := range triggers {
		for k, c := range on {
			if c == onEvent {
				left = append(left, f+": "+k)
			}
		}
	}
	slices.Sort(left)
	t.Logf("event triggers left, removed one per pull request in later slices of #1665:\n  %s", strings.Join(left, "\n  "))
}
