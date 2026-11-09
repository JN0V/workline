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
		// English prose is no mark (ADR-0065, change 2): only the line.
		"The check (`roles/documentalist`). The paths come from the error text, because no `init` code was given in this run.": false,
		"- `src/`: no code was read.": false,
	} {
		if got := Guessed(scope); got != want {
			t.Errorf("Guessed(%q) = %v, want %v", scope, got, want)
		}
	}
}

// The product owner's writing check (ADR-0065, change 2): each path a
// Scope writes is a file or folder files hold, a glob matching one, or
// on a line `New:`; a name, a dotted word, an Action are no path.
func TestUnfound(t *testing.T) {
	files := []string{"src/export/csv.go", "README.md"}
	for scope, want := range map[string][]string{
		"- `src/export/csv.go`: rows.\n- `src/export/`.\n- `src/export/*.go`.": nil,
		"- `src/import/xml.go`: the importer.":                                 {"src/import/xml.go"},
		"- `src/import/**`: the importers.":                                    {"src/import/**"},
		"- New: `src/import/xml.go`, the importer.":                            nil,
		"- **New:** `newmod/x.kt`.":                                            nil,
		"- `newmod/x.kt`.":                                                     {"newmod/x.kt"},
		"- `TracingViewModel`, `os.Getenv`, `csv.go`.":                         nil,
		"- `dorny/paths-filter`, `actions/checkout@v4`.":                       nil,
		"- `src/export/csv.go:12`, `src/export/csv.go:3-9`.":                   nil,
	} {
		if got := Unfound(scope, files); !slices.Equal(got, want) {
			t.Errorf("Unfound(%q) = %q, want %q", scope, got, want)
		}
	}
}

// An old Scope saying no code was given holds no other issue back
// (#998); a phrase going on past `code` is no such word.
func TestCodeUnread(t *testing.T) {
	for s, want := range map[string]bool{
		"- `x.go`: no `init` code was given in this run.": true,
		"No code was read for it.":                        true,
		"no other code was given a new name":              false,
		"- `x.go`: the parser.":                           false,
	} {
		if got := CodeUnread(s); got != want {
			t.Errorf("CodeUnread(%q) = %v, want %v", s, got, want)
		}
	}
}
