package engine

import (
	"testing"

	"github.com/JN0V/workline/internal/intent"
)

// An order carrying an effort becomes two acts, each its own kind; an
// effort said by name reads as its level; an order with no effort, or an
// effort with no priority, stays one act (ADR-0040).
func TestSplitEffort(t *testing.T) {
	in := []intent.Intention{
		{Kind: "order", Value: map[string]any{"issue": 9, "priority": 2, "effort": 1, "why": "w"}},
		{Kind: "order", Value: map[string]any{"issue": 10, "effort": "High", "why": "w"}},
		{Kind: "order", Value: map[string]any{"issue": 11, "priority": 3, "why": "w"}},
		{Kind: "effort", Value: map[string]any{"issue": 12, "effort": 2, "ticked": "ann", "own": true}},
		{Kind: "refine", Value: map[string]any{"issue": 13}},
	}
	out := splitEffort(in)
	var kinds []string
	for _, i := range out {
		kinds = append(kinds, i.Kind)
	}
	if want := []string{"order", "effort", "effort", "order", "effort", "refine"}; len(kinds) != len(want) {
		t.Fatalf("kinds = %v, want %v", kinds, want)
	} else {
		for i := range want {
			if kinds[i] != want[i] {
				t.Fatalf("kinds = %v, want %v", kinds, want)
			}
		}
	}
	order := out[0].Value.(map[string]any)
	if _, has := order["effort"]; has || order["priority"] != 2 {
		t.Errorf("the order keeps its priority, not the effort: %v", order)
	}
	if e := out[1].Value.(map[string]any); e["effort"] != 1 || e["issue"] != 9 || e["why"] != "w" {
		t.Errorf("effort = %v", e)
	}
	if e := out[2].Value.(map[string]any); e["effort"] != 3 || e["issue"] != 10 {
		t.Errorf("an effort by name: %v", e)
	}
	// What reads as no level is refused as one, never cut nor guessed.
	for _, v := range []any{1.9, "big", true} {
		got := splitEffort([]intent.Intention{{Kind: "order", Value: map[string]any{"issue": 9, "effort": v}}})
		if e := got[len(got)-1].Value.(map[string]any); e["effort"] != 0 {
			t.Errorf("effort %v read as %v, want none", v, e["effort"])
		}
	}
}
