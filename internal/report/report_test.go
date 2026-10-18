package report

import (
	"os"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/verdict"
)

// On a merge request, code scanning gets neither the documentalist's, the
// reviewer's nor the gates' findings, which workline posts there itself
// (#412, ADR-0049); on any other event, it gets them all.
func TestForCodeScanning(t *testing.T) {
	items := []Item{{Role: "documentalist"}, {Role: "reviewer"}, {Role: "committer"}, {Role: "gate:secrets"}}
	var kept []string
	for _, i := range ForCodeScanning("merge-request", items) {
		kept = append(kept, i.Role)
	}
	if got := strings.Join(kept, ","); got != "committer" {
		t.Errorf("merge-request keeps %s, want committer", got)
	}
	if got := ForCodeScanning("schedule", items); len(got) != len(items) {
		t.Errorf("schedule keeps %d of %d", len(got), len(items))
	}
}

// Annotations read the files at the head judged: a heading's line, a file
// that head does not hold left out; a long message cut, and beyond the
// cap, the rest counted, not written: the run's findings are data.
func TestAnnotations(t *testing.T) {
	files := map[string]string{"docs/a.md": "---\n---\n# A\n\n## Token refresh\n"}
	read := func(path string) ([]byte, error) {
		if s, ok := files[path]; ok {
			return []byte(s), nil
		}
		return nil, os.ErrNotExist
	}
	notes, left := Annotations(read, []Item{
		{Role: "documentalist", Status: verdict.Pass, Finding: verdict.Finding{Rule: "suspect", Where: "docs/a.md#token-refresh", Message: "changed"}},
		{Role: "documentalist", Status: verdict.Pass, Finding: verdict.Finding{Rule: "suspect", Where: "docs/gone.md", Message: "gone"}},
		{Role: "documentalist", Status: verdict.Block, Finding: verdict.Finding{Rule: "count-off", Where: "docs/a.md", Message: strings.Repeat("é", 3000)}},
	})
	if left != 0 || len(notes) != 2 {
		t.Fatalf("%d notes, %d left: %+v", len(notes), left, notes)
	}
	if n := notes[0]; n.Path != "docs/a.md" || n.Line != 5 || n.Level != "notice" || n.Title != "documentalist/suspect" {
		t.Errorf("first: %+v, want docs/a.md:5, a notice", n)
	}
	if n := notes[1]; n.Level != "failure" || len(n.Message) > maxAnnotationLen+len("…") || !strings.HasSuffix(n.Message, "…") {
		t.Errorf("second: level %s, %d bytes; want a failure, cut", n.Level, len(n.Message))
	}
	var many []Item
	for range maxAnnotations + 3 {
		many = append(many, Item{Role: "documentalist", Finding: verdict.Finding{Rule: "suspect", Where: "docs/a.md"}})
	}
	if notes, left := Annotations(read, many); len(notes) != maxAnnotations || left != 3 {
		t.Errorf("%d notes, %d left; want %d and 3", len(notes), left, maxAnnotations)
	}
}
