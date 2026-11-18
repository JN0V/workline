package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sampleTree writes the given files, by their path under a new temporary
// root, and returns that root.
func sampleTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, body := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// goFile is a Go source file of the given package importing one path.
func goFile(pkg, imported string) string {
	return "package " + pkg + "\n\nimport _ \"" + imported + "\"\n"
}

const (
	judgeCode  = "github.com/JN0V/workline/roles/judge/code"
	testerCode = "github.com/JN0V/workline/roles/tester/code"
	registry   = "github.com/JN0V/workline/roles"
)

// Given a sample tree where internal/engine/x.go imports roles/judge/code,
// When the check scans it, Then it fails, naming the file and the import.
func TestImportsCoreImportsARole(t *testing.T) {
	root := sampleTree(t, map[string]string{
		"internal/engine/x.go": goFile("engine", judgeCode),
	})
	found, err := checkImports(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 {
		t.Fatalf("want one violation, got %q", found)
	}
	for _, want := range []string{"internal/engine/x.go", "roles/judge/code"} {
		if !strings.Contains(found[0], want) {
			t.Errorf("the violation %q does not name %q", found[0], want)
		}
	}
}

// Given a sample tree where roles/developer/code/a.go imports
// roles/tester/code, When the check scans it, Then it fails, naming both
// roles.
func TestImportsARoleImportsAnother(t *testing.T) {
	root := sampleTree(t, map[string]string{
		"roles/developer/code/a.go": goFile("code", testerCode),
	})
	found, err := checkImports(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 {
		t.Fatalf("want one violation, got %q", found)
	}
	for _, want := range []string{"roles/developer/code/a.go", "developer", "tester", "roles/tester/code"} {
		if !strings.Contains(found[0], want) {
			t.Errorf("the violation %q does not name %q", found[0], want)
		}
	}
}

// Given the first import only in internal/engine/x_test.go, When the check
// scans it, Then it fails.
func TestImportsTestFilesToo(t *testing.T) {
	root := sampleTree(t, map[string]string{
		"internal/engine/x_test.go": goFile("engine", judgeCode),
	})
	found, err := checkImports(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || !strings.Contains(found[0], "internal/engine/x_test.go") {
		t.Errorf("want one violation naming internal/engine/x_test.go, got %q", found)
	}
}

// Given a sample tree where cmd/workline/main.go imports the roles'
// registry, When the check scans it, Then it passes: only internal/ and
// the roles are checked.
func TestImportsOutsideInternalIsNotChecked(t *testing.T) {
	root := sampleTree(t, map[string]string{
		"cmd/workline/main.go": goFile("main", registry),
	})
	found, err := checkImports(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 0 {
		t.Errorf("cmd/ may import the roles, got %q", found)
	}
}

// Given workline's own tree, When go test ./tests/... runs, Then the check
// passes.
func TestImportsHoldOnTheRepository(t *testing.T) {
	found, err := checkImports("..")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range found {
		t.Error(line)
	}
}
