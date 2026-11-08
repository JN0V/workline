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
