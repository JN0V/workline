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
	"github.com/JN0V/workline/internal/kit/verdict"
	"github.com/JN0V/workline/internal/work"
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
		"body": backlog.Said("The tester", refusedPre, why, next)}}}
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

// refusePost ends post with v; blocked, the refusal is written on issue n,
// in the tester's one comment there, applied though the run blocks: post's
// out/on-block.yaml (ADR-0059, decision 1).
func refusePost(runDir string, n int, v verdict.Verdict) int {
	if code := sayBlock(runDir, n, &v); code != 0 {
		return code
	}
	return write(runDir, v)
}

// sayBlock writes, for a blocking verdict, the tester's comment on the
// issue (out/on-block.yaml); its exit, 0 when written or not needed.
// A refusal the answer mends that came back as the one before is final:
// asked again, the agent would loop (#1666).
func sayBlock(runDir string, n int, v *verdict.Verdict) int {
	if v.Status == verdict.Block {
		why := v.Summary
		if len(v.Findings) > 0 {
			why = v.Findings[0].Message
		}
		var mark string
		if v.Again && !v.Final && len(v.Findings) > 0 {
			var t task
			if data, err := os.ReadFile(filepath.Join(runDir, "in", taskFile)); err == nil && json.Unmarshal(data, &t) == nil {
				var times int
				times, mark = repeatedRefusal(t, *v)
				if times > 1 {
					v.Final, v.Again = true, false
					why = unchangedWhy(times, why)
					v.Summary = "a refusal came back unchanged: " + v.Summary
				}
			}
		}
		next := "waiting for its tests, the line blocks it with this reason; a person writes them, or puts " + backlog.LabelNeedsTests + " back for another run"
		if out := verdict.OutcomeOf(v.Status, v.Findings); out.Crossed() {
			// Too big for its context (#1160): no run again mends it; the
			// product owner is asked first (ADR-0064, step 3).
			next = "the line asks the product owner on its size at once, no run again on the same input fitting better: it narrows the issue's " + work.Name(work.Scope) + " or grants it a larger size, and sends it back, or leaves it to a person, who may raise context.budget in the tester's role.yaml"
		} else if out.Machine() {
			// A machine error is no refusal of the issue (ADR-0063, I3):
			// the line keeps its labels, and runs it again.
			next = "the line keeps its labels and runs the tester again, its comment saying when; past its tries, a person reads it"
		}
		said := []intent.Intention{{Kind: "comment", Value: map[string]any{"issue": n, "sticky": "issue",
			"body": backlog.Said("The tester", refusedRun, why, next) + mark}}}
		if err := intent.Write(filepath.Join(runDir, "out", "on-block.yaml"), said); err != nil {
			return fail(err)
		}
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
