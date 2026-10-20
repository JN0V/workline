package engine

import (
	"testing"

	"github.com/JN0V/workline/internal/verdict"
)

// A release held for its docs, as a real release run reports it: what
// holds it at `block`, what only informs with no level (#536).
func TestReleaseHeldReadsTheBlocksOnly(t *testing.T) {
	informs := []verdict.Finding{{Rule: "doc-too-long", Where: "CHANGELOG.md"}, {Rule: "links-not-checked", Where: "README.md"}}
	for _, c := range []struct {
		name string
		r    Result
		want bool
	}{
		{"docs due, beside what informs", Result{Status: verdict.Block, Findings: append([]verdict.Finding{{Rule: "suspect", Level: "block"}, {Rule: verdict.WaitsForNight, Level: "warn"}}, informs...)}, true},
		{"its fix elsewhere", Result{Status: verdict.Block, Findings: append([]verdict.Finding{{Rule: "fixed-elsewhere", Level: "block"}}, informs...)}, true},
		{"another block too", Result{Status: verdict.Block, Findings: []verdict.Finding{{Rule: "suspect", Level: "block"}, {Rule: "shallow-clone", Level: "block"}}}, false},
		{"a block that says nothing of it", Result{Status: verdict.Block, Findings: informs}, false},
		{"passed", Result{Status: verdict.Pass, Findings: []verdict.Finding{{Rule: "suspect", Level: "block"}}}, false},
	} {
		if got := ReleaseHeld(&c.r); got != c.want {
			t.Errorf("%s: ReleaseHeld = %v, want %v", c.name, got, c.want)
		}
	}
}

// The verdict's steps say which held: a release held is red only when a
// step blocked for something else.
func TestHeadVerdictHeldBySteps(t *testing.T) {
	findings := []verdict.Finding{{Rule: "doc-too-long"}, {Rule: "suspect", Level: "block"}, {Rule: verdict.WaitsForNight, Level: "warn"}}
	held := HeadVerdict{Status: verdict.Block, Findings: findings,
		Steps: []StepStatus{{Name: "committer", Status: verdict.Pass}, {Name: "documentalist", Status: verdict.Block, Held: true}}}
	if class, why := held.class(); class != verdictHeld || why != heldAtNight {
		t.Errorf("held: class %q (%s), want held, at night", class, why)
	}
	red := held
	red.Steps = []StepStatus{{Name: "committer", Status: verdict.Block}, {Name: "documentalist", Status: verdict.Block, Held: true}}
	if class, _ := red.class(); class != verdictRed {
		t.Errorf("a committer's block beside: class %q, want red", class)
	}
}
