package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// Reason shown
//
// Given a measure that fails (the project sets no test command),
// When `workline test measure --json` runs,
// Then the result is on stdout and the reason is on stderr.
func TestMeasureJSONSaysItsReasonInTheLog(t *testing.T) {
	bin := sinceBuild(t)
	sinceEnv(t)
	repo := t.TempDir()
	stdout, stderr, code := sweepRun(t, bin, repo, "test", "measure", "--repo", repo, "--json")
	if code != 1 {
		t.Errorf("exit code = %d, want 1\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	var res struct {
		Status   string `json:"status"`
		Summary  string `json:"summary"`
		Findings []struct {
			Rule string `json:"rule"`
		} `json:"findings"`
	}
	if err := json.Unmarshal([]byte(stdout), &res); err != nil {
		t.Fatalf("stdout is not the JSON result: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if res.Status != "block" {
		t.Errorf("status = %q, want block\n%s", res.Status, stdout)
	}
	if !strings.Contains(stderr, "the measure did not run") {
		t.Errorf("the log (stderr) lacks the summary of the failed measure:\n%s", stderr)
	}
	if !strings.Contains(stderr, "setting-missing: no test command") {
		t.Errorf("the log (stderr) lacks the rule and message of the failed measure:\n%s", stderr)
	}
	if strings.Contains(stdout, "setting-missing: no test command") {
		t.Errorf("stdout holds the reason line, it should stay the JSON result alone:\n%s", stdout)
	}
}
