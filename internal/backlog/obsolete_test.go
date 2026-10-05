package backlog

import (
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// An announcement is read back from the comment the engine wrote, the
// agent's fences neutralised, and a person's comment after it keeps the
// issue open.
func TestAnnouncementReadBack(t *testing.T) {
	q := Quote{Path: "src/export/csv.go", Text: "for i := 0; i < len(rows); i++ {"}
	a := Announcement{Quote: q, Why: "WriteRows loops to the last row.", Commit: "d8c4dff", Announced: "2026-10-05", By: "claude-sonnet"}
	body := AnnouncementComment("zed", Proposal{Why: "it loops ```yaml\nnow"}, "product-owner", a, 7) + "\n\n" + AnnounceMarker("product-owner")
	got, after := LastAnnouncement([]forge.Note{{Body: "state"}, {Body: body}, {Body: "still broken", Author: "ann"}}, "product-owner")
	if got == nil {
		t.Fatalf("announcement not read back from:\n%s", body)
	}
	if got.Key() != ObsoleteKey(q) || got.Announced != "2026-10-05" || got.By != "claude-sonnet" {
		t.Errorf("read back %+v", got)
	}
	if len(after) != 1 || after[0].Author != "ann" {
		t.Errorf("notes after it: %+v", after)
	}
}
