package codepick

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/testdef"
)

// repo commits files in a new repository.
func repo(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, text := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{{"init", "-q", "-b", "main"}, {"add", "."}, {"-c", "user.name=t", "-c", "user.email=t@example.invalid", "commit", "-qm", "code"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return dir
}

func pick(t *testing.T, dir string, a Ask) (Pick, string) {
	t.Helper()
	p, err := New(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	pk := p.Pick(a)
	var b strings.Builder
	pk.Write(&b, Style{Numbered: true})
	return pk, b.String()
}

func filler(n int, format string) string {
	var b strings.Builder
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, format+"\n", i)
	}
	return b.String()
}

// A name defined in many files, one of them past a long file's first
// lines: each definition is given, a test's copy counted, the callers
// after (#1057).
func TestEveryDefinitionOfAName(t *testing.T) {
	var files = map[string]string{
		"a/ui.h":            "class A {\n  String version() const { return \"1.0\"; }\n};\n",
		"b/ui.h":            filler(300, "// b %d") + "class B {\n  String version() const {\n    return \"1.1\";\n  }\n};\n",
		"c/ui.cpp":          "String C::version() const {\n  return \"1.2\";\n}\n",
		"app/show.cpp":      "void show(A& a) {\n  print(a.version());\n}\n",
		"test/test_ui.cpp":  "String Fake::version() const { return \"0\"; }\n",
		"test/test_ui2.cpp": "String Fake2::version() const { return \"0\"; }\n",
		"test/test_ui3.cpp": "String Fake3::version() const { return \"0\"; }\n",
	}
	_, task := pick(t, repo(t, files), Ask{Text: "Hardcoded fallbacks in `version()` drift", Budget: 100})
	for _, want := range []string{`return "1.0"`, `return "1.1";`, `String C::version() const {`, "print(a.version());",
		"`version` (6 definitions, 1 of them not given, most in tests, examples or tools)"} {
		if !strings.Contains(task, want) {
			t.Errorf("want %q in:\n%s", want, task)
		}
	}
	if strings.Contains(task, "// b 1\n") || strings.Contains(task, "   1  // b 1") {
		t.Errorf("a long file's first lines given:\n%s", task)
	}
}

// A line named after a long file gives it, its neighbours, not the file;
// a span gives its middle too; a line past the end, nothing; a piece the
// budget runs out in is said cut (#1063).
func TestLinesNamedAfterAFile(t *testing.T) {
	dir := repo(t, map[string]string{"docs/big.md": filler(7576, "item %d"), "run.sh": filler(200, "x=%d")})
	pk, task := pick(t, dir, Ask{Text: "Done: `docs/big.md:6353`, and docs/big.md, lines 9000 to 9001.", Budget: 1500})
	if !strings.Contains(task, "6353  item 6353") || !strings.Contains(task, "6338  item 6338") || strings.Contains(task, "   1  item 1\n") {
		t.Errorf("the line named and its neighbours:\n%s", task)
	}
	if n := pk.Files[0].GivenLines(); n != 2*around+1 {
		t.Errorf("%d lines given, want %d", n, 2*around+1)
	}
	pk, _ = pick(t, dir, Ask{Text: "`run.sh:50-150`", Budget: 150})
	if g := pk.Files[0].Given; len(g) != 1 || g[0].From != 35 || g[0].To != 165 {
		t.Errorf("a span's pieces = %+v, want 35 to 165", g)
	}
	_, task = pick(t, dir, Ask{Text: "`run.sh:50-150`", Budget: 30, BudgetName: "code-lines-max"})
	if !strings.Contains(task, "(cut, past code-lines-max: what it names goes on to line ") {
		t.Errorf("cut not said:\n%s", task)
	}
}

// A name that defines no function, a static member defined past a long
// header's first lines: the lines naming it are given (#1056); a C++
// method out of its class is given whole (#1064).
func TestNamesPastTheFirstLines(t *testing.T) {
	header := "class W {\n  static char buf_[64];\n  void reset();\n};\n" + filler(880, "// filler %d") +
		"inline void W::reset() {\n  n = 0;\n}\n\nchar W::buf_[64];\n"
	_, task := pick(t, repo(t, map[string]string{"web/w.h": header}), Ask{Text: "`buf_` and `reset()` in `web/w.h`", Budget: 100})
	for _, want := range []string{"char W::buf_[64];", "  static char buf_[64];", "inline void W::reset() {\n 886    n = 0;\n 887  }"} {
		if !strings.Contains(task, want) {
			t.Errorf("want %q in:\n%s", want, task)
		}
	}
}

// A Python function a text names is given, and the module its file
// imports named as a path.
func TestAPythonFunction(t *testing.T) {
	src := "from tools.semver import parse\n\n" + filler(200, "# %d") + "def check(root):\n    errors = []\n    return errors\n\n\ndef other():\n    pass\n"
	pk, task := pick(t, repo(t, map[string]string{"tools/check.py": src, "tools/semver.py": "def parse(v):\n    return v\n"}), Ask{Text: "`check()` misses the root", Budget: 50})
	if !strings.Contains(task, "def check(root):\n 204      errors = []") || strings.Contains(task, "def other") {
		t.Errorf("the function named:\n%s", task)
	}
	if !slices.Contains(pk.Linked, "`tools/semver.py` (imported)") {
		t.Errorf("imports: %v", pk.Linked)
	}
}

// Code that fills the budget keeps a share for the nearest test, its
// module's, not another's (#1059, #1047).
func TestRoomForATest(t *testing.T) {
	dir := repo(t, map[string]string{
		"Console/include/Console.h":                  filler(400, "int f%d();"),
		"Console/test/test_console_x/test_console.c": filler(30, "// test %d"),
		"Core/test/t_a/test_a.c":                     "// core\n",
	})
	pk, _ := pick(t, dir, Ask{Text: "`Console/include/Console.h`", Tests: testdef.From([]string{"**/test/**"}, nil), TestRoom: true, Budget: 100})
	tf := pk.Test()
	if tf == nil || tf.Path != "Console/test/test_console_x/test_console.c" || tf.GivenLines() != 20 || pk.Kept != 20 {
		t.Fatalf("the test: %+v, kept %d", tf, pk.Kept)
	}
	if n := pk.Files[0].GivenLines(); n != 80 {
		t.Errorf("code given: %d lines, want 80", n)
	}
}

// The tests nearest a file: its folder's, else its module's, the one
// holding its name first (#1047).
func TestNearestTests(t *testing.T) {
	tests := []string{"Core/test/t_a/test_a.cpp", "Console/test/test_capture/test_capture.cpp", "Console/test/test_console_x/test_console_x.cpp"}
	got := NearestTests(tests, []string{"**/test/**"}, []string{"Console/include/Console.h"})
	if want := []string{"Console/test/test_console_x/test_console_x.cpp", "Console/test/test_capture/test_capture.cpp"}; !slices.Equal(got, want) {
		t.Errorf("kin tests: %v, want %v", got, want)
	}
	if got := NearestTests(tests, []string{"**/test/**"}, []string{"top.h"}); len(got) != 0 {
		t.Errorf("a file at the root has no kin: %v", got)
	}
}

// A span given for one issue is not given again for the next of the same
// task.
func TestOnceATask(t *testing.T) {
	p, err := New(repo(t, map[string]string{"a.go": "package a\n\nfunc Run() {}\n"}), "")
	if err != nil {
		t.Fatal(err)
	}
	first := p.Pick(Ask{Text: "`Run()`", Budget: 100})
	second := p.Pick(Ask{Text: "`Run()` again", Budget: 100})
	if len(first.Paths()) != 1 || len(second.Paths()) != 0 {
		t.Errorf("first %v, second %v", first.Paths(), second.Paths())
	}
}

// The file an issue was imported from is where it was written, never its
// code (#1063); a name used everywhere gives a few places, a third of the
// budget at most, the rest counted.
func TestImportedFileAndUsesCapped(t *testing.T) {
	var calls strings.Builder
	for i := range 40 {
		fmt.Fprintf(&calls, "func use%d() {\n\tconnect()\n}\n\n", i)
	}
	dir := repo(t, map[string]string{"docs/roadmap.md": filler(7000, "item %d"), "net/net.go": "package net\n\nfunc connect() {}\n", "net/uses.go": "package net\n\n" + calls.String()})
	text := "`connect()` fails.\n\nOpened from `docs/roadmap.md`, lines 6353 to 6353 by the product-owner role.\n\n<!-- workline:import=docs/roadmap.md:2e2aab9c6fc4 -->"
	pk, task := pick(t, dir, Ask{Text: text, Budget: 60})
	if slices.Contains(pk.Named(), "docs/roadmap.md") || !strings.Contains(task, "func connect() {}") {
		t.Errorf("the imported file given, or the code missed:\n%s", task)
	}
	if n := strings.Count(task, "\tconnect()"); n == 0 || n > 6 || !strings.Contains(task, "`connect` (") {
		t.Errorf("%d places naming it given:\n%s", n, task)
	}
}

// The functions a definition calls, the longest first: three one-line
// helpers called before it do not push out the one that matters (#1129).
func TestCalleesTheLongestFirst(t *testing.T) {
	dir := repo(t, map[string]string{"led.c": `void logA() { puts("a"); }
void logB() { puts("b"); }
void logC() { puts("c"); }
void updateEffects() {
    if (solid) {
        write(pin, stored);
        return;
    }
}
void tick() {
    logA(); logB(); logC();
    updateEffects();
}
`})
	pk, out := pick(t, dir, Ask{Text: "`tick()` lights a disabled LED", Budget: 200})
	var why []string
	for _, f := range pk.Files {
		for _, g := range f.Given {
			why = append(why, g.Why)
		}
	}
	if w := strings.Join(why, "; "); !strings.Contains(w, "`updateEffects`, called by `tick`") || strings.Count(w, "called by") != 3 {
		t.Errorf("not updateEffects() and two helpers: %s\n%s", w, out)
	}
}

// Two functions of the name called in the file: which is called is not
// known, neither is given (#1129).
func TestCalleeOfTwoDefinitionsNotGiven(t *testing.T) {
	dir := repo(t, map[string]string{"a.go": `package a

type A struct{}
type B struct{}

func (A) String() string {
	return "a"
}

func (B) String() string {
	return "b"
}

func Render(a A) string {
	return a.String()
}
`})
	_, out := pick(t, dir, Ask{Text: "`Render` prints the wrong text", Budget: 200})
	if strings.Contains(out, `return "a"`) || strings.Contains(out, `return "b"`) {
		t.Errorf("a String() given, which one is called unknown:\n%s", out)
	}
}

// The routes a text writes, a file's path not one (#1152).
func TestRoutes(t *testing.T) {
	got := routes("POST `/api/components/enable` with `name=x`; see docs/x.md and /etc/a.conf, then /api/components/enable again.")
	if !slices.Equal(got, []string{"/api/components/enable"}) {
		t.Fatalf("routes: %v", got)
	}
}

// A bound in bytes: long lines given past it are cut, said (#1152).
func TestBytesBound(t *testing.T) {
	dir := repo(t, map[string]string{"a.go": "package a\n\n// Fetch is long.\nfunc Fetch() {\n" + filler(200, "\t_ = \"a line of some forty bytes, number %03d\"") + "}\n"})
	pk, _ := pick(t, dir, Ask{Text: "`Fetch` fails", Budget: 1500, Bytes: 2000})
	if n := pk.Bytes(); n > 2000 || n < 1500 {
		t.Fatalf("given %d bytes, want 1500 to 2000", n)
	}
	if pk.Files[0].Cut == 0 {
		t.Fatalf("the cut is not said: %+v", pk.Files[0].Given)
	}
}

// A module's folder named: its build file, then its code folders below,
// its tests and resources aside, each folder named sharing the bound;
// a folder asked for alike (#1410).
func TestModuleFolder(t *testing.T) {
	dir := repo(t, map[string]string{
		"core/x/build.gradle.kts":                            "plugins { kotlin(\"jvm\") }\n",
		"core/x/src/main/kotlin/a/x/Store.kt":                "package a.x\n\nclass Store {\n    fun save() = 1\n}\n",
		"core/x/src/main/res/values/strings.xml":             "<resources/>\n",
		"core/x/src/test/kotlin/a/x/StoreTest.kt":            "package a.x\n\nclass StoreTest\n",
		"feature/x/build.gradle.kts":                         "plugins { kotlin(\"jvm\") }\n",
		"feature/x/src/main/kotlin/a/x/feature/FileStore.kt": "package a.x.feature\n\nclass FileStore\n",
	})
	pk, _ := pick(t, dir, Ask{Text: "the counter runs on", Paths: []string{"core/x", "feature/x"}, Budget: 600})
	want := []string{"core/x/build.gradle.kts", "feature/x/build.gradle.kts", "core/x/src/main/kotlin/a/x/Store.kt", "feature/x/src/main/kotlin/a/x/feature/FileStore.kt"}
	if got := pk.Paths(); !slices.Equal(got, want) {
		t.Errorf("a module's folders named give %v, want %v", got, want)
	}
	pk, _ = pick(t, dir, Ask{Text: "the counter runs on", Needs: []string{"core/x"}, Budget: 600})
	if got := pk.Paths(); !slices.Contains(got, "core/x/src/main/kotlin/a/x/Store.kt") {
		t.Errorf("a module's folder asked for gives %v, not its code", got)
	}
}

// A function a long file defines, named, is given past the file's
// first lines and its bytes bound (#1332).
func TestNamedFunctionInALongFile(t *testing.T) {
	dir := repo(t, map[string]string{"pkg/spec.py": filler(1000, "X_%04d = 'a constant of some forty bytes'") +
		"\n\ndef round_trip(rows):\n    return list(rows)\n\n" + filler(300, "Y_%04d = 'a constant of some forty bytes'")})
	_, out := pick(t, dir, Ask{Text: "`round_trip` loses a row", Paths: []string{"pkg/spec.py"}, Budget: 600, Bytes: 16000})
	if !strings.Contains(out, "def round_trip(rows):") {
		t.Errorf("the function named is not given:\n%.2000s", out)
	}
}
