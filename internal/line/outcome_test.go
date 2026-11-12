package line

import (
	"testing"
	"time"

	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/verdict"
)

// The outcome is the woken role's own run's, the first step: a handoff's
// error after it is not its own (ADR-0063, §6).
func TestOutcomeOfTheRolesOwnRun(t *testing.T) {
	failed := &engine.Result{Status: verdict.Block, NoVerdict: true,
		Findings: []verdict.Finding{{Rule: "engine-error", Message: "post wrote no readable verdict: gone"}}}
	cases := []struct {
		name   string
		steps  []Step
		status string
		want   string
	}{
		{"its run failed", []Step{{Name: "developer", Result: failed}}, verdict.Block, "machine-error(run)"},
		{"a handoff's", []Step{{Name: "developer", Result: &engine.Result{Status: verdict.Pass}}, {Name: "x (handoff)", Result: failed}}, verdict.Pass, "progress"},
		{"no step", nil, verdict.Block, "progress"},
	}
	for _, c := range cases {
		if got := outcomeOf(c.steps, c.status).String(); got != c.want {
			t.Errorf("%s: %s, want %s", c.name, got, c.want)
		}
	}
}

// A machine error is run again after the back-off, ErrorBudget times
// within ErrorWindow of the first; the next waits for a person
// (ADR-0063, decided 4).
func TestErrorsRetryWithinTheirBudget(t *testing.T) {
	at := time.Date(2026, 11, 5, 8, 0, 0, 0, time.UTC)
	o := verdict.Outcome{Kind: verdict.MachineError, Of: "build", Said: "error: no matching constructor"}
	e := (*Errors)(nil).with(o, at)
	if again, until, spent := e.retry(at); again || !until.Equal(at.Add(ErrorBackoff)) || spent != "" {
		t.Fatalf("right after the error: again %v, until %v, spent %q; want a wait until the back-off", again, until, spent)
	}
	if again, _, _ := e.retry(at.Add(ErrorBackoff)); !again {
		t.Fatal("past the back-off: not run again")
	}
	e = e.with(o, at.Add(time.Hour))
	if again, _, _ := e.retry(at.Add(2 * time.Hour)); !again || e.By["build"] != 2 {
		t.Fatalf("the second error: again %v, count %d", again, e.By["build"])
	}
	e = e.with(o, at.Add(3*time.Hour))
	if _, _, spent := e.retry(at.Add(4 * time.Hour)); spent == "" {
		t.Fatal("the third error of a kind: the budget not spent")
	}
	other := (*Errors)(nil).with(o, at)
	if _, _, spent := other.retry(at.Add(ErrorWindow)); spent == "" {
		t.Fatal("past the window: the budget not spent")
	}
	// Each kind its own window: a run error a day after a build error is
	// run again.
	run := verdict.Outcome{Kind: verdict.MachineError, Of: "run", Said: "the agent's run failed"}
	mixed := other.with(run, at.Add(ErrorWindow+time.Hour))
	if again, _, spent := mixed.retry(at.Add(ErrorWindow + 2*time.Hour)); !again || spent != "" {
		t.Fatalf("a new kind past another kind's window: again %v, spent %q", again, spent)
	}
	// A size error recorded before it blocked at once, "budget" then: no
	// try mends it (#1160).
	for _, kind := range []string{"budget", verdict.Size} {
		e := (*Errors)(nil).with(verdict.Outcome{Kind: verdict.MachineError, Of: kind, Said: "prompt too large"}, at)
		if again, _, spent := e.retry(at.Add(ErrorBackoff)); again || spent == "" {
			t.Errorf("a %s error past the back-off: again %v, spent %q; want spent at once", kind, again, spent)
		}
	}
}

// A developer's run lost in a record from before typed outcomes reads as
// one machine error of kind run, its time kept (#953).
func TestALostRunIsOneMachineError(t *testing.T) {
	e := errorsOf(Record{Lost: "2026-11-03T15:27:30Z"})
	if e == nil || e.By["run"] != 1 || e.Last != "2026-11-03T15:27:30Z" {
		t.Fatalf("errors of a lost run: %+v", e)
	}
	if again, _, _ := e.retry(time.Date(2026, 11, 5, 8, 0, 0, 0, time.UTC)); !again {
		t.Fatal("a run lost days ago, before typed outcomes: not run again")
	}
	if errorsOf(Record{}) != nil {
		t.Fatal("a record with no error has errors")
	}
}

// Code said lacking is new when it names none of what was given, an
// anchor or a folder's slash aside (#1410).
func TestSaysNew(t *testing.T) {
	for _, c := range []struct {
		given, said string
		want        bool
	}{
		{"core/x, feature/x", "core/x/src/main/kotlin/A.kt", true},
		{"core/x, feature/x", "feature/x/", false},
		{"src/a.go", "./src/a.go#Run, src/b.go", false},
		{"calc", "", false},
	} {
		if got := saysNew(c.given, c.said); got != c.want {
			t.Errorf("saysNew(%q, %q) = %v, want %v", c.given, c.said, got, c.want)
		}
	}
}
