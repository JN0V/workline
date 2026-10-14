package sample

import (
	"strings"
	"testing"
)

// A panel's vote says its majority, a 2–1 flagged; without one, why.
func TestPanelWords(t *testing.T) {
	for _, c := range []struct {
		p    *Panel
		want string
	}{
		{&Panel{For: "act", Votes: []string{"act", "act", "auditor"}}, "2–1 for the act (flagged)"},
		{&Panel{For: "auditor", Votes: []string{"auditor", "auditor", "auditor"}}, "3–0 for the auditor"},
		{&Panel{Votes: []string{"act", "auditor", ""}, Why: "a judge did not answer"}, "no majority: a judge did not answer"},
		{nil, "—"},
	} {
		if got := c.p.Words(); got != c.want {
			t.Errorf("Words = %q, want %q", got, c.want)
		}
	}
}

// A week's rows survive their hiding: a dash, `<` or `>` in them never
// ends the comment that hides them.
func TestHiddenRows(t *testing.T) {
	rs := []Row{{N: 1, Role: "product-owner", Kind: "close-duplicate", Ref: "#7", Verdict: "false", Panel: "2–1 for the act --> (flagged)"}}
	hidden := hiddenRows(rs)
	if strings.Count(hidden, "-->") != 1 || !strings.HasSuffix(hidden, " -->") {
		t.Fatalf("the rows end their comment early: %s", hidden)
	}
	got, ok := parseRows("text\n\n" + hidden + "\n\n<!-- workline:sample:2026-W41 -->")
	if !ok || len(got) != 1 || got[0] != rs[0] {
		t.Errorf("read back: %+v %v", got, ok)
	}
}

// The maintainer's answer to a row: a 👎's named rows contested, the rest
// agreed; a 👎 naming none leaves them unanswered; a 👍 agrees with all.
func TestWeekAnswer(t *testing.T) {
	down := week{Down: []string{"lead"}, Contested: map[int]bool{2: true}}
	if down.answer(2) != "contested" || down.answer(1) != "agreed" {
		t.Errorf("named rows: %s %s", down.answer(2), down.answer(1))
	}
	unnamed := week{Down: []string{"lead"}, Unnamed: true, Contested: map[int]bool{}}
	if unnamed.answer(1) != "unanswered" {
		t.Errorf("a 👎 naming no row: %s", unnamed.answer(1))
	}
	if up := (week{Up: []string{"lead"}, Contested: map[int]bool{}}); up.answer(3) != "agreed" {
		t.Errorf("a 👍: %s", up.answer(3))
	}
	if none := (week{Contested: map[int]bool{}}); none.answer(1) != "unanswered" {
		t.Errorf("no answer: %s", none.answer(1))
	}
}
