package developer

import "testing"

// A tests-lens finding on code goes to the tester, its Draft open; one
// on a frozen test, on the developer's own test, of another lens, or
// with no Draft stays the developer's (#1265).
func TestSplitTests(t *testing.T) {
	b := Bounds{Issue: 3, Draft: 1, Head: "abc", Frozen: map[string]int{"tests/test_mul.sh": 3}}
	asked := []Asked{
		{Key: "tests|lib.sh|a", Lens: "tests", Where: "lib.sh:2", Holds: "warns"},
		{Key: "tests|tests/test_mul.sh|b", Lens: "tests", Where: "tests/test_mul.sh:4", Holds: "warns"},
		{Key: "tests|tests/test_neg.sh|c", Lens: "tests", Where: "tests/test_neg.sh:1", Holds: "blocks"},
		{Key: "correctness|lib.sh|d", Lens: "correctness", Where: "lib.sh:2", Holds: "blocks"},
		{Key: "person|1", Where: "@ann", Holds: "a person"},
	}
	mine, theirs := splitTests(asked, b)
	if len(theirs) != 1 || theirs[0].Key != "tests|lib.sh|a" {
		t.Errorf("the tester's: %v", theirs)
	}
	if len(mine) != 4 {
		t.Errorf("the developer's: %v", mine)
	}
	resumed := asked[0]
	resumed.Waited = true
	if mine, theirs := splitTests([]Asked{resumed}, b); len(theirs) != 0 || len(mine) != 1 {
		t.Errorf("asked again from the record, the developer's still: %v, %v", mine, theirs)
	}
	grouped := asked[0]
	grouped.Text = "Important: a — no test. (also found on this line — correctness: nil deref on empty amount (verified))"
	if mine, theirs := splitTests([]Asked{grouped}, b); len(theirs) != 0 || len(mine) != 1 {
		t.Errorf("a code finding grouped under it, the developer's: %v, %v", mine, theirs)
	}
	grouped.Text = "Important: a — no test. (also found on this line — tests: no test of zero)"
	if _, theirs := splitTests([]Asked{grouped}, b); len(theirs) != 1 {
		t.Errorf("test findings alone grouped, the tester's: %v", theirs)
	}
	b.Draft = 0
	if _, theirs := splitTests(asked, b); len(theirs) != 0 {
		t.Errorf("no Draft, none the tester's: %v", theirs)
	}
}
