package committer

import "testing"

func TestTrailers(t *testing.T) {
	msg := "fix: x\n\nBody line.\n\nRefs: AC-3\nCo-Authored-By: A <a@b.c>\n"
	got := Trailers(msg)
	if len(got) != 2 || got[0] != "Refs: AC-3" || got[1] != "Co-Authored-By: A <a@b.c>" {
		t.Fatalf("Trailers = %q", got)
	}
	if got := Trailers("fix: x\n\nNote: this is prose, not a trailer paragraph\nsecond line"); got != nil {
		t.Fatalf("a prose paragraph is not trailers: %q", got)
	}
	if got := Trailers("fix: x"); got != nil {
		t.Fatalf("no trailers: %q", got)
	}
}

func TestTrailersMustStayOnTheirOwnLines(t *testing.T) {
	glued := "fix: x\n\nBody.\nCo-Authored-By: A <a@b.c> Refs: BUG-64"
	for _, tr := range Trailers(glued) {
		if tr == "Co-Authored-By: A <a@b.c>" {
			t.Fatal("a trailer glued to another line must not count as kept")
		}
	}
}
