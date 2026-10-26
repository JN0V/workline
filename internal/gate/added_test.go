package gate

import (
	"os"
	"path/filepath"
	"testing"
)

// Two findings alike in a file are told apart by their rank, and neither
// fingerprint changes while they only move: no line number is in it.
func TestStampTellsAlikeApartAndKeepsThemWhenMoved(t *testing.T) {
	stamped := func(text string, at ...int) []string {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		var rs []*result
		for _, n := range at {
			rs = append(rs, &result{tool: "golangci-lint", rule: "errcheck", text: "not checked", uri: "a.go", line: n})
		}
		stamp(rs, &lines{root: dir})
		var out []string
		for _, r := range rs {
			out = append(out, r.posted)
		}
		return out
	}
	before := stamped("package a\n\tos.Remove(p)\n\tos.Remove(p)\n", 2, 3)
	if before[0] == before[1] || before[0] == "" {
		t.Fatalf("two alike, one fingerprint: %v", before)
	}
	moved := stamped("package a\n\n\n\n\tos.Remove(p)\n\tos.Remove(p)\n", 5, 6)
	if moved[0] != before[0] || moved[1] != before[1] {
		t.Errorf("moved, the fingerprints changed: %v, were %v", moved, before)
	}
	edited := stamped("package a\n\tos.Remove(q)\n\tos.Remove(p)\n", 2, 3)
	if edited[0] == before[0] || edited[1] != before[0] {
		t.Errorf("one line edited: %v, were %v; want the first new, the second the first's", edited, before)
	}
}
