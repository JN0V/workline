package line

import (
	"testing"
	"time"

	"github.com/JN0V/workline/internal/kit/verdict"
)

// A first tester run that ended with no answer it could use (kind
// answer) is not held for hours: right after it the issue waits the usual
// machine-error back-off, and past it the step is due again.
func TestFirstAnswerFailureIsDueAfterTheUsualBackoff(t *testing.T) {
	at := time.Date(2026, 11, 5, 8, 0, 0, 0, time.UTC)
	o := verdict.Outcome{Kind: verdict.MachineError, Of: "answer", Said: "no test file the tester may write was proposed for the issue's scenarios"}
	e := (*Errors)(nil).with(o, at)
	if again, until, spent := e.retry(at.Add(time.Minute)); again || !until.Equal(at.Add(ErrorBackoff)) || spent != "" {
		t.Fatalf("a minute after the first answer failure: again %v, until %v, spent %q; want a wait until the usual back-off, %s", again, until, spent, at.Add(ErrorBackoff))
	}
	if again, until, spent := e.retry(at.Add(ErrorBackoff)); !again || spent != "" {
		t.Fatalf("after the usual back-off, a hold kept until %v: again %v, spent %q; want due again", until, again, spent)
	}
}
