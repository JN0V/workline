package engine

import "testing"

func TestAskAgain(t *testing.T) {
	for _, c := range []struct {
		promoteAfter int
		want         string // tier of each attempt after the first, until it stops
	}{
		{0, ""},
		{1, "standard"},
		{2, "light standard"},
		{3, "light light standard"},
	} {
		got, tier := "", "light"
		for failures := 1; ; failures++ {
			retry, next := askAgain(failures, c.promoteAfter, tier)
			if !retry {
				break
			}
			tier = next
			got += " " + tier
		}
		if got != " "+c.want && !(c.want == "" && got == "") {
			t.Errorf("promote-after %d: attempts after the first on %q, want %q", c.promoteAfter, got, c.want)
		}
	}
}
