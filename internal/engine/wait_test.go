package engine

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/role"
)

// A run's wait on its agents (ai-max-seconds, ADR-0050): each call gets
// what is left at most, none once it is spent, and the answer that came
// too late is no answer.
func TestRunWaitIsBounded(t *testing.T) {
	r := &role.Role{} // its own model.timeout: 3m
	res := &Result{}
	if got, ok := res.bounded(r); !ok || got != r || !res.waitSince.IsZero() {
		t.Fatal("with no cap, the role answers within its own model.timeout, and no wait is counted")
	}

	res.maxSeconds = 30
	got, ok := res.bounded(r)
	limit, err := got.Model.AnswerTimeout()
	if !ok || err != nil || limit > 30*time.Second || limit < 29*time.Second || res.waitSince.IsZero() {
		t.Fatalf("the first call starts the wait and may take the run's 30 s: %v %v %v", ok, limit, err)
	}
	if r.Model.Timeout != "" {
		t.Fatal("the role itself is left as it was")
	}
	short := &role.Role{Model: role.Model{Timeout: "10s"}}
	if got, _ := res.bounded(short); got != short {
		t.Fatal("a model.timeout shorter than what is left stands")
	}
	if err := res.cutShort(fmt.Errorf("%w: no answer", agent.ErrUnavailable)); err == nil || errors.Is(err, errTimeSpent) {
		t.Fatalf("an agent failing with time left failed on its own: %v", err)
	}

	res.waitSince = time.Now().Add(-31 * time.Second)
	if !res.overTime() {
		t.Fatal("31 s waited, 30 allowed: over")
	}
	if _, ok := res.bounded(r); ok || !res.timeCapped {
		t.Fatal("once the wait is spent, no call is made, and the run says why")
	}
	if err := res.cutShort(fmt.Errorf("%w: claude did not answer", agent.ErrUnavailable)); !errors.Is(err, errTimeSpent) {
		t.Fatalf("an answer past the wait is no answer, not an outage: %v", err)
	}
	if err := res.cutShort(agent.ErrInvalidOutput); !errors.Is(err, agent.ErrInvalidOutput) {
		t.Fatalf("an answer that came, unreadable, stays what it is: %v", err)
	}
}
