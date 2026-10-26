package linters

import (
	"slices"
	"testing"
)

func TestRanNamesTheToolsACheckRuns(t *testing.T) {
	for command, want := range map[string][]string{
		"golangci-lint run --output.sarif.path={out}/lint.sarif ./...":                                  {"golangci-lint"},
		"ruff check --output-format sarif -o {out}/r.sarif . && npx eslint -f sarif -o {out}/e.sarif .": {"eslint", "ruff"},
		"go test ./...":      {"go test"},
		"echo golangci-lint": nil,
		"make lint":          nil,
	} {
		got := Ran(command)
		slices.Sort(got)
		if !slices.Equal(got, want) {
			t.Errorf("Ran(%q) = %q, want %q", command, got, want)
		}
	}
	if got := Configs("golangci-lint"); !slices.Contains(got, ".golangci.yml") {
		t.Errorf("golangci-lint's configs = %q", got)
	}
}

func TestSuppressedInItsLanguagesCommentsOnly(t *testing.T) {
	lint := []string{"golangci-lint", "ruff", "eslint", "clang-tidy"}
	for _, c := range []struct {
		file, text string
		tools      []string
		want       string
	}{
		{"a.go", "\tos.Remove(p) //nolint:errcheck", lint, "//nolint"},
		{"a.go", "\tos.Remove(p) // nolint", lint, "//nolint"},
		{"a.go", "\tos.Remove(p) //nolint:errcheck", []string{"ruff"}, ""}, // no check runs golangci-lint
		{"a.go", `	s := "nolint"`, lint, ""},
		{"README.md", "Add `//nolint` with a reason.", lint, ""},
		{"a.py", "import os  # noqa: F401", lint, "# noqa"},
		{"a.py", "import os  # NOQA", lint, "# noqa"},
		{"a.ts", "// eslint-disable-next-line no-console", lint, "eslint-disable"},
		{"a.js", "/* eslint-disable */", lint, "eslint-disable"},
		{"a.cpp", "int x; // NOLINT(readability-*)", lint, "NOLINT"},
		{"a.cpp", "int nolint_count;", lint, ""},
	} {
		s, ok := Suppressed(c.file, c.text, c.tools)
		if got := s.Name; !ok && got != "" || ok && got != c.want || !ok && c.want != "" {
			t.Errorf("Suppressed(%q, %q) = %q, %v; want %q", c.file, c.text, got, ok, c.want)
		}
	}
}
