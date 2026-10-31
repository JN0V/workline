package tester

// What the tester writes on an issue it refuses (ADR-0059, decision 1):
// why, and what comes next, in its one comment there, edited in place
// (principle 14). Never a job's log alone.

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/verdict"
)

// saidFile holds the verdict of a run that refused its issue, said there:
// post writes it, after the engine applied the comment pre left.
const saidFile = "said.json"

// refuseIssue ends pre with v, a refusal of issue n, written on the issue: the
// comment is applied though the run blocks (`on-block`), no agent asked.
func refuseIssue(runDir string, n int, v verdict.Verdict, next string) int {
	why := v.Summary
	if len(v.Findings) > 0 {
		why = v.Findings[0].Message
	}
	said := []intent.Intention{{Kind: "comment", Value: map[string]any{"issue": n, "sticky": "issue", "on-block": true,
		"body": backlog.Said("The tester", "writes no tests for this issue yet", why, next)}}}
	if err := intent.Write(filepath.Join(runDir, "in", "fallback.yaml"), said); err != nil {
		return fail(err)
	}
	data, err := json.Marshal(v)
	if err != nil {
		return fail(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "in", saidFile), data, 0o644); err != nil {
		return fail(err)
	}
	return 0
}

// saidPost writes the verdict pre left; its exit says it.
func saidPost(runDir string) (int, bool) {
	data, err := os.ReadFile(filepath.Join(runDir, "in", saidFile))
	if os.IsNotExist(err) {
		return 0, false
	}
	var v verdict.Verdict
	if err == nil {
		err = json.Unmarshal(data, &v)
	}
	if err != nil {
		return fail(err), true
	}
	return write(runDir, v), true
}
