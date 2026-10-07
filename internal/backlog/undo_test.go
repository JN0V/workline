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
