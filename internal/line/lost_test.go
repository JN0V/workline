package line

import (
	"testing"

	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/verdict"
)

// A run is lost only when the developer's own run left no verdict
// (#953): another engine error, once it may have pushed, or a handoff's
// run, is not a loss, never woken again for it.
func TestLostOnlyWithNoVerdict(t *testing.T) {
	noVerdict := &engine.Result{Status: verdict.Block, NoVerdict: true,
		Findings: []verdict.Finding{{Rule: "engine-error", Message: "post wrote no readable verdict: gone"}}}
	other := &engine.Result{Status: verdict.Block,
		Findings: []verdict.Finding{{Rule: "engine-error", Message: "opening the pull request: 502"}}}
	cases := []struct {
		name  string
		steps []Step
		want  string
	}{
		{"no verdict", []Step{{Name: "developer", Result: noVerdict}}, "post wrote no readable verdict: gone"},
		{"another error", []Step{{Name: "developer", Result: other}}, ""},
		{"a handoff's", []Step{{Name: "developer", Result: &engine.Result{Status: verdict.Pass}}, {Name: "x (handoff)", Result: noVerdict}}, ""},
		{"no step", nil, ""},
	}
	for _, c := range cases {
		if got := lost(c.steps); got != c.want {
			t.Errorf("%s: lost = %q, want %q", c.name, got, c.want)
		}
	}
}
