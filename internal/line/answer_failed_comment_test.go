package line

import (
	"strings"
	"testing"
	"time"

	"github.com/JN0V/workline/internal/kit/verdict"
)

// answerFailedComment is the failure comment the line writes for a
// tester's run that ended with no answer it could use, after n such runs
// in a row, the last at at.
func answerFailedComment(n int, at time.Time) string {
	o := verdict.Outcome{Kind: verdict.MachineError, Of: "answer", Said: "no test file the tester may write was proposed for the issue's scenarios"}
	var e *Errors
	for i := 0; i < n; i++ {
		e = e.with(o, at.Add(time.Duration(i)*time.Minute))
	}
	w := StateWrite{Issue: 7, Failed: true, Record: Record{State: "testing", Errors: e}}
	return failedBody(w, "tester")
}

// Answer retry time: Given an answer error that failed once, When the
// failure comment is written, Then it gives the six-hour hold, not 10
// minutes.
func TestAnswerFailureCommentGivesTheSixHourHold(t *testing.T) {
	at := time.Date(2026, 11, 5, 8, 0, 0, 0, time.UTC)
	got := answerFailedComment(1, at)
	if !strings.Contains(got, "from 2026-11-05 14:00 UTC") {
		t.Errorf("the comment after one answer error does not give the six-hour hold, 2026-11-05 14:00 UTC:\n%s", got)
	}
	if strings.Contains(got, "2026-11-05 08:10 UTC") {
		t.Errorf("the comment after one answer error gives the 10 minutes back-off:\n%s", got)
	}
	if !strings.Contains(got, "1 of 2 tries") {
		t.Errorf("the comment after one answer error does not show 1 of 2 tries:\n%s", got)
	}
}

// Answer spent: Given an answer error at AnswerBudget, When the comment is
// written, Then it says the next run blocks the issue.
func TestAnswerFailureCommentSaysTheNextRunBlocks(t *testing.T) {
	at := time.Date(2026, 11, 5, 8, 0, 0, 0, time.UTC)
	got := answerFailedComment(AnswerBudget, at)
	if !strings.Contains(got, "its tries are spent: the line moves it to Blocked for a person at its next run") {
		t.Errorf("the comment at AnswerBudget answer errors does not say the next run blocks the issue:\n%s", got)
	}
	if strings.Contains(got, "again on it from") {
		t.Errorf("the comment at AnswerBudget answer errors still promises a retry:\n%s", got)
	}
}
