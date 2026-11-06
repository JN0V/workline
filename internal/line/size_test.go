package line

import (
	"reflect"
	"testing"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/engine"
)

// A run's fresh tokens are added by role; a run that called no agent, or
// whose calls said no tokens, adds none; the record read is not changed.
func TestSpendCountsFreshTokensByRole(t *testing.T) {
	was := map[string][]int{"tester": {7000}}
	steps := []Step{
		{Name: "product-owner", Result: &engine.Result{Calls: []agent.Call{{TokensIn: 3000, TokensCached: 1000, TokensOut: 500}, {TokensIn: 100}}}},
		{Name: "reviewer", Result: &engine.Result{}},
		{Name: "tester", Result: &engine.Result{Calls: []agent.Call{{Failed: "unreachable"}}}},
		{Name: "tester", Result: &engine.Result{Calls: []agent.Call{{TokensIn: 2000}}}},
	}
	got := spend(was, steps)
	want := map[string][]int{"product-owner": {2600}, "tester": {7000, 2000}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("spend = %v, want %v", got, want)
	}
	if !reflect.DeepEqual(was, map[string][]int{"tester": {7000}}) {
		t.Errorf("the record read was changed: %v", was)
	}
	if n := (Record{Spent: got}).Fresh(); n != 11600 {
		t.Errorf("Fresh = %d, want 11600", n)
	}
}

func TestTokensSaidShort(t *testing.T) {
	for n, want := range map[int]string{950: "950", 10500: "10.5k", 400000: "400k", 1600000: "1.6M", 156200: "156.2k"} {
		if got := Tokens(n); got != want {
			t.Errorf("Tokens(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestP85ByNearestRank(t *testing.T) {
	if got := p85([]int{10, 9, 8, 7, 6, 5, 4, 3, 2, 1}); got != 9 {
		t.Errorf("p85 of 1..10 = %d, want 9", got)
	}
	if got := p85([]int{42}); got != 42 {
		t.Errorf("p85 of one = %d, want 42", got)
	}
}
