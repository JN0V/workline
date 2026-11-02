package conformance

import (
	enc "encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.yaml.in/yaml/v3"
)

// The step that says whether a reconcile is due, run as GitHub runs it,
// `gh` a stand-in (#853): its store is this workflow's runs, read in one
// call, since a run an event starts may not write the Actions cache. Not
// due while a reconciler's run or a heartbeat is queued, waiting or
// running, nor 55 minutes after the last that succeeded; due, it starts
// the reconciler's own run, never a heartbeat, which would cancel the one
// waiting.
func TestReconcileDueReadsTheReconcilersRuns(t *testing.T) {
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq is not installed")
	}
	ago := func(d time.Duration) string { return time.Now().UTC().Add(-d).Format(time.RFC3339) }
	run := func(title, status, conclusion string, ended time.Duration) map[string]any {
		return map[string]any{"display_title": title, "status": status, "conclusion": conclusion,
			"created_at": ago(ended + time.Hour), "updated_at": ago(ended)}
	}
	const beat, rec, issue = "workline develop heartbeat", "workline develop reconcile", "workline develop #5"
	cases := []struct {
		name  string
		runs  []map[string]any // nil: the call fails
		start bool
	}{
		{"a heartbeat waits", []map[string]any{run(beat, "waiting", "", 0), run(beat, "completed", "success", 2*time.Hour)}, false},
		{"the reconciler's run queued", []map[string]any{run(rec, "queued", "", 0)}, false},
		{"the last 10 min ago", []map[string]any{run(beat, "completed", "success", 10*time.Minute)}, false},
		{"the last 56 min ago, later ones failed, an issue's run going",
			[]map[string]any{run(issue, "in_progress", "", 0), run(beat, "completed", "cancelled", time.Minute),
				run(rec, "completed", "failure", 2*time.Minute), run(rec, "completed", "success", 56*time.Minute)}, true},
		{"none ever", []map[string]any{run(issue, "completed", "success", time.Minute)}, true},
		{"the runs not read", nil, false},
	}
	for _, f := range []string{"../../ci/github/workline-develop.yml", "../../.github/workflows/workline-develop.yml"} {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var wf struct {
			Jobs map[string]struct {
				Steps []struct {
					Run string `yaml:"run"`
				} `yaml:"steps"`
			} `yaml:"jobs"`
		}
		if err := yaml.Unmarshal(data, &wf); err != nil {
			t.Fatal(err)
		}
		steps := wf.Jobs["reconcile-due"].Steps
		if len(steps) != 1 {
			t.Fatalf("%s: reconcile-due has %d steps, want one", f, len(steps))
		}
		for _, c := range cases {
			dir := t.TempDir()
			bin := filepath.Join(dir, "bin")
			if err := os.MkdirAll(bin, 0o755); err != nil {
				t.Fatal(err)
			}
			listing := filepath.Join(dir, "runs-listing.json")
			body, _ := enc.Marshal(map[string]any{"total_count": len(c.runs), "workflow_runs": c.runs})
			if err := os.WriteFile(listing, body, 0o644); err != nil {
				t.Fatal(err)
			}
			fail := "0"
			if c.runs == nil {
				fail = "1"
			}
			calls := filepath.Join(dir, "calls")
			// gh api URL --jq F: the listing through jq; gh workflow run: logged.
			writeExec(t, filepath.Join(bin, "gh"), `#!/bin/bash
echo "$*" >> "`+calls+`"
if [ "$1" = api ]; then
  [ `+fail+` = 0 ] || { echo "HTTP 502" >&2; exit 1; }
  exec jq "$4" "`+listing+`"
fi
exit 0
`)
			summary := filepath.Join(dir, "summary")
			cmd := exec.Command("bash", "--noprofile", "--norc", "-eo", "pipefail", "-c", steps[0].Run)
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"), "GITHUB_STEP_SUMMARY="+summary, "GH_TOKEN=t",
				"GITHUB_REPOSITORY=o/r", "GITHUB_REF_NAME=main", "GITHUB_WORKFLOW_REF=o/r/.github/workflows/workline-develop.yml@refs/heads/main")
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Errorf("%s, %s: the step failed: %v\n%s", f, c.name, err, out)
				continue
			}
			got, _ := os.ReadFile(calls)
			lines := strings.Split(strings.TrimSpace(string(got)), "\n")
			if len(lines) == 0 || !strings.HasPrefix(lines[0], "api repos/o/r/actions/workflows/workline-develop.yml/runs?event=workflow_dispatch&branch=main&per_page=100 --jq ") {
				t.Errorf("%s, %s: the runs not read first, in one call: %q", f, c.name, got)
			}
			started := strings.Count(string(got), "workflow run workline-develop.yml -R o/r --ref main -f reconcile=true")
			if strings.Contains(string(got), "reconcile=next") || len(lines) != 1+started || (started == 1) != c.start {
				t.Errorf("%s, %s: calls %q, want the reconciler's run started: %v\n%s", f, c.name, got, c.start, out)
			}
			if c.runs == nil && !strings.Contains(string(out), "::warning title=workline::this workflow's runs not read") {
				t.Errorf("%s, %s: the failed read not said: %s", f, c.name, out)
			}
		}
	}
}
