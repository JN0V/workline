package tester

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/intent"
)

func TestKeptIn(t *testing.T) {
	base := lines("class A {\n  fun t1() {}\n}\n")
	for _, c := range []struct {
		content string
		kept    bool
		line    string
	}{
		{"class A {\n  fun t1() {}\n\n  fun t2() {}\n}\n", true, ""},
		{"import x\nclass A {\n  fun t1() {}\n}\nfun t3() {}", true, ""},
		{"class A {\n  fun t1() { x }\n}\n", false, "  fun t1() {}"},
		{"class A {\n}\n", false, "  fun t1() {}"},
	} {
		_, l, kept := keptIn(base, lines(c.content))
		if kept != c.kept || l != c.line {
			t.Errorf("%q: kept %v %q, want %v %q", c.content, kept, l, c.kept, c.line)
		}
	}
}

func TestFrozenRunsKotlinClass(t *testing.T) {
	// A test added before the class's closing brace: the run is the new
	// test, its own closing brace in it, the class's left to the base.
	base := lines("class A {\n  @Test fun t1() {\n    x\n  }\n}\n")
	recorded := lines("class A {\n  @Test fun t1() {\n    x\n  }\n\n  @Test fun t2() {\n    y\n  }\n}\n")
	runs := frozenRuns(base, recorded)
	if len(runs) != 1 || runs[0].start != 5 || strings.Join(runs[0].lines, "|") != "|  @Test fun t2() {|    y|  }" {
		t.Fatalf("runs = %+v", runs)
	}
	if n := runs[0].name(); n != "line 6, `@Test fun t2() {`" {
		t.Errorf("name = %q", n)
	}
	// Another test added after it: kept; a line inserted in it: not.
	beside := lines("class A {\n  @Test fun t1() {\n    x\n  }\n\n  @Test fun t2() {\n    y\n  }\n\n  @Test fun t3() {}\n}\n")
	if _, ok := missingRun(runs, beside); !ok {
		t.Error("a test added beside the frozen one read as changing it")
	}
	inside := lines("class A {\n  @Test fun t1() {\n    x\n  }\n\n  @Test fun t2() {\n    return\n    y\n  }\n}\n")
	if _, ok := missingRun(runs, inside); ok {
		t.Error("a line inserted in the frozen test read as kept")
	}
	// A new file is one run, whole.
	if runs := frozenRuns(nil, recorded); len(runs) != 1 || len(runs[0].lines) != len(recorded) {
		t.Errorf("a new file's runs = %+v", runs)
	}
}

func TestBaseLineChangedDiff(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull) // no machine's hooks
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	dir := t.TempDir()
	for _, args := range [][]string{{"init", "-q", "-b", "main"}, {"-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "--allow-empty", "-m", "x"}} {
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatal(string(out))
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "a_test.go"), []byte("package a\n\nfunc TestOne(t *testing.T) {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "."}, {"-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "-m", "y"}} {
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatal(string(out))
		}
	}
	added := "--- a/a_test.go\n+++ b/a_test.go\n@@ -3,0 +4,2 @@\n+\n+func TestTwo(t *testing.T) {}\n"
	if l, ok := baseLineChanged(dir, "main", "a_test.go", intent.Intention{Kind: "patch", Value: added}); !ok {
		t.Errorf("an addition read as changing %s", l)
	}
	changed := "--- a/a_test.go\n+++ b/a_test.go\n@@ -3 +3 @@\n-func TestOne(t *testing.T) {}\n+func TestOne(t *testing.T) { t.Skip() }\n"
	if l, ok := baseLineChanged(dir, "main", "a_test.go", intent.Intention{Kind: "patch", Value: changed}); ok || l != "`func TestOne(t *testing.T) {}`" {
		t.Errorf("a changed line: %q %v", l, ok)
	}
	whole := map[string]any{"file": "a_test.go", "content": "package a\n\nfunc TestTwo(t *testing.T) {}\n"}
	if l, ok := baseLineChanged(dir, "main", "a_test.go", intent.Intention{Kind: "patch", Value: whole}); ok || l != "line 3, `func TestOne(t *testing.T) {}`" {
		t.Errorf("a whole file replacing a line: %q %v", l, ok)
	}
	if _, ok := baseLineChanged(dir, "main", "b_test.go", intent.Intention{Kind: "patch", Value: map[string]any{"file": "b_test.go", "content": "x"}}); !ok {
		t.Error("a new file read as changing the base")
	}
}
