package tester

import "testing"

// Corpus grew: Given the project sets a growth limit of 20% and the last
// gardening saw 100 tests, When the corpus reaches 121 tests, Then the
// tester gardens it. At 119 tests it does not.
func TestGrowthDue(t *testing.T) {
	for _, c := range []struct {
		baseline, current, limit int
		want                     bool
	}{
		{100, 121, 20, true},
		{100, 119, 20, false},
		{100, 120, 20, false}, // strictly above the limit only
		{100, 100, 20, false},
		{100, 80, 20, false},
		{50, 61, 20, true},
	} {
		if got := GrowthDue(c.baseline, c.current, c.limit); got != c.want {
			t.Errorf("GrowthDue(%d, %d, %d) = %v, want %v", c.baseline, c.current, c.limit, got, c.want)
		}
	}
}
