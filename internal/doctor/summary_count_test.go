package doctor

import "testing"

func TestSummaryCounts(t *testing.T) {
	for n, want := range map[int]string{1: "1 problem", 2: "2 problems"} {
		if got := countNoun(n, "problem"); got != want {
			t.Fatalf("%q != %q", got, want)
		}
	}
}
