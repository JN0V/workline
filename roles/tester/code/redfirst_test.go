package tester

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// The fast path passes only what it is sure of: each error a missing name
// in a test of the change, the base lacking it, the issue naming it
// verbatim; with no issue, the change's code naming it. Anything else is
// the judge's, never a no of its own (ADR-0065).
func TestFast(t *testing.T) {
	r := &redFirst{issue: "Add `lostCount` to the statistics.", code: func() string { return "" },
		inBase: map[string]bool{"lostCount": false, "stubTwo": false, "count": true}}
	missing := func(name string) buildError {
		return buildError{file: "T.kt", msg: "Unresolved reference '" + name + "'.", inTest: true, missing: true, name: name}
	}
	for _, c := range []struct {
		about string
		errs  []buildError
		want  bool
	}{
		{"the issue's name, the base lacking it", []buildError{missing("lostCount"), missing("Stats::lostCount")}, true},
		{"a name the issue never names", []buildError{missing("lostCount"), missing("stubTwo")}, false},
		{"a name the base holds", []buildError{missing("count")}, false},
		{"another error", []buildError{missing("lostCount"), {file: "T.kt", msg: "Too many arguments", inTest: true}}, false},
		{"a missing name read with no name", []buildError{{file: "T.kt", msg: "cannot find symbol", inTest: true, missing: true}}, false},
		{"an error outside the tests", []buildError{{file: "Calc.kt", msg: "Unresolved reference 'lostCount'.", missing: false, name: "lostCount"}}, false},
		{"no error", nil, false},
	} {
		if got := r.fast(c.errs); got != c.want {
			t.Errorf("%s: fast = %v, want %v", c.about, got, c.want)
		}
	}
	noIssue := &redFirst{code: func() string { return "func Mul(a, b int) int { return a * b }" }, inBase: map[string]bool{"Mul": false}}
	if !noIssue.fast([]buildError{{file: "m_test.go", msg: "undefined: Mul", inTest: true, missing: true, name: "Mul"}}) {
		t.Error("no issue known: a name the change's code adds is red")
	}
}

// The judge reads the names the errors give, each said held in the base
// or not; the words they quote count, a sentence does not.
func TestFactsNames(t *testing.T) {
	r := &redFirst{repo: ".", issue: "Restore the count.", code: func() string { return "" },
		test: func(string) string { return "check(restore(1, 2))" }, tests: []string{"T.kt"},
		inBase: map[string]bool{"restore": true, "nextIndex": false}}
	errs := []buildError{
		{file: "T.kt", line: "e: T.kt:1:7 Too many arguments for 'restore'.", msg: "Too many arguments for 'restore'.", inTest: true},
		{file: "T.kt", line: "e: T.kt:2:1 'nextIndex' overrides nothing.", msg: "'nextIndex' overrides nothing.", inTest: true},
	}
	got := r.facts(errs, "e: T.kt:1:7 Too many arguments for 'restore'.\nBUILD FAILED\n", nil)
	for _, want := range []string{"Restore the count.", "### `T.kt`", "check(restore(1, 2))", "e: T.kt:2:1 'nextIndex' overrides nothing.",
		"- `restore`: held somewhere in the base's files", "- `nextIndex`: in none of the base's files", "BUILD FAILED"} {
		if !strings.Contains(got, want) {
			t.Errorf("facts lack %q:\n%s", want, got)
		}
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
	// A line quoted as an error is not quoted again in the tail.
	if got := errorLines("> Task :a\ne: a/T.kt:1:2 Boom\n", rows, 5); got != "its errors: e: a/T.kt:1:2 Boom; its output ends: > Task :a" {
		t.Errorf("twice: %q", got)
	}
	// One line of tens of KB, and many: each cut, and the whole.
	long := "e: a/T.kt:1:2 " + strings.Repeat("x", 50000)
	got := errorLines(strings.Repeat(long+"\n", 20)+strings.Repeat("y", 50000)+"\n", rows, 10)
	if n := utf8.RuneCountInString(got); n > quoteMax {
		t.Errorf("%d runes, past %d", n, quoteMax)
	}
	if !strings.HasPrefix(got, "its errors: e: a/T.kt:1:2 xxx") || strings.Contains(got, strings.Repeat("x", quoteLine)) || !strings.Contains(got, "x… / ") {
		t.Errorf("each line not cut at %d runes: %.400q", quoteLine, got)
	}
}
