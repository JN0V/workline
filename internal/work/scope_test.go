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
// A Scope's word names one file, or by a pattern the files it matches
// (#801); a folder or a pattern matching nothing names none.
func TestFiles(t *testing.T) {
	files := []string{"lib.sh", "src/calc/add.sh", "src/calc/add.go", "src/calc/sub.sh", "src/calc/deep/x.sh", "tools/ops.sh"}
	for word, want := range map[string][]string{
		"src/calc/*.sh":   {"src/calc/add.sh", "src/calc/sub.sh"},
		"src/**/*.sh":     {"src/calc/add.sh", "src/calc/sub.sh", "src/calc/deep/x.sh"},
		"src/calc/a?d.go": {"src/calc/add.go"},
		"./src/calc/*.sh": {"src/calc/add.sh", "src/calc/sub.sh"},
		"lib.sh":          {"lib.sh"},
		"ops.sh":          {"tools/ops.sh"},
		"src/calc":        nil, // a folder: the caller reads it
		"src/calc/*.py":   nil,
		"../*.sh":         nil,
	} {
		if got := Files(word, files); !slices.Equal(got, want) {
			t.Errorf("Files(%q) = %v, want %v", word, got, want)
		}
	}
}

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

// A Scope marked as a guess starts a line with `Guessed:`, plain, bold
// or in a bullet (#970); the word elsewhere marks nothing.
func TestGuessed(t *testing.T) {
	for scope, want := range map[string]bool{
		"Guessed: no code was read.\n\n- `src/`.":   true,
		"- `src/`.\n\n**Guessed:** from the title.": true,
		"- guessed: from the folders.":              true,
		"- `src/`: the parser, guessed before.":     false,
		"- `src/lib.sh`: `add`.":                    false,
		// Before the mark, its own words said no code was given (#998).
		"The check (`roles/documentalist`). The paths come from the error text, because no `init` code was given in this run.": true,
		"- `src/`: the code of the parser was not read.":                                                                        true,
		"- `src/`: no code is read from the network.":                                                                           false,
		"- `src/`: the code was given a new name.":                                                                              false,
	} {
		if got := Guessed(scope); got != want {
			t.Errorf("Guessed(%q) = %v, want %v", scope, got, want)
		}
	}
}
