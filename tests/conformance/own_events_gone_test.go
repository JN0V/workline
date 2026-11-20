package conformance

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// The three workflows of workline's own that still start on an event, one
// removed per pull request: the beat dispatches what each started.
// workline-merge.yml is not among them: it keeps its pull_request_target
// event, so that a label taken off stops auto-merge at once, where the
// beat lists the merge job only once an hour.
var ownEventWorkflows = []string{
	"workline-develop.yml",
	"workline-answer.yml",
	"workline-objection.yml",
}

// One workflow, its events gone
// Given one of the workflows (develop, answer, objection),
// When its pull request merges,
// Then that workflow keeps only workflow_dispatch (and the loop's
// restart, a schedule, where it has one), and the triggers test lists no
// event for it.
func TestOwnWorkflowDoneKeepsOnlyDispatchAndListsNoEvent(t *testing.T) {
	done := 0
	var left []string
	for _, name := range ownEventWorkflows {
		data, err := os.ReadFile(filepath.Join("../../.github/workflows", name))
		if err != nil {
			t.Fatal(err)
		}
		var wf struct {
			On map[string]any `yaml:"on"`
		}
		if err := yaml.Unmarshal(data, &wf); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		var events []string
		for k := range wf.On {
			if k != "workflow_dispatch" && k != "schedule" {
				events = append(events, k)
			}
		}
		slices.Sort(events)
		if len(events) > 0 {
			left = append(left, name+": "+strings.Join(events, ", "))
			continue
		}
		// A workflow done: the beat dispatches it, and the triggers test
		// lists no event for it.
		done++
		if _, ok := wf.On["workflow_dispatch"]; !ok {
			t.Errorf("%s: no event left, and no workflow_dispatch: the beat cannot start it", name)
		}
		for k, c := range triggers[name] {
			if c == onEvent {
				t.Errorf("%s: the triggers test still lists the event %s", name, k)
			}
		}
	}
	if done == 0 {
		t.Errorf("none of the three workflows is done; events left:\n  %s", strings.Join(left, "\n  "))
	}
}
