package tester

import (
	"slices"
	"testing"
)

func TestUncached(t *testing.T) {
	for _, c := range []struct {
		env  []string
		want string
	}{
		{[]string{"HOME=/h"}, "GOFLAGS=-count=1"},
		{[]string{"GOFLAGS=-mod=mod", "HOME=/h"}, "GOFLAGS=-mod=mod -count=1"},
		{[]string{"GOFLAGS=-count=2"}, "GOFLAGS=-count=2"},
	} {
		got := uncached(c.env)
		if n := len(slices.DeleteFunc(slices.Clone(got), func(e string) bool { return e != c.want })); n != 1 {
			t.Errorf("uncached(%q) = %q, want one %q", c.env, got, c.want)
		}
		if slices.Contains(c.env, "HOME=/h") && !slices.Contains(got, "HOME=/h") {
			t.Errorf("uncached(%q) lost HOME: %q", c.env, got)
		}
	}
}

func TestFrozenIn(t *testing.T) {
	r := record{Files: map[string]string{"calc/mul_test.go": "a1", "tests/test_x.py": "b1"}}
	ids := map[string]bool{
		"calc.TestMul": true, "calc.TestMulBig": true, "calc.TestMul/zero": true,
		"tests.test_x.test_y[1]": true, "tests.test_x.test_y_more": true, "rust::mod::test_z": true,
	}
	for _, c := range []struct {
		test string
		want []string
	}{
		{"calc.TestMulBig", []string{"calc.TestMulBig"}},                            // a report's id: itself
		{"calc/mul_test.go:TestMul", []string{"calc.TestMul", "calc.TestMul/zero"}}, // its subtest too
		{"tests/test_x.py:test_y", []string{"tests.test_x.test_y[1]"}},              // a parameter's mark
		{"calc/mul_test.go:TestDiv", nil},                                           // missing
		{"other/file.go:TestMul", nil},                                              // not a frozen file: no loose match
		{"rust::mod::test_q", nil},                                                  // a report's id, not there
	} {
		if got := r.frozenIn(c.test, ids); !slices.Equal(got, c.want) {
			t.Errorf("frozenIn(%q) = %q, want %q", c.test, got, c.want)
		}
	}
}

func TestWithTests(t *testing.T) {
	r := record{Tests: []string{"a.A.one", "b.B.two", "B.kt:three"}}
	got := r.withTests("B.kt", map[string]result{"b.B.two": failed, "base.old": passed}, []string{"b.B.twice"})
	if want := []string{"a.A.one", "b.B.twice"}; !slices.Equal(got.Tests, want) {
		t.Errorf("withTests = %q, want %q", got.Tests, want)
	}
}
