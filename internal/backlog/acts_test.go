package backlog

import "testing"

func TestSuggestFromActs(t *testing.T) {
	for _, c := range []struct {
		level        string
		done, undone int
		want         string
	}{
		{Normal, 9, 0, ""},            // too few
		{Normal, 10, 0, Enterprising}, // none undone
		{Normal, 10, 1, ""},           // one in ten: not more
		{Normal, 10, 2, Cautious},     // more than one in ten
		{Enterprising, 10, 0, ""},     // already the most
		{Enterprising, 20, 3, Normal}, // more than one in ten
		{Cautious, 30, 0, ""},         // the report's, from the proposals
		{"unknown", 30, 10, ""},       // a level not read: nothing
	} {
		if got, _ := SuggestFromActs(c.level, c.done, c.undone); got != c.want {
			t.Errorf("%s, %d done, %d undone: %q, want %q", c.level, c.done, c.undone, got, c.want)
		}
	}
}
