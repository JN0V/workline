package tester

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// Given the tester setting maps `roles/*/cases/**` to the package
// `./tests/conformance`, When a change touches `roles/reviewer/cases/x.yaml`,
// Then the folders to run are `./tests/conformance`, whatever tests lie
// beside the role's code; a file the mapping does not name is still found
// walking up from it.
func TestPackagesMapRoleCasesToTheConformanceSuite(t *testing.T) {
	tree := t.TempDir()
	for _, f := range []string{
		"tests/conformance/runner_test.go",
		"roles/reviewer/cases/x.yaml", "roles/reviewer/code/a.go", "roles/reviewer/code/a_test.go",
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
	mapping := map[string]string{"roles/*/cases/**": "./tests/conformance"}
	for _, tt := range []struct {
		files, want []string
	}{
		{[]string{"roles/reviewer/cases/x.yaml"}, []string{"./tests/conformance"}},
		{[]string{"roles/reviewer/cases/x.yaml", "roles/reviewer/code/a.go"}, []string{"./roles/reviewer/code", "./tests/conformance"}},
	} {
		if got := Packages(tree, tt.files, patterns, nil, mapping); !slices.Equal(got, tt.want) {
			t.Errorf("Packages(%v) = %v, want %v", tt.files, got, tt.want)
		}
	}
}

// Given the same mapping, When the green proof reads the head's reports and
// a case frozen as `<file>:<role>/<case>` is named by the conformance
// package's TestConformance subtest, Then the case is found there, and
// not under another package's TestConformance.
func TestFrozenInRoleCaseIsASubtestOfTheConformanceSuite(t *testing.T) {
	file := "roles/reviewer/cases/x.yaml"
	r := record{Files: map[string]string{file: "a1"}}
	id := "github.com/JN0V/workline/tests/conformance.TestConformance/reviewer/x"
	ids := map[string]bool{
		id: true,
		"github.com/JN0V/workline/internal/line.TestConformance/reviewer/x": true,
		"github.com/JN0V/workline/tests/conformance.TestOther/reviewer/x":   true,
	}
	if got, want := r.frozenIn(file+":reviewer/x", ids), []string{id}; !slices.Equal(got, want) {
		t.Errorf("frozenIn(%q) = %q, want only %q", file+":reviewer/x", got, want)
	}
}
