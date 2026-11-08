package tester

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// A report pattern with `**` names files folders deep, as Gradle and
// Maven write one per module (#1175); `.git` is never walked; a path or
// a pattern one folder deep reads as before.
func TestReportFilesSpanFolders(t *testing.T) {
	tree := t.TempDir()
	for _, f := range []string{
		"app/build/test-results/testDebugUnitTest/TEST-a.xml",
		"core/session/build/test-results/testDebugUnitTest/TEST-b.xml",
		"core/session/build/test-results/testDebugUnitTest/binary/output.bin",
		".git/build/test-results/x/TEST-c.xml",
		"target/surefire-reports/TEST-d.xml",
		"junit.xml",
	} {
		p := filepath.Join(tree, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	rel := func(pattern string) []string {
		var out []string
		for _, f := range reportFiles(tree, pattern) {
			r, _ := filepath.Rel(tree, f)
			out = append(out, filepath.ToSlash(r))
		}
		return out
	}
	for pattern, want := range map[string][]string{
		"**/build/test-results/**/*.xml": {"app/build/test-results/testDebugUnitTest/TEST-a.xml", "core/session/build/test-results/testDebugUnitTest/TEST-b.xml"},
		"core/**/TEST-*.xml":             {"core/session/build/test-results/testDebugUnitTest/TEST-b.xml"},
		"**/surefire-reports/*.xml":      {"target/surefire-reports/TEST-d.xml"},
		"target/surefire-reports/*.xml":  {"target/surefire-reports/TEST-d.xml"},
		"junit.xml":                      {"junit.xml"},
		"nowhere/**/*.xml":               nil,
	} {
		if got := rel(pattern); !slices.Equal(got, want) {
			t.Errorf("reportFiles(%q) = %v, want %v", pattern, got, want)
		}
	}
}
