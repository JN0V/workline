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
// Build files (go.mod, package.json) change nothing with no module-files.
func TestPackagesWalkUpToTests(t *testing.T) {
	tree := t.TempDir()
	for _, f := range []string{
		"internal/a/a.go", "internal/a/a_test.go",
		"tests/conformance/runner_test.go", "tests/conformance/cases/dev/x.yaml",
		"docs/x.md", "cmd/tool/main.go", "go.mod", "web/package.json", "web/src/x.ts",
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
	got := Packages(tree, []string{"internal/a/a.go", "tests/conformance/cases/dev/x.yaml", "docs/x.md", "cmd/tool/main.go", "internal/a/gone/b.go", "web/src/x.ts"}, patterns, nil)
	if want := []string{"./internal/a", "./tests/conformance"}; !slices.Equal(got, want) {
		t.Errorf("Packages = %v, want %v", got, want)
	}
	if err := os.WriteFile(filepath.Join(tree, "root_test.go"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Packages(tree, []string{"docs/x.md"}, patterns, nil); !slices.Equal(got, []string{"."}) {
		t.Errorf("a file under a root holding tests: %v, want [.]", got)
	}
}

// With module-files, the folders are the build modules a change touches
// (#1176): the nearest folder above each file holding a build file; a
// file of the root module reaches every module, so none: the whole suite
// runs; a file with no module above adds none; a name may be a pattern
// (`*.csproj`).
func TestPackagesBuildModules(t *testing.T) {
	tree := t.TempDir()
	for _, f := range []string{
		"settings.gradle.kts", "build.gradle.kts",
		"core/session/build.gradle.kts", "core/session/src/main/kotlin/x/A.kt", "core/session/src/test/kotlin/x/ATest.kt",
		"core/timelapse/build.gradle", "core/timelapse/src/main/kotlin/x/B.kt",
		"tools/App/App.csproj", "tools/App/Program.cs",
		"docs/x.md",
	} {
		p := filepath.Join(tree, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	modules := []string{"build.gradle.kts", "build.gradle", "*.csproj"}
	patterns := []string{"**/src/test/**", "**/*Test.kt"}
	for _, tt := range []struct {
		files, want []string
	}{
		{[]string{"core/session/src/main/kotlin/x/A.kt"}, []string{"./core/session"}},
		{[]string{"core/session/src/test/kotlin/x/ATest.kt", "core/session/build.gradle.kts"}, []string{"./core/session"}},
		{[]string{"core/timelapse/src/main/kotlin/x/B.kt", "core/session/src/main/kotlin/x/gone/C.kt"}, []string{"./core/session", "./core/timelapse"}},
		{[]string{"tools/App/Program.cs"}, []string{"./tools/App"}},
		{[]string{"build.gradle.kts"}, nil},
		{[]string{"core/session/src/main/kotlin/x/A.kt", "build.gradle.kts"}, nil},
		{[]string{"docs/x.md", "core/timelapse/src/main/kotlin/x/B.kt"}, nil},
	} {
		if got := Packages(tree, tt.files, patterns, modules); !slices.Equal(got, tt.want) {
			t.Errorf("Packages(%v) = %v, want %v", tt.files, got, tt.want)
		}
	}
	if err := os.Remove(filepath.Join(tree, "build.gradle.kts")); err != nil {
		t.Fatal(err)
	}
	got := Packages(tree, []string{"docs/x.md", "core/timelapse/src/main/kotlin/x/B.kt"}, patterns, modules)
	if want := []string{"./core/timelapse"}; !slices.Equal(got, want) {
		t.Errorf("no root module: Packages = %v, want %v: a file with no module above adds none", got, want)
	}
}
