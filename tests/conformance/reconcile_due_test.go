package conformance

import (
	enc "encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"go.yaml.in/yaml/v3"
)

// The step that says whether a reconcile is due, run as GitHub runs it,
// `gh` a stand-in (#853): its store is this workflow's runs, read in one
// call, since a run an event starts may not write the Actions cache. Not
// due while a reconciler's run or a heartbeat is queued, waiting or
// running, nor within the gap after the last that succeeded (55 minutes,
// or the variable WORKLINE_RECONCILE_GAP_MINUTES); due, it starts
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
	// A page of the issues' own runs alone: a busy line.
	busy := make([]map[string]any, 100)
	for i := range busy {
		busy[i] = run(issue, "completed", "success", time.Duration(i)*time.Second)
	}
	one := func(runs ...map[string]any) [][]map[string]any { return [][]map[string]any{runs} }
	cases := []struct {
		name  string
		pages [][]map[string]any // nil: the call fails
		start bool
		gap   string // the variable WORKLINE_RECONCILE_GAP_MINUTES, "" unset
	}{
		{"a heartbeat waits", one(run(beat, "waiting", "", 0), run(beat, "completed", "success", 2*time.Hour)), false, ""},
		{"the reconciler's run queued", one(run(rec, "queued", "", 0)), false, ""},
		{"the last 10 min ago", one(run(beat, "completed", "success", 10*time.Minute)), false, ""},
		{"the last 56 min ago, later ones failed, an issue's run going",
			one(run(issue, "in_progress", "", 0), run(beat, "completed", "cancelled", time.Minute),
				run(rec, "completed", "failure", 2*time.Minute), run(rec, "completed", "success", 56*time.Minute)), true, ""},
		{"none ever", one(run(issue, "completed", "success", time.Minute)), true, ""},
		{"the runs not read", nil, false, ""},
		{"a heartbeat waits past a page of issues' runs", [][]map[string]any{busy, {run(beat, "waiting", "", 0), run(rec, "completed", "success", 70*time.Minute)}}, false, ""},
		{"the last 10 min ago past a page of issues' runs", [][]map[string]any{busy, {run(beat, "completed", "success", 10*time.Minute)}}, false, ""},
		{"the last 10 min ago, a gap of 5", one(run(beat, "completed", "success", 10*time.Minute)), true, "5"},
		{"the last 3 min ago, a gap of 5", one(run(beat, "completed", "success", 3*time.Minute)), false, "5"},
		{"the last 10 min ago, a gap not a number", one(run(beat, "completed", "success", 10*time.Minute)), false, "soon"},
		{"the last 10 min ago, a gap of 08", one(run(beat, "completed", "success", 10*time.Minute)), true, "08"},
		{"the last 10 min ago, a gap of 0", one(run(beat, "completed", "success", 10*time.Minute)), false, "0"},
		{"the last 10 min ago, a gap too long", one(run(beat, "completed", "success", 10*time.Minute)), false, "99999999999999999999"},
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
			pages := filepath.Join(dir, "pages")
			if err := os.MkdirAll(pages, 0o755); err != nil {
				t.Fatal(err)
			}
			for i, runs := range append(slices.Clone(c.pages), []map[string]any{}) {
				body, _ := enc.Marshal(map[string]any{"total_count": 0, "workflow_runs": runs})
				if err := os.WriteFile(filepath.Join(pages, strconv.Itoa(i+1)+".json"), body, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			fail := "0"
			if c.pages == nil {
				fail = "1"
			}
			calls := filepath.Join(dir, "calls")
			// gh api URL --jq F: the URL's page through jq; gh workflow run: logged.
			writeExec(t, filepath.Join(bin, "gh"), `#!/bin/bash
echo "$*" >> "`+calls+`"
if [ "$1" = api ]; then
  [ `+fail+` = 0 ] || { echo "HTTP 502" >&2; exit 1; }
  exec jq "$4" "`+pages+`/${2##*&page=}.json"
fi
exit 0
`)
			summary := filepath.Join(dir, "summary")
			cmd := exec.Command("bash", "--noprofile", "--norc", "-eo", "pipefail", "-c", steps[0].Run)
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"), "GITHUB_STEP_SUMMARY="+summary, "GH_TOKEN=t",
				"GAP_MINUTES="+c.gap, "GITHUB_REPOSITORY=o/r", "GITHUB_REF_NAME=main", "GITHUB_WORKFLOW_REF=o/r/.github/workflows/workline-develop.yml@refs/heads/main")
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Errorf("%s, %s: the step failed: %v\n%s", f, c.name, err, out)
				continue
			}
			got, _ := os.ReadFile(calls)
			lines := strings.Split(strings.TrimSpace(string(got)), "\n")
			// One call a page, read on only past a page with no reconciler's
			// run that succeeded.
			reads := max(len(c.pages), 1)
			for i := range reads {
				if len(lines) <= i || !strings.HasPrefix(lines[i], "api repos/o/r/actions/workflows/workline-develop.yml/runs?event=workflow_dispatch&branch=main&per_page=100&page="+strconv.Itoa(i+1)+" --jq ") {
					t.Errorf("%s, %s: the runs not read first, one call a page, %d pages: %q", f, c.name, reads, got)
				}
			}
			started := strings.Count(string(got), "workflow run workline-develop.yml -R o/r --ref main -f reconcile=true")
			if strings.Contains(string(got), "reconcile=next") || len(lines) != reads+started || (started == 1) != c.start {
				t.Errorf("%s, %s: calls %q, want the reconciler's run started: %v\n%s", f, c.name, got, c.start, out)
			}
			if c.pages == nil && !strings.Contains(string(out), "::warning title=workline::this workflow's runs not read") {
				t.Errorf("%s, %s: the failed read not said: %s", f, c.name, out)
			}
			if bad := slices.Contains([]string{"soon", "0", "99999999999999999999"}, c.gap); bad != strings.Contains(string(out), "WORKLINE_RECONCILE_GAP_MINUTES="+c.gap+" is not 1 to 9999 minutes: 55 taken") {
				t.Errorf("%s, %s: the gap %q, a warning said: %v, want %v: %s", f, c.name, c.gap, !bad, bad, out)
			}
		}
	}
}
