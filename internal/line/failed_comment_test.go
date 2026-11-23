package line

import (
	"strings"
	"testing"
	"time"

	"github.com/JN0V/workline/internal/kit/verdict"
)

// Apply stuck: Given an apply error that has failed twice, When the
// failure comment is written, Then it says the tries are spent and the
// issue goes to Blocked, not that the line runs again.
func TestFailureCommentSaysAnApplyErrorAtItsBudgetIsStuck(t *testing.T) {
	at := time.Date(2026, 11, 5, 8, 0, 0, 0, time.UTC)
	o := verdict.Outcome{Kind: verdict.MachineError, Of: ApplyKind, Said: "the diff does not apply"}
	e := (*Errors)(nil).with(o, at).with(o, at.Add(time.Hour))
	if e.By[ApplyKind] != ApplyBudget {
		t.Fatalf("setup: %d apply failures, want %d", e.By[ApplyKind], ApplyBudget)
	}
	body := failedBody(StateWrite{Issue: 7, Record: Record{Errors: e}}, "developer")
	if !strings.Contains(body, "its tries are spent") || !strings.Contains(body, "Blocked") {
		t.Errorf("the comment of an apply error at its budget does not say it is stuck and goes to Blocked:\n%s", body)
	}
	if strings.Contains(body, "2 of 2 tries") || strings.Contains(body, "runs the developer again") {
		t.Errorf("the comment of an apply error at its budget says a run will come:\n%s", body)
	}
}
