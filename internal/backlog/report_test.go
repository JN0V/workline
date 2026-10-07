package backlog

import (
	"strings"
	"testing"
)

func TestPlainLine(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"Refine #83: Need (draft), Validation (draft). Need and Validation can be drafted.", "Add Need and Validation to it (Need and Validation as drafts for you to correct)."},
		{"Refine #9: Verification, Scope. No code given.", "Add Verification and Scope to it."},
		{"Refine #9: Need (draft), Verification, Validation (draft), Scope.", "Add Need, Verification, Validation and Scope to it (Need and Validation as drafts for you to correct)."},
		{"Rename #9 to \"Rows\".", "Rename #9 to \"Rows\"."},
		{"Refine #9 without a colon", "Refine #9 without a colon"},
	} {
		if got := plainLine(c.in); got != c.want {
			t.Errorf("plainLine(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestWhatToDoPause(t *testing.T) {
	proposed := []Pending{{Issue: 9, Act: "rename", Line: "Rename #9.", Proposal: &Proposal{Do: "rename", Issue: 9}}}
	for _, c := range []struct {
		ignored, max int
		want, not    string
	}{
		{2, 3, "**Answer before the next run, or the role pauses**: nobody answered its last 2 runs (it pauses at 3).", "within"},
		{1, 3, "**Answer within 2 runs, or the role pauses**: nobody answered its last 1 run (it pauses at 3).", "before the next run"},
		{3, 3, "**Paused**: 3 runs in a row", "Answer"},
		{8, 0, "**Never paused** (ignored-runs-max: 0)", "Answer"},
		{0, 3, "**1 proposal to decide**", "pauses"},
	} {
		p := &Plan{Record: Record{Ignored: c.ignored, Proposed: proposed}, config: Config{IgnoredMax: c.max}}
		got := p.whatToDo(1, 0, 0, 0)
		if !strings.Contains(got, c.want) || strings.Contains(got, c.not) {
			t.Errorf("ignored %d of %d: %q, want %q without %q", c.ignored, c.max, got, c.want, c.not)
		}
	}
}

func TestWhatToDoSettle(t *testing.T) {
	p := &Plan{Record: Record{Proposed: []Pending{{Issue: 9, Act: "milestone", Line: "Move #9 out.", Proposal: &Proposal{Do: "milestone", Issue: 9}}}}, config: Config{IgnoredMax: 3}}
	got := p.whatToDo(0, 0, 0, 0)
	if !strings.Contains(got, "**1 issue for you to settle**") || strings.Contains(got, "Nothing waits on you") {
		t.Errorf("a slip with nowhere to go: %q", got)
	}
}
