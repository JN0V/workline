package doctor

import "testing"

func TestSummaryCounts(t *testing.T) {
	for n, want := range map[int]string{1: "1 problem", 2: "2 problems"} {
		if got := countNoun(n, "problem"); got != want {
			t.Fatalf("%q != %q", got, want)
		}
	}
}

func TestSummaryLine(t *testing.T) {
	for _, c := range []struct {
		errors, warns int
		want          string
	}{
		{0, 1, "1 warning: some checks do not run"},
		{0, 2, "2 warnings: some checks do not run"},
		{1, 1, "1 problem, 1 warning"},
		{2, 2, "2 problems, 2 warnings"},
		{0, 0, "everything workline uses is set up"},
	} {
		if got := summary(c.errors, c.warns); got != c.want {
			t.Fatalf("summary(%d, %d) = %q, want %q", c.errors, c.warns, got, c.want)
		}
	}
}
