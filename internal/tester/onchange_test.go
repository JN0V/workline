package tester

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// The folders a change touches that hold tests (#703): a Go file's own
// package, a conformance case's runner above it, none for a file with no
// test above it; a pattern naming a folder (`tests/**`) names no test file.
func TestPackagesWalkUpToTests(t *testing.T) {
	tree := t.TempDir()
	for _, f := range []string{
		"internal/a/a.go", "internal/a/a_test.go",
		"tests/conformance/runner_test.go", "tests/conformance/cases/dev/x.yaml",
		"docs/x.md", "cmd/tool/main.go",
	} {
		p := filepath.Join(tree, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	patterns := []string{"**/*_test.go", "tests/**"}
	got := Packages(tree, []string{"internal/a/a.go", "tests/conformance/cases/dev/x.yaml", "docs/x.md", "cmd/tool/main.go", "internal/a/gone/b.go"}, patterns)
	if want := []string{"./internal/a", "./tests/conformance"}; !slices.Equal(got, want) {
		t.Errorf("Packages = %v, want %v", got, want)
	}
	if err := os.WriteFile(filepath.Join(tree, "root_test.go"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Packages(tree, []string{"docs/x.md"}, patterns); !slices.Equal(got, []string{"."}) {
		t.Errorf("a file under a root holding tests: %v, want [.]", got)
	}
}
