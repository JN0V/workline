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
// run with the job's own token; the reconciler runs in its own run, by
// the heartbeat, by hand or once due, no agent, a read token, and starts
// each due issue's run.
func TestLineOfStatesChainsAndReconciles(t *testing.T) {
	for _, file := range []string{"../../ci/github/workline-develop.yml", "../../.github/workflows/workline-develop.yml"} {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		var wf struct {
			RunName string `yaml:"run-name"`
			Jobs    map[string]struct {
				If          string            `yaml:"if"`
				Concurrency map[string]any    `yaml:"concurrency"`
				Permissions map[string]string `yaml:"permissions"`
				Steps       []struct {
					ID   string            `yaml:"id"`
					Uses string            `yaml:"uses"`
					Run  string            `yaml:"run"`
					Env  map[string]string `yaml:"env"`
				} `yaml:"steps"`
			} `yaml:"jobs"`
		}
		if err := yaml.Unmarshal(data, &wf); err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		apply, chained, roomed := wf.Jobs["apply"], -1, false
		for i, s := range apply.Steps {
			switch s.ID {
			case "next":
				ask := strings.Index(s.Run, `workline route issue --target "issue:$ISSUE" --ai none`)
				due := strings.Index(s.Run, `index($n)`)
				start := strings.Index(s.Run, "gh workflow run")
				if ask >= 0 && strings.Contains(s.Run, "--input reconcile=list") && ask < due && due < start && s.Env["GH_TOKEN"] == "${{ github.token }}" {
					chained = i
				}
			case "room":
				// Then the room its move freed (#960): the issues waiting on
				// it, each one's run started, the issue's own left to next.
				ask := strings.Index(s.Run, "workline route reconcile --ai none")
				start := strings.Index(s.Run, "gh workflow run")
				roomed = chained >= 0 && i == len(apply.Steps)-1 && ask >= 0 && ask < start && strings.Contains(s.Run, "--input reconcile=list --input room=true") &&
					strings.Contains(s.Run, "(.due // [])[]") && strings.Contains(s.Run, `[ "$n" != "$ISSUE" ]`) && s.Env["GH_TOKEN"] == "${{ github.token }}"
			}
		}
		if chained < 0 || apply.Permissions["actions"] != "write" {
			t.Errorf("%s, apply: it does not ask the line whether the issue is due, then start its run with the job's token", file)
		}
		if !roomed {
			t.Errorf("%s, apply: its last step does not ask the line, after the issue's own, which issues the room freed wakes, and start their runs", file)
		}
		rec := wf.Jobs["reconcile"]
		reconciled, cached := false, false
		for _, s := range rec.Steps {
			list := strings.Index(s.Run, "workline route reconcile --ai none")
			start := strings.Index(s.Run, "gh workflow run")
			reconciled = reconciled || list >= 0 && list < start && strings.Contains(s.Run, "--input reconcile=list") && strings.Contains(s.Run, "(.due // [])[]")
			cached = cached || strings.HasPrefix(s.Uses, "actions/cache")
		}
		// Its own run alone, by the heartbeat (next) or by hand (true):
		// those runs, by name, keep when it last ran; a run an event starts
		// may not write the Actions cache (#853).
		if !reconciled || !strings.Contains(rec.If, "github.event_name == 'workflow_dispatch' && (inputs.reconcile == 'true' || inputs.reconcile == 'next')") ||
			strings.Contains(rec.If, "'schedule'") || strings.Contains(rec.If, "'issues'") || cached ||
			rec.Permissions["actions"] != "write" || rec.Permissions["issues"] != "read" || rec.Permissions["contents"] != "read" {
			t.Errorf("%s, reconcile: not its own run alone, or not each due issue's run started with a read token, or kept in the cache: %q %v", file, rec.If, rec.Permissions)
		}
		// The schedule and the line's every label ask whether it is due:
		// the heartbeat stopped (#789, #845). One at a time, no other
		// right than to start a run.
		due := wf.Jobs["reconcile-due"]
		if !strings.Contains(due.If, "github.event_name == 'schedule'") || !strings.Contains(due.If, "(github.event_name == 'issues' && startsWith(github.event.label.name, 'workline:'))") ||
			due.Concurrency["group"] != "workline-reconcile-due" || due.Concurrency["cancel-in-progress"] != false ||
			len(due.Permissions) != 1 || due.Permissions["actions"] != "write" || len(due.Steps) != 1 {
			t.Errorf("%s, reconcile-due: not on the schedule and a label, one at a time, or holds more than the right to start a run: %q %v %v", file, due.If, due.Concurrency, due.Permissions)
		}
		if !strings.Contains(wf.RunName, "inputs.reconcile == 'next' && 'workline develop heartbeat'") ||
			!strings.Contains(wf.RunName, "inputs.reconcile == 'true' && 'workline develop reconcile'") {
			t.Errorf("%s: the reconciler's runs not named, its store: run-name %q", file, wf.RunName)
		}
		if judge := wf.Jobs["judge"]; !strings.Contains(judge.If, "inputs.reconcile == ''") {
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
	// Never a title a person writes: one could take another issue's name.
	if !strings.Contains(wf.RunName, "format('workline develop #{0}', inputs.issue)") || !strings.Contains(wf.RunName, "format('workline develop #{0}', github.event.issue.number)") ||
		strings.Contains(wf.RunName, ".title") ||
		// A label of the engine's, or not a state's, runs no step: its run
		// never named for the issue, which would mark it busy.
		!strings.Contains(wf.RunName, "github.event.sender.type != 'Bot'") || !strings.Contains(wf.RunName, "startsWith(github.event.label.name, 'workline:')") {
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
// skipped while a run of that issue is queued or running (#806): only
// the runs not ended are listed, never pushed out of a page by the many
// ended ones.
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
		if name == "apply" && strings.Contains(file, "develop") {
			// The chain starts its next step even so: the run waiting may
			// have asked the line before this apply moved the state.
			continue
		}
		for _, s := range job.Steps {
			start := strings.Index(s.Run, "gh workflow run")
			if start < 0 || !strings.Contains(s.Run, "-f issue=") {
				continue
			}
			busy := strings.Index(s.Run, "gh run list")
			if busy < 0 || busy > start || !strings.Contains(s.Run, "for s in queued in_progress pending waiting") || !strings.Contains(s.Run, `--status "$s"`) ||
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
