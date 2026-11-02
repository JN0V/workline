package tester

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

// A pattern gives the code it matches as the code the Scope names, its
// tests given beside, once (#801); one matching tests alone gives them.
func TestWriteCodeOfAPattern(t *testing.T) {
	repo, _ := gitRepo(t, map[string]string{"add.go": "package calc\n", "add_test.go": "package calc\n", "notes.md": "x\n"})
	s := Settings{Paths: []string{"**/*_test.go"}}
	var b strings.Builder
	writeCode(&b, repo, "- `*.go`", s)
	task := b.String()
	_, after, _ := strings.Cut(task, "## The code the Scope names")
	code, tests, _ := strings.Cut(after, "## Tests already there")
	if !strings.Contains(code, "### `add.go`") || strings.Contains(code, "add_test.go") || strings.Contains(task, "notes.md") {
		t.Errorf("the code named:\n%s", task)
	}
	if !strings.Contains(tests, "### `add_test.go`") {
		t.Errorf("the tests beside:\n%s", task)
	}
	b.Reset()
	writeCode(&b, repo, "- `*_test.go`", s)
	if code, _, _ := strings.Cut(b.String(), "## Tests already there"); !strings.Contains(code, "## The code the Scope names\n\n### `add_test.go`") {
		t.Errorf("a pattern of tests alone names none:\n%s", b.String())
	}
}

// A function the Scope names past code-lines-max in a long file is given
// whole, the rest of the file named; a small file is given whole (#908).
func TestWriteCodeGivesAFunctionDeepInAFile(t *testing.T) {
	var long strings.Builder
	long.WriteString("package doc\n")
	for i := range 2940 {
		fmt.Fprintf(&long, "\n// f%d is filler.\nfunc f%d() {}\n", i, i)
	}
	long.WriteString("\n// personIssue is the issue a person reads.\nfunc personIssue(n int) string {\n\tif n > 0 {\n\t\treturn \"deep\"\n\t}\n\treturn \"\"\n}\n")
	repo, _ := gitRepo(t, map[string]string{"documentalist.go": long.String(), "small.go": "package doc\n\nfunc small() {}\n", "engine.go": "package doc\n\nfunc a() {}\n\nfunc sticky() {\n\tx := 1\n\t_ = x\n}\n"})
	n := strings.Count(long.String(), "\n")
	s := Settings{Paths: []string{"**/*_test.go"}, CodeLinesMax: 1500}
	var b strings.Builder
	writeCode(&b, repo, "- `personIssue` (`documentalist.go:9999`) builds it.\n- `engine.go:6` rewrites it.\n- `small.go`.", s)
	task := b.String()
	for _, want := range []string{"func personIssue(n int) string {\n\tif n > 0 {\n\t\treturn \"deep\"\n\t}\n\treturn \"\"\n}", "// personIssue is the issue a person reads.",
		fmt.Sprintf("### `documentalist.go` (%d lines: lines %d to %d given", n, n-6, n), "### `engine.go`\n", "func sticky() {", "### `small.go`\n\n````\npackage doc\n\nfunc small() {}\n````"} {
		if !strings.Contains(task, want) {
			t.Errorf("want %q in:\n%.3000s", want, task)
		}
	}
	if strings.Contains(task, "func f1()") || strings.Contains(task, "(cut: the first") {
		t.Errorf("the file's first lines given:\n%.3000s", task)
	}
}

func TestScenarios(t *testing.T) {
	got := Scenarios("Intro line.\n\n- **Old debt:** Given `main` with 140 findings, When a merge request\n  adds one, Then only that one blocks.\n- **Moved code**: Given a move, When the gate runs, Then nothing new.\n- Given no name, When read, Then dropped.\n")
	want := []Scenario{
		{Name: "Old debt", Text: "Given `main` with 140 findings, When a merge request adds one, Then only that one blocks."},
		{Name: "Moved code", Text: "Given a move, When the gate runs, Then nothing new."},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("scenario %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestNamesAScenarioWhole(t *testing.T) {
	for _, code := range []string{"func TestOldDebt(t *testing.T)", "def test_old_debt():", "// Scenario: Old debt",
		`t.Run("Old debt", func(t *testing.T) {`, "func TestGate_OldDebt(t *testing.T)"} {
		if !names(code, "Old debt") {
			t.Errorf("%q does not name the scenario", code)
		}
	}
	for _, code := range []string{"func TestOldDebtGone(t *testing.T)", "def test_old_debt_gone():", "// Scenario: Old debt gone",
		"// the old debt is kept", `msg := "old debt and more"`, "func TestNewOldDebt(t *testing.T)"} {
		if names(code, "Old debt") {
			t.Errorf("%q names the scenario, though only a longer name or prose holds it", code)
		}
	}
	if names("// Scenario: !!", "!!") {
		t.Error("a name of no word is named")
	}
}

func TestClaimTakesTheLongestName(t *testing.T) {
	sc := []Scenario{{Name: "Minus one"}, {Name: "Minus one hundred"}}
	if i := claim(words("Minus one hundred: a person reads it"), sc, hasPrefix); i != 1 {
		t.Errorf("note claimed by %d, want 1", i)
	}
	if i := claim(words("test_minus_one"), sc, hasSuffix); i != 0 {
		t.Errorf("test claimed by %d, want 0", i)
	}
}

// A one-letter word in a camel-cased name: `TestSpaceInAName` names
// "Space in a name", `TestHTTPServer` "HTTP server".
func TestWordsOfCapitals(t *testing.T) {
	if !names("func TestSpaceInAName(t *testing.T)", "Space in a name") {
		t.Error("TestSpaceInAName does not name \"Space in a name\"")
	}
	if got := words("TestHTTPServer"); !slices.Equal(got, []string{"test", "http", "server"}) {
		t.Errorf("words(TestHTTPServer) = %q", got)
	}
}
