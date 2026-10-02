package sample

import (
	"fmt"
	"testing"
	"time"
)

func TestWeek(t *testing.T) {
	for in, want := range map[string]string{
		"2026-W01": "2025-12-29 2026-01-05",
		"2026-W40": "2026-09-28 2026-10-05",
		"2020-W53": "2020-12-28 2021-01-04",
	} {
		_, from, to, err := Week(in, time.Time{})
		if got := from.Format("2006-01-02") + " " + to.Format("2006-01-02"); err != nil || got != want {
			t.Errorf("Week(%s) = %s %v, want %s", in, got, err, want)
		}
	}
	for _, bad := range []string{"2026-40", "2026-W54", "2026-W00", "2025-W53"} {
		if _, _, _, err := Week(bad, time.Time{}); err == nil {
			t.Errorf("Week(%s): an error was due", bad)
		}
	}
	// None given: the last whole week, whatever the day.
	got, _, _, _ := Week("", time.Date(2026, 10, 2, 9, 0, 0, 0, time.UTC))
	if got != "2026-W39" {
		t.Errorf("the week before Friday 2026-10-02 is %s, want 2026-W39", got)
	}
}

func TestPick(t *testing.T) {
	var cands []candidate
	for i := range 25 {
		cands = append(cands, candidate{doc: fmt.Sprintf("docs/d%02d.md", i), commit: fmt.Sprintf("c%02d", i)})
	}
	for n, want := range map[int]int{0: 0, 1: 1, 10: 1, 11: 2, 25: 3} {
		if got := len(Pick("2026-W40", cands[:n])); got != want {
			t.Errorf("%d vouched for: %d read, want %d", n, got, want)
		}
	}
	// A rerun of the week picks the same; another week, most likely others.
	a, b := Pick("2026-W40", cands), Pick("2026-W40", append([]candidate(nil), cands...))
	if fmt.Sprint(a) != fmt.Sprint(b) {
		t.Errorf("the same week picked %v, then %v", a, b)
	}
	if c := Pick("2026-W41", cands); fmt.Sprint(a) == fmt.Sprint(c) {
		t.Logf("two weeks picked the same docs: possible, unlikely (%v)", c)
	}
}
