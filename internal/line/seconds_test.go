package line

import (
	"testing"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/routing"
)

// A time is said compactly: tenths under 10 s, whole seconds under a
// minute, then minutes; a budget of whole seconds or minutes as written.
func TestSecondsSaidCompactly(t *testing.T) {
	for s, want := range map[float64]string{0: "0 s", 0.5: "0.5 s", 1: "1 s", 9.94: "9.9 s", 9.96: "10 s", 42.4: "42 s",
		59.6: "1 min", 180: "3 min", 192.4: "3 min 12 s", 600: "10 min"} {
		if got := Seconds(s); got != want {
			t.Errorf("Seconds(%v) = %q, want %q", s, got, want)
		}
	}
}

// Each step says its seconds after the total, with what its agents took
// of them; over the moment's budget, the step that took the most is
// named, once; within it, nothing.
func TestTimesAndTheBudget(t *testing.T) {
	r := &Result{Seconds: 200.3, Steps: []Step{
		{Name: "committer", Seconds: 0.5},
		{Name: "gate:lint", Seconds: 12, From: "gates.json"},
		{Name: "reviewer", Seconds: 187.4, Result: &engine.Result{Calls: []agent.Call{
			{Task: "part", Seconds: 120}, {Task: "judge", Seconds: 30}, {Task: "judge", Seconds: 25.5}}}},
	}}
	if got, want := r.Times(), "3 min 20 s in all — committer 0.5 s, gate:lint 12 s (its job), reviewer 3 min 7 s (lenses 2 min, judges 56 s)"; got != want {
		t.Errorf("Times() = %q\nwant %q", got, want)
	}
	cfg := &routing.Config{Budgets: map[string]float64{"verdict": 180}}
	Measure(r, cfg, "verdict")
	want := "the head's verdict took 3 min 20 s, over its budget of 3 min (ADR-0050); the step that took the most: reviewer, 3 min 7 s (lenses 2 min, judges 56 s)"
	if got := r.OverBudget(); got != want {
		t.Errorf("OverBudget() = %q\nwant %q", got, want)
	}
	if r.Moment.Step != "reviewer" || !r.Moment.Over {
		t.Errorf("moment = %+v", *r.Moment)
	}
	cfg.Budgets["verdict"] = 600
	Measure(r, cfg, "verdict")
	if got := r.OverBudget(); got != "" || r.Moment.Over || r.Moment.Step != "" {
		t.Errorf("within its budget, said %q, moment %+v", got, *r.Moment)
	}
	Measure(r, cfg, "")
	if r.Moment != nil || r.OverBudget() != "" {
		t.Errorf("a moment with no budget measured: %+v", r.Moment)
	}
}

// Several runs of one moment, the pre-push hook's ranges, are one: their
// steps in order, their seconds and calls added.
func TestJoin(t *testing.T) {
	all := Join([]*Result{
		{Seconds: 0.6, AgentCalls: 1, Steps: []Step{{Name: "committer", Seconds: 0.5}}},
		{Seconds: 1.2, Steps: []Step{{Name: "committer", Seconds: 0.4}, {Name: "documentalist", Seconds: 0.7}}},
	})
	if all.Seconds != 1.8 || all.AgentCalls != 1 || len(all.Steps) != 3 || all.Steps[2].Name != "documentalist" {
		t.Errorf("joined: %+v", all)
	}
	if got, want := all.Times(), "1.8 s in all — committer 0.5 s, committer 0.4 s, documentalist 0.7 s"; got != want {
		t.Errorf("Times() = %q, want %q", got, want)
	}
}

// The committer's agent is its rewrite.
func TestCommitterCallIsItsRewrite(t *testing.T) {
	s := Step{Name: "committer", Seconds: 14.6, Result: &engine.Result{Calls: []agent.Call{{Seconds: 14.1}}}}
	if got, want := s.timed(), "committer 15 s (rewrite 14 s)"; got != want {
		t.Errorf("timed() = %q, want %q", got, want)
	}
}

// Each moment by its event and run (ADR-0050, section 1).
func TestMomentOf(t *testing.T) {
	for _, c := range []struct {
		event  string
		inputs map[string]string
		only   bool
		asked  int
		want   string
	}{
		{"commit-msg", nil, false, 0, "commit"},
		{"commit-msg", nil, false, 1, "commit-rewrite"},
		{"pre-push", nil, false, 0, "push"},
		{"merge-request", nil, true, 0, "first-result"},
		{"merge-request", map[string]string{WarnInput: WarnOnly}, false, 3, "ready-review"},
		{"merge-request", map[string]string{WarnInput: WarnApart}, false, 3, "verdict"},
		{"merge-request", map[string]string{QueueInput: "true"}, true, 0, "queue"},
		{"merge-request", map[string]string{QueueInput: "true"}, false, 0, "queue"},
		{"merge-request", map[string]string{QueueInput: "false"}, false, 0, "verdict"},
		{"schedule", nil, false, 9, ""},
		{"release", nil, false, 1, ""},
	} {
		if got := momentOf(c.event, c.inputs, c.only, c.asked); got != c.want {
			t.Errorf("momentOf(%s, %v, %v, %d) = %q, want %q", c.event, c.inputs, c.only, c.asked, got, c.want)
		}
	}
}
