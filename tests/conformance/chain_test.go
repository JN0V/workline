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
		reconciled, byHand := false, false
		for _, s := range rec.Steps {
			// By hand (input reconcile) or on schedule: run whenever it last
			// ran; on the line's other events, 55 minutes after (#789).
			byHand = byHand || strings.Contains(s.Env["FORCED"], "inputs.reconcile == 'true'") && strings.Contains(s.Env["FORCED"], "github.event_name == 'schedule'")
			list := strings.Index(s.Run, "workline route reconcile --ai none")
			start := strings.Index(s.Run, "gh workflow run")
			reconciled = reconciled || list >= 0 && list < start && strings.Contains(s.Run, "--input reconcile=list") && strings.Contains(s.Run, "(.due // [])[]")
		}
		if !reconciled || !strings.Contains(rec.If, "github.event_name == 'schedule'") || !strings.Contains(rec.If, "(github.event_name == 'workflow_dispatch' && inputs.reconcile == 'true')") || !byHand ||
			rec.Permissions["actions"] != "write" || rec.Permissions["issues"] != "read" || rec.Permissions["contents"] != "read" {
			t.Errorf("%s, reconcile: not hourly and by hand, or not each due issue's run started with a read token: %q %v", file, rec.If, rec.Permissions)
		}
		if judge := wf.Jobs["judge"]; !strings.Contains(judge.If, "inputs.reconcile != 'true'") {
			t.Errorf("%s, judge: runs on the reconciler's dispatch too: %q", file, judge.If)
		}
		lineRunsOnce(t, file, data)
	}
}

// lineRunsOnce checks one event starts one run of an issue's step (#806):
// each run of an issue is named for it, every start of one is skipped
// while a run of that issue is queued or running, and the judge asks the
// line whether the step is due before it sets up the agent and the
// proof.
func lineRunsOnce(t *testing.T, file string, data []byte) {
	t.Helper()
	var wf struct {
		RunName string `yaml:"run-name"`
		Jobs    map[string]struct {
			Steps []struct {
				ID   string `yaml:"id"`
				Uses string `yaml:"uses"`
				If   string `yaml:"if"`
				Run  string `yaml:"run"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(data, &wf); err != nil {
		t.Fatalf("%s: %v", file, err)
	}
	if !strings.Contains(wf.RunName, "format('workline develop #{0}', inputs.issue)") || !strings.Contains(wf.RunName, "format('workline develop #{0}', github.event.issue.number)") {
		t.Errorf("%s: a run of an issue is not named for it: run-name %q", file, wf.RunName)
	}
	startsOnce(t, file, data)
	due, set := -1, -1
	for i, s := range wf.Jobs["judge"].Steps {
		switch {
		case s.ID == "due":
			due = i
			if !strings.Contains(s.Run, "--input reconcile=list") || !strings.Contains(s.Run, "--ai none") {
				t.Errorf("%s, judge: its due step does not ask the line, no agent", file)
			}
		case s.ID == "line" || strings.Contains(s.Run, "npm install") || s.Uses == "./.github/actions/tests" ||
			s.Uses == "./.github/actions/workline-setup":
			if set < 0 {
				set = i
			}
			if !strings.Contains(s.If, "steps.due.outputs.due == 'true'") {
				t.Errorf("%s, judge: step %d runs though the issue's step is not due: if %q", file, i, s.If)
			}
		}
	}
	if due < 0 || due > set {
		t.Errorf("%s, judge: no due step before the agent and the proof are set up", file)
	}
}

// startsOnce checks every start of an issue's run by workflow_dispatch is
// skipped while a run of that issue, other than this one, is queued or
// running (#806).
func startsOnce(t *testing.T, file string, data []byte) {
	t.Helper()
	var wf struct {
		Jobs map[string]struct {
			Steps []struct {
				Run string `yaml:"run"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(data, &wf); err != nil {
		t.Fatalf("%s: %v", file, err)
	}
	for name, job := range wf.Jobs {
		for _, s := range job.Steps {
			start := strings.Index(s.Run, "gh workflow run")
			if start < 0 || !strings.Contains(s.Run, "-f issue=") {
				continue
			}
			busy := strings.Index(s.Run, "gh run list")
			if busy < 0 || busy > start || !strings.Contains(s.Run, `select(.status != \"completed\"`) && !strings.Contains(s.Run, `select(.status != "completed"`) ||
				!strings.Contains(s.Run, `grep -qxF "workline develop #$`) {
				t.Errorf("%s, %s: an issue's run started though one of it is queued or running", file, name)
			}
		}
	}
}

// The night's states moved: their next steps started once each (#806).
func TestGardeningStartsEachIssueOnce(t *testing.T) {
	for _, file := range []string{"../../ci/github/workline-gardening.yml", "../../.github/workflows/workline-gardening.yml"} {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		startsOnce(t, file, data)
	}
}
