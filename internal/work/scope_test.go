package work

import (
	"slices"
	"testing"
)

// A Scope names a file with its line, as the product owner's writing
// rules ask (`parts.go:943`): the word read is the file, the line off.
func TestScopeWordsTakeTheLineOff(t *testing.T) {
	got := ScopeWords("- `uncovered` (`internal/b/parts.go:943`), `parts.go:10-20`, `owner/repo:x.go`, `a.go`.")
	want := []string{"uncovered", "internal/b/parts.go", "parts.go", "owner/repo:x.go", "a.go"}
	if !slices.Equal(got, want) {
		t.Errorf("ScopeWords = %q, want %q", got, want)
	}
}

// A word names a repository file whole, or by its end when one file
// only ends so; never several, never a folder, never outside.
func TestInRepo(t *testing.T) {
	files := []string{"README.md", "docs/README.md", "internal/b/parts.go", "internal/b/parts_test.go", "x/a.go", "y/a.go"}
	for word, want := range map[string]string{
		"internal/b/parts.go":   "internal/b/parts.go",
		"parts.go":              "internal/b/parts.go",
		"b/parts.go":            "internal/b/parts.go",
		"./internal/b/parts.go": "internal/b/parts.go",
		"README.md":             "README.md", // whole before any end
		"a.go":                  "",          // two files end so
		"internal/b":            "",          // a folder is no file
		"arts.go":               "",          // an end is a whole name
		"uncovered":             "",
		"../parts.go":           "",
		".":                     "",
	} {
		if got := InRepo(word, files); got != want {
			t.Errorf("InRepo(%q) = %q, want %q", word, got, want)
		}
	}
}
