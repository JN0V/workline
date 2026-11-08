package tester

import (
	"slices"
	"testing"
)

// The names an error gives: a quoted signature's own name, a lone quoted
// word's last part; quoting none, the words the test calls (#1192).
func TestNamesOf(t *testing.T) {
	test := "if Mul(2, 3) != Add(6) {"
	for _, c := range []struct {
		msg  string
		want []string
	}{
		{"Too many arguments for 'fun restoreFromExisting(existingCount: Int): Unit'.", []string{"restoreFromExisting"}},
		{"'nextIndex' overrides nothing.", []string{"nextIndex"}},
		{"Argument type mismatch: actual type is 'kotlin.String', but 'kotlin.Int' was expected.", []string{"String", "Int"}},
		{"not enough arguments in call to Add", []string{"Add"}},
		{`cannot use "Kitchen" (untyped string constant) as int value in variable declaration`, nil},
		{"Cannot infer type for this parameter. Please specify it explicitly.", nil},
	} {
		if got := namesOf(c.msg, test); !slices.Equal(got, c.want) {
			t.Errorf("namesOf(%q) = %q, want %q", c.msg, got, c.want)
		}
	}
}

// A missing name the issue never names follows from another when the
// test reaches it as a member or names a type it does not call; a helper
// it calls is its own (#1111).
func TestKnockOn(t *testing.T) {
	test := "val g = CompositionGrid(9, GridType.THIRDS)\ncheck(g.vertical == 3)\ncheck(stubTwo() == 2)\nFakeBroker()"
	for name, want := range map[string]bool{"GridType": true, "vertical": true, "stubTwo": false, "FakeBroker": false} {
		if got := knockOn(name, test); got != want {
			t.Errorf("knockOn(%q) = %v, want %v", name, got, want)
		}
	}
}

// A name the test declares is its own, an override not (#1192 review).
func TestDeclaredIn(t *testing.T) {
	test := "class FakeStore : Store {\n    override fun nextIndex() = 5\n}\nfun helper() = 1\nfunc stubTwo() int { return 2 }\n"
	for name, want := range map[string]bool{"FakeStore": true, "helper": true, "stubTwo": true, "nextIndex": false, "Store": false} {
		if got := declaredIn(name, test); got != want {
			t.Errorf("declaredIn(%q) = %v, want %v", name, got, want)
		}
	}
}

// In a file holding a red error, an error quoting nothing is a knock-on
// only when it says a type could not be inferred; a parse error, or K1's
// unquoted type mismatch, is the test's own.
func TestSortKnockOnsNarrow(t *testing.T) {
	r := &redFirst{issue: "Add a `grid` of thirds.", code: func() string { return "" },
		test: func(string) string { return "val g = grid(9)\n" }}
	errs := []buildError{
		{file: "T.kt", line: "e: T.kt:11:20 Unresolved reference 'grid'.", msg: "Unresolved reference 'grid'.", inTest: true, missing: true, name: "grid"},
		{file: "T.kt", line: "e: T.kt:12:38 Cannot infer type for this parameter.", msg: "Cannot infer type for this parameter.", inTest: true},
		{file: "T.kt", line: "e: T.kt:14:9 Type mismatch: inferred type is String but Int was expected", msg: "Type mismatch: inferred type is String but Int was expected", inTest: true},
		{file: "T.kt", line: "e: T.kt:20:1 Expecting an element", msg: "Expecting an element", inTest: true},
	}
	broken, unnamed, _ := r.sort(errs)
	if len(unnamed) != 0 || len(broken) != 2 || broken[0].line != errs[2].line || broken[1].line != errs[3].line {
		t.Errorf("broken %v, unnamed %v: want the type mismatch and the parse error", broken, unnamed)
	}
}

// What a failed run says went wrong: the rows' lines, else the lines
// saying an error, never one saying there is none; then its last lines.
func TestErrorLines(t *testing.T) {
	rows := []BuildErrors{{Error: `^e: (?P<file>.+?\.kt):\d+:\d+ (?P<message>.+)$`}}
	if got := errorLines("> Task :a\ne: a/T.kt:1:2 Boom\nBUILD FAILED in 8s\n", rows, 1); got != "its errors: e: a/T.kt:1:2 Boom; its output ends: BUILD FAILED in 8s" {
		t.Errorf("rows: %q", got)
	}
	if got := errorLines("No error found\nerror: kapt failed\nBUILD FAILED\n", rows, 1); got != "its error lines: error: kapt failed; its output ends: BUILD FAILED" {
		t.Errorf("error lines: %q", got)
	}
	if got := errorLines("No error found\nBUILD FAILED\n", rows, 10); got != "no error line in its output; its output ends: No error found / BUILD FAILED" {
		t.Errorf("tail: %q", got)
	}
}
