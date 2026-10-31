package conformance

import (
	"os"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// workline's own line routes the tester (ADR-0055, issue #264) and the
// developer (ADR-0054, issue #696): both run the tester's test-command,
// which runs Go, so each job that routes them leaves Go set up —
// workline-develop.yml's judge, the line of states (ADR-0059), which
// proves the tester's tests red and the developer's code green;
// workline.yml's judge on the tester's own branches, where the green
// proof runs; workline-develop.yml's review round on the developer's
// pull request.
// Taken from the cache, the engine's action sets no Go up unless asked.
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
	for _, c := range []struct{ role, event, workflow, job, want string }{
		{"tester", "issue", "workline-develop.yml", "judge", "true"},
		{"tester", "merge-request", "workline.yml", "judge", "'workline/tester/'"},
		{"developer", "issue", "workline-develop.yml", "judge", "true"},
		{"developer", "issue", "workline-develop.yml", "review", "true"},
	} {
		if !slices.Contains(cfg.Routing.Events[c.event], c.role) {
			t.Errorf("the %s line does not route the %s", c.event, c.role)
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
					t.Errorf("%s, job %s: the %s runs Go there, but the engine's action leaves Go set up only when %q", c.workflow, c.job, c.role, s.With["go"])
				}
			}
		}
		if !found {
			t.Errorf("%s, job %s: no engine step", c.workflow, c.job)
		}
	}
}
