package github

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// developWorkflows are the two copies of the develop workflow: the
// template the project ships and the project's own workflow.
var developWorkflows = []string{
	"workline-develop.yml",
	"../../../.github/workflows/workline-develop.yml",
}

// roundScript reads the script of the review job's step `round`.
func roundScript(t *testing.T, file string) string {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var w struct {
		Jobs map[string]struct {
			Steps []struct {
				ID  string `yaml:"id"`
				Run string `yaml:"run"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(data, &w); err != nil {
		t.Fatalf("%s: %v", file, err)
	}
	for _, s := range w.Jobs["review"].Steps {
		if s.ID == "round" {
			return s.Run
		}
	}
	t.Fatalf("%s: the review job has no step `round`", file)
	return ""
}

// runRoundWithRefusedForge runs the round step's script with a gh that
// answers 403 and a git and a workline that only record they were called.
// It returns the step's log, the calls recorded and the step's exit error.
func runRoundWithRefusedForge(t *testing.T, file string) (string, string, error) {
	t.Helper()
	sh, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash is not installed")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.Mkdir(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	calls := filepath.Join(dir, "calls")
	stubs := map[string]string{
		"gh":       "echo 'gh: HTTP 403: Resource not accessible by integration (https://api.github.com/repos/o/r/pulls/7)' >&2\nexit 1\n",
		"git":      "echo \"git $*\" >> \"$CALLS\"\ncase \"$1\" in rev-parse) [ \"${2:-}\" = HEAD ] && echo abc123; exit 0;; esac\nexit 0\n",
		"workline": "echo \"workline $*\" >> \"$CALLS\"\necho '{}'\nexit 0\n",
		"jq":       "exit 0\n",
	}
	for name, body := range stubs {
		if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\n"+body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	script := filepath.Join(dir, "round.sh")
	if err := os.WriteFile(script, []byte(roundScript(t, file)), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(sh, "-e", script)
	cmd.Dir = dir
	cmd.Env = []string{
		"PATH=" + bin + ":/usr/bin:/bin",
		"CALLS=" + calls,
		"GITHUB_OUTPUT=" + filepath.Join(dir, "output"),
		"GITHUB_STEP_SUMMARY=" + filepath.Join(dir, "summary"),
		"GH_TOKEN=t", "GH_REPO=o/r", "PR=7", "BRANCH=workline/developer/12",
	}
	out, runErr := cmd.CombinedOutput()
	made, _ := os.ReadFile(calls)
	return string(out), string(made), runErr
}

// Forge error said
// Given a workflow step whose forge call fails with 403,
// When the step reads its result,
// Then the log prints the call and its status as an error line.
// It reads both copies of the develop workflow, the template and the
// project's own.
func TestDevelopRoundSaysAForgeCallRefused(t *testing.T) {
	for _, file := range developWorkflows {
		t.Run(file, func(t *testing.T) {
			out, _, _ := runRoundWithRefusedForge(t, file)
			for _, line := range strings.Split(out, "\n") {
				if strings.HasPrefix(line, "::error") && strings.Contains(line, "gh pr view") && strings.Contains(line, "403") {
					return
				}
			}
			t.Errorf("%s: no error line holding the call (gh pr view) and its status (403) in the log:\n%s", file, out)
		})
	}
}

// Never read as empty
// Given the same failure,
// When the step decides what to do,
// Then it does not act as if the forge answered nothing: it stops with
// an error, lays no checkout and starts no round.
func TestDevelopRoundStopsOnAForgeCallRefused(t *testing.T) {
	for _, file := range developWorkflows {
		t.Run(file, func(t *testing.T) {
			out, calls, err := runRoundWithRefusedForge(t, file)
			var exit *exec.ExitError
			if !errors.As(err, &exit) {
				t.Errorf("%s: the step did not stop with an error (err = %v):\n%s", file, err, out)
			}
			if strings.Contains(calls, "workline ") || strings.Contains(calls, "checkout") {
				t.Errorf("%s: the step went on as if the forge answered nothing; it called:\n%s", file, calls)
			}
		})
	}
}

// releaseWorkflow is the template of the release workflow.
const releaseWorkflow = "workline-release.yml"

// releaseScript reads the script of the first step of the release
// workflow's job `again`.
func releaseScript(t *testing.T, file string) string {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var w struct {
		Jobs map[string]struct {
			Steps []struct {
				Run string `yaml:"run"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(data, &w); err != nil {
		t.Fatalf("%s: %v", file, err)
	}
	steps := w.Jobs["again"].Steps
	if len(steps) == 0 || steps[0].Run == "" {
		t.Fatalf("%s: the job `again` has no step with a script", file)
	}
	return steps[0].Run
}

// Forge error said, for the release re-run
// Given the release workflow's step, with two open release pull
// requests, and a `gh run view` of the first one's run refused with 403,
// When the step reads that run's status,
// Then the log prints the call that failed (gh run view 101) and its
// status (403) as an error line; the step does not act as if the forge
// answered nothing: it still re-runs the second pull request's run (102),
// and ends with an error, so the refusal is not lost.
func TestReleaseSaysARefusedRunViewAndGoesOnToTheNextPullRequest(t *testing.T) {
	sh, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash is not installed")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.Mkdir(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	calls := filepath.Join(dir, "calls")
	gh := `case "$1 $2" in
"pr list") echo "1 sha1"; echo "2 sha2";;
"run list") case "$*" in *sha1*) echo 101;; *sha2*) echo 102;; esac;;
"run watch") exit 0;;
"run view") case "$3" in
  101) echo 'gh: HTTP 403: Resource not accessible by integration (https://api.github.com/repos/o/r/actions/runs/101)' >&2; exit 1;;
  *) echo completed;;
  esac;;
"run rerun") echo "gh run rerun $3" >> "$CALLS";;
esac
exit 0
`
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte("#!/bin/sh\n"+gh), 0o755); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(dir, "release.sh")
	if err := os.WriteFile(script, []byte(releaseScript(t, releaseWorkflow)), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(sh, "-e", script)
	cmd.Dir = dir
	cmd.Env = []string{
		"PATH=" + bin + ":/usr/bin:/bin",
		"CALLS=" + calls,
		"GH_TOKEN=t", "GH_REPO=o/r", "EVENT=issues",
		"MAIN_STARTED=", "MAIN_ENDED=",
		"BRANCHES=^(release-please--)",
	}
	outBytes, runErr := cmd.CombinedOutput()
	out := string(outBytes)
	made, _ := os.ReadFile(calls)

	said := false
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "::error") && strings.Contains(line, "gh run view 101") && strings.Contains(line, "403") {
			said = true
		}
	}
	if !said {
		t.Errorf("no error line holding the call (gh run view 101) and its status (403) in the log:\n%s", out)
	}
	if !strings.Contains(string(made), "gh run rerun 102") {
		t.Errorf("the refused call stopped the loop: the second pull request's run was not re-run; calls:\n%s", made)
	}
	var exit *exec.ExitError
	if !errors.As(runErr, &exit) {
		t.Errorf("the step did not end with an error after a refused call (err = %v):\n%s", runErr, out)
	}
}
