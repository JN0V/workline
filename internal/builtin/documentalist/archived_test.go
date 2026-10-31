package documentalist

import "testing"

func TestArchivedReadsTheHeader(t *testing.T) {
	for doc, want := range map[string]bool{
		"---\narchived: 2026-10-04\n---\n# Old\n": true,
		"---\narchived: false\n---\n# Old\n":      false,
		"---\nsources: [a.go]\n---\n# Doc\n":      false,
		"# No header\n":                           false,
		"---\narchived: true\n---\n# Old\n":       true,
		"---\narchived: 0\n---\n# Old\n":          false,
		"---\narchived: off\n---\n# Old\n":        false,
		"---\narchived: n\n---\n# Old\n":          false,
		"---\narchived: {a: b}\n---\n# Old\n":     false,
		"---\narchived: [x]\n---\n# Old\n":        false,
	} {
		if got := archived(doc); got != want {
			t.Errorf("archived(%q) = %v, want %v", doc, got, want)
		}
	}
}
