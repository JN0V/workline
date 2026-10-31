package conformance

import (
	"os"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// The line of states on GitHub (ADR-0059), in the template and in
// workline's own workflow alike: the job that applied asks the issue line,
// no agent, whether the issue's new state is due, and only then starts its
// run with the job's own token; the reconciler runs hourly or by hand,
// no agent, a read token, and starts each due issue's run.
func TestLineOfStatesChainsAndReconciles(t *testing.T) {
	for _, file := range []string{"../../ci/github/workline-develop.yml", "../../.github/workflows/workline-develop.yml"} {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		var wf struct {
			Jobs map[string]struct {
				If          string            `yaml:"if"`
				Permissions map[string]string `yaml:"permissions"`
				Steps       []struct {
					ID  string            `yaml:"id"`
					Run string            `yaml:"run"`
					Env map[string]string `yaml:"env"`
				} `yaml:"steps"`
			} `yaml:"jobs"`
		}
		if err := yaml.Unmarshal(data, &wf); err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		apply, chained := wf.Jobs["apply"], false
		for i, s := range apply.Steps {
			if s.ID != "next" {
				continue
			}
			ask := strings.Index(s.Run, `workline route issue --target "issue:$ISSUE" --ai none`)
			due := strings.Index(s.Run, `index($n)`)
			start := strings.Index(s.Run, "gh workflow run")
			chained = i == len(apply.Steps)-1 && ask >= 0 && strings.Contains(s.Run, "--input reconcile=list") && ask < due && due < start &&
				s.Env["GH_TOKEN"] == "${{ github.token }}"
		}
		if !chained || apply.Permissions["actions"] != "write" {
			t.Errorf("%s, apply: its last step does not ask the line whether the issue is due, then start its run with the job's token", file)
		}
		rec := wf.Jobs["reconcile"]
		reconciled := false
		for _, s := range rec.Steps {
			list := strings.Index(s.Run, "workline route reconcile --ai none")
			start := strings.Index(s.Run, "gh workflow run")
			reconciled = reconciled || list >= 0 && list < start && strings.Contains(s.Run, "--input reconcile=list") && strings.Contains(s.Run, "(.due // [])[]")
		}
		if !reconciled || !strings.Contains(rec.If, "github.event_name == 'schedule'") || !strings.Contains(rec.If, "inputs.reconcile == 'true'") ||
			rec.Permissions["actions"] != "write" || rec.Permissions["issues"] != "read" || rec.Permissions["contents"] != "read" {
			t.Errorf("%s, reconcile: not hourly and by hand, or not each due issue's run started with a read token: %q %v", file, rec.If, rec.Permissions)
		}
		if judge := wf.Jobs["judge"]; !strings.Contains(judge.If, "inputs.reconcile != 'true'") {
			t.Errorf("%s, judge: runs on the reconciler's dispatch too: %q", file, judge.If)
		}
	}
}
