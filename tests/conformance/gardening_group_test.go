package conformance

import (
	"os"
	"testing"

	"go.yaml.in/yaml/v3"
)

// Sunday in the backlog group
// Given the gardening workflow template,
// When its concurrency.group expression is evaluated with the schedule
// `7 18 * * 0` and an empty inputs.job,
// Then the group is workline-gardening-backlog: the Sunday full run
// triages the backlog, so it never runs beside a nightly or by-hand
// backlog run.
func TestGardeningSundayRunIsInTheBacklogGroup(t *testing.T) {
	data, err := os.ReadFile("../../defaults/ci/github/workline-gardening.yml")
	if err != nil {
		t.Fatal(err)
	}
	var wf struct {
		Concurrency struct {
			Group string `yaml:"group"`
		} `yaml:"concurrency"`
	}
	if err := yaml.Unmarshal(data, &wf); err != nil {
		t.Fatal(err)
	}
	if wf.Concurrency.Group == "" {
		t.Fatal("workline-gardening.yml: no concurrency group")
	}
	ctx := map[string]any{"github.event.schedule": "7 18 * * 0", "inputs.job": ""}
	if got, want := expand(t, wf.Concurrency.Group, ctx), "workline-gardening-backlog"; got != want {
		t.Errorf("Sunday schedule: concurrency group %q; want %q", got, want)
	}
}
