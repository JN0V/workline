package backlog

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// A split stays watched while one of its parts may still be closed as not
// planned; once none is left open — closed, or gone from the forge
// (deleted, moved) — there is nothing left to undo.
func TestSplitWatchedWhileAPartIsOpen(t *testing.T) {
	state := func(split string) string {
		return "```yaml\nsources: []\nconfirmed: 0000000\nsplit: " + split + "\n```\n\n<!-- workline:sticky=product-owner/state -->"
	}
	path := filepath.Join(t.TempDir(), "forge.json")
	data := `{"issues": [
	  {"id": 1, "title": "Parent, a part open", "body": "", "labels": [], "comments": [` + quote(state("[11, 12]")) + `]},
	  {"id": 2, "title": "Parent, a part gone", "body": "", "labels": [], "comments": [` + quote(state("[13, 14]")) + `]},
	  {"id": 11, "title": "Open part", "body": "", "labels": []},
	  {"id": 12, "title": "Done part", "body": "", "labels": [], "closed": true, "reason": "completed"},
	  {"id": 13, "title": "Done part", "body": "", "labels": [], "closed": true, "reason": "completed"}
	], "merge-requests": []}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	f := &forge.Fake{Path: path}
	open := map[int]forge.Issue{1: {ID: 1}, 2: {ID: 2}}
	standing, undone, _, err := findUndone(f, "product-owner", open, []Done{{Issue: 1, Act: "split"}, {Issue: 2, Act: "split"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(undone) != 0 {
		t.Errorf("undone: %v", undone)
	}
	if len(standing) != 1 || standing[0].Issue != 1 {
		t.Errorf("still watched: %v, want the split of #1 alone", standing)
	}
}

// A closed parent's split links stay the role's while a child that waits
// is open; a state that does not read keeps them watched, never dropped;
// a parent gone from the forge is let go, not a failed run.
func TestClosedParentKeepsItsLinks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "forge.json")
	data := `{"issues": [
	  {"id": 1, "title": "Parent closed", "body": "", "labels": [], "closed": true, "comments": [` + quote("```yaml\nsources: []\nconfirmed: 0000000\nsplit: [11, 12]\nafter: {12: [11]}\n```\n\n<!-- workline:sticky=product-owner/state -->") + `]},
	  {"id": 2, "title": "Parent closed, its state broken", "body": "", "labels": [], "closed": true, "comments": [` + quote("```yaml\nsplit: [\n```\n\n<!-- workline:sticky=product-owner/state -->") + `]},
	  {"id": 11, "title": "Done part", "body": "", "labels": [], "closed": true},
	  {"id": 12, "title": "Waiting part", "body": "", "labels": []}
	], "merge-requests": []}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	f := &forge.Fake{Path: path}
	open := map[int]forge.Issue{12: {ID: 12}}
	standing, _, own, err := findUndone(f, "product-owner", open, []Done{{Issue: 1, Act: "split"}, {Issue: 2, Act: "split"}, {Issue: 3, Act: "split"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(own[12]) != 1 || own[12][0] != 11 {
		t.Errorf("own = %v, want #12's link to #11", own)
	}
	if len(standing) != 2 || standing[0].Issue != 1 || standing[1].Issue != 2 {
		t.Errorf("still watched: %v, want #1 and #2, #3 (gone) let go", standing)
	}
}

// An open split parent whose state does not read is still watched: never
// a panic, never dropped.
func TestOpenParentUnreadStateWatched(t *testing.T) {
	path := filepath.Join(t.TempDir(), "forge.json")
	data := `{"issues": [
	  {"id": 1, "title": "Parent, its state broken", "body": "", "labels": [], "comments": [` + quote("```yaml\nsplit: [\n```\n\n<!-- workline:sticky=product-owner/state -->") + `]}
	], "merge-requests": []}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	f := &forge.Fake{Path: path}
	standing, undone, _, err := findUndone(f, "product-owner", map[int]forge.Issue{1: {ID: 1}}, []Done{{Issue: 1, Act: "split"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(undone) != 0 || len(standing) != 1 || standing[0].Issue != 1 {
		t.Errorf("standing %v, undone %v: want the split of #1 still watched", standing, undone)
	}
}

func quote(s string) string {
	b := []byte{'"'}
	for _, r := range s {
		switch r {
		case '"', '\\':
			b = append(b, '\\', byte(r))
		case '\n':
			b = append(b, '\\', 'n')
		default:
			b = append(b, string(r)...)
		}
	}
	return string(append(b, '"'))
}
