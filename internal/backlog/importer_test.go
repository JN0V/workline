package backlog

import "testing"

// A roadmap's entries: an id in a heading of level 2 to 4, the text down to
// the next heading of its level or above, the state its heading says.
func TestParseRoadmap(t *testing.T) {
	text := "# R\n\n## SEC-12 — OTA: no check [MEDIUM] — **DONE (2026-09-27)**\n\nDone.\n\n" +
		"## Priority 3\n\n### BUG-4 — NTP: use-after-free [HIGH]\n\nThe pointer.\n\n#### Fix\n\nCopy it.\n\n" +
		"### CI-7: `local_ci.sh` counts lines [LOW] — **NEW (2026-09-20)**\n\nA count.\n\n## Archive\n\nNot an entry.\n"
	got := ParseRoadmap(text, DoneWords)
	if len(got) != 3 {
		t.Fatalf("got %d entries: %+v", len(got), got)
	}
	if !got[0].Done || got[0].Title != "OTA: no check [MEDIUM]" {
		t.Errorf("SEC-12 = %+v", got[0])
	}
	if got[1].Done || got[1].ID != "BUG-4" || got[1].Body != "The pointer.\n\n#### Fix\n\nCopy it." || got[1].Line != 9 {
		t.Errorf("BUG-4 = %+v", got[1])
	}
	if got[2].Done || got[2].Title != "`local_ci.sh` counts lines [LOW]" || got[2].Body != "A count." {
		t.Errorf("CI-7 = %+v", got[2])
	}
}
