package steer

import "testing"

// Comment only: given a rule `/x.go # todo`, when the file x.go is checked,
// then it has no owner.
func TestCodeownersCommentOnlyHasNoOwner(t *testing.T) {
	co := ParseCodeowners("CODEOWNERS", []byte("/x.go # todo\n"))
	if owned, required := co.Owns("x.go"); owned || required {
		t.Errorf("Owns: owned %v required %v, want neither", owned, required)
	}
	if co.Holds("x.go") {
		t.Error("Holds: x.go is held, want no owner")
	}
}

// Cancelling rule: given an owned rule followed by `/x.go # todo`, when x.go
// is checked, then it is unowned.
func TestCodeownersCommentOnlyRuleCancelsOwnedRule(t *testing.T) {
	co := ParseCodeowners("CODEOWNERS", []byte("/x.go @alice\n/x.go # todo\n"))
	if owned, required := co.Owns("x.go"); owned || required {
		t.Errorf("Owns: owned %v required %v, want neither", owned, required)
	}
	if co.Holds("x.go") {
		t.Error("Holds: x.go is held, want unowned")
	}
}
