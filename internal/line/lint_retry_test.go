package line

import (
	"strings"
	"testing"
	"time"

	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/kit/verdict"
)

// Retries bounded: Given the tester's tests failing the project's lint on
// every run, When its retries are spent, Then the issue is blocked, its
// reason naming the last lint finding. A lint finding the agent can mend
// is a machine error of the tester's run, run again within the line's
// budget (ErrorBudget); past it, the issue waits for a person, with the
// finding said.
func TestLintSlipRetriesAreBounded(t *testing.T) {
	const finding = "the gate lint, tests/test_mul_by_zero.sh:4: Exit status of `rm` is not checked (errcheck)"
	steps := []Step{{Name: "tester", Result: &engine.Result{Status: verdict.Block,
		Findings: []verdict.Finding{{Rule: "tests-lint", Where: "tests/test_mul_by_zero.sh:4", Message: finding}}}}}
	o := outcomeOf(steps, verdict.Block)
	if !o.Machine() {
		t.Fatalf("a lint slip in the tester's tests is %s, want a machine error the line runs again", o)
	}
	at := time.Date(2026, 11, 5, 8, 0, 0, 0, time.UTC)
	e := (*Errors)(nil).with(o, at)
	if again, _, spent := e.retry(at.Add(ErrorBackoff)); !again || spent != "" {
		t.Fatalf("after the first slip: again %v, spent %q; want the tester run again", again, spent)
	}
	e = e.with(o, at.Add(time.Hour))
	if again, _, spent := e.retry(at.Add(2 * time.Hour)); !again || spent != "" {
		t.Fatalf("after the second slip: again %v, spent %q; want the tester run again", again, spent)
	}
	e = e.with(o, at.Add(3*time.Hour))
	again, _, spent := e.retry(at.Add(4 * time.Hour))
	if again || !strings.Contains(spent, "Exit status of `rm` is not checked") {
		t.Fatalf("after the third slip: again %v, spent %q; want the retries spent, the last lint finding named", again, spent)
	}
}
