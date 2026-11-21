package tester

import "testing"

// Corpus grew: Given the project sets a growth limit of 20% and the last
// gardening saw 100 tests, When the corpus reaches 121 tests, Then the
// tester gardens it. At 119 tests it does not.
func TestGrowthDue(t *testing.T) {
	for _, c := range []struct {
		current int
		want    bool
	}{
		{121, true},
		{120, false}, // exactly the limit is not above it
		{119, false},
		{100, false},
	} {
		if got := GrowthDue(100, c.current, 20); got != c.want {
			t.Errorf("GrowthDue(100, %d, 20) = %v, want %v", c.current, got, c.want)
		}
	}
}
