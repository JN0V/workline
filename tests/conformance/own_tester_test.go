package conformance

import (
	"os"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// workline's own line routes the tester (ADR-0055, issue #264): its
// test-command runs Go, so each job that routes it leaves Go set up — the
// gardening's judge, which proves the tests red, at night; workline.yml's
// judge on the tester's own branches, where the green proof runs. Taken
// from the cache, the engine's action sets no Go up unless asked.
func TestOwnTesterHasGoWhereItRuns(t *testing.T) {
	data, err := os.ReadFile("../../.workline/config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Routing struct {
			Events map[string][]string `yaml:"events"`
		} `yaml:"routing"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ event, workflow, job, want string }{
		{"schedule", "workline-gardening.yml", "judge", "true"},
		{"merge-request", "workline.yml", "judge", "'workline/tester/'"},
	} {
		if !slices.Contains(cfg.Routing.Events[c.event], "tester") {
			t.Errorf("the %s line does not route the tester", c.event)
			continue
		}
		data, err := os.ReadFile("../../.github/workflows/" + c.workflow)
		if err != nil {
			t.Fatal(err)
		}
		var wf struct {
			Jobs map[string]struct {
				Steps []engineStep `yaml:"steps"`
			} `yaml:"jobs"`
		}
		if err := yaml.Unmarshal(data, &wf); err != nil {
			t.Fatalf("%s: %v", c.workflow, err)
		}
		found := false
		for _, s := range wf.Jobs[c.job].Steps {
			if s.Uses == engineAction {
				found = true
				if !strings.Contains(s.With["go"], c.want) {
					t.Errorf("%s, job %s: the tester runs Go there, but the engine's action leaves Go set up only when %q", c.workflow, c.job, s.With["go"])
				}
			}
		}
		if !found {
			t.Errorf("%s, job %s: no engine step", c.workflow, c.job)
		}
	}
}
