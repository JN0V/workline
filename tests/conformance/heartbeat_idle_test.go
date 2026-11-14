package conformance

import (
	enc "encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// The step that starts the next reconcile, by the heartbeat, run as
// GitHub runs it, `gh` a stand-in whose environment has a wait timer of
// 55 minutes: it reads what the reconcile's `route reconcile` listed
// (due.json). A line with an issue due, or a review round due, gets its
// heartbeat; an idle line gets none, and the summary says why.
//
// Idle line
// Given no issue and no review round due, and none waiting on a time,
// When a reconcile ends,
// Then it starts no heartbeat and says why in its summary.
func TestHeartbeatNotStartedWhenTheLineIsIdle(t *testing.T) {
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq is not installed")
	}
	cases := []struct {
		name  string
		due   map[string]any // what the reconcile listed
		start bool
	}{
		{"nothing due", map[string]any{"status": "pass", "summary": "reconcile: nothing due (0 issues on the line acted on already)"}, false},
		{"an issue due", map[string]any{"status": "pass", "summary": "reconcile: 1 issue due: #5", "due": []int{5}}, true},
		{"a review round due", map[string]any{"status": "pass", "summary": "reconcile: 0 issues due:; 1 review round due",
			"reviews": []map[string]any{{"id": 7, "branch": "workline/developer/5"}}}, true},
	}
	for _, f := range []string{"../../ci/github/workline-develop.yml", "../../.github/workflows/workline-develop.yml"} {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var wf struct {
			Jobs map[string]struct {
				Steps []struct {
					Name string `yaml:"name"`
					Run  string `yaml:"run"`
				} `yaml:"steps"`
			} `yaml:"jobs"`
		}
		if err := yaml.Unmarshal(data, &wf); err != nil {
			t.Fatal(err)
		}
		var script string
		for _, s := range wf.Jobs["reconcile"].Steps {
			if s.Name == "The next reconcile, by the heartbeat" {
				script = s.Run
			}
		}
		if script == "" {
			t.Fatalf("%s: no step `The next reconcile, by the heartbeat` in the job reconcile", f)
		}
		for _, c := range cases {
			dir := t.TempDir()
			bin := filepath.Join(dir, "bin")
			if err := os.MkdirAll(bin, 0o755); err != nil {
				t.Fatal(err)
			}
			body, _ := enc.Marshal(c.due)
			if err := os.WriteFile(filepath.Join(dir, "due.json"), body, 0o644); err != nil {
				t.Fatal(err)
			}
			calls := filepath.Join(dir, "calls")
			// gh api: the environment's wait timer, 55 minutes; gh workflow run: logged.
			writeExec(t, filepath.Join(bin, "gh"), `#!/bin/bash
echo "$*" >> "`+calls+`"
if [ "$1" = api ]; then
  echo 55
fi
exit 0
`)
			summary := filepath.Join(dir, "summary")
			cmd := exec.Command("bash", "--noprofile", "--norc", "-eo", "pipefail", "-c", script)
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"), "GITHUB_STEP_SUMMARY="+summary, "GH_TOKEN=t",
				"GITHUB_REPOSITORY=o/r", "GITHUB_REF_NAME=main", "GITHUB_WORKFLOW_REF=o/r/.github/workflows/workline-develop.yml@refs/heads/main")
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Errorf("%s, %s: the step failed: %v\n%s", f, c.name, err, out)
				continue
			}
			got, _ := os.ReadFile(calls)
			said, _ := os.ReadFile(summary)
			started := strings.Contains(string(got), "workflow run workline-develop.yml --ref main -f reconcile=next")
			if started != c.start {
				t.Errorf("%s, %s: the heartbeat started: %v, want %v; calls %q", f, c.name, started, c.start, got)
			}
			if !c.start {
				if !strings.Contains(string(said), "no heartbeat") || strings.Contains(string(said), "wait timer of 5 minutes") {
					t.Errorf("%s, %s: the summary does not say why no heartbeat is started: %q", f, c.name, said)
				}
			}
		}
	}
}
