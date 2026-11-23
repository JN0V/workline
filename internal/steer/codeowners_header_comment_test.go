package steer

import "testing"

// Comment-only header: given a section header `[Docs] # todo` and a bare
// pattern under it, when ownership is read, then the pattern has no owner.
func TestCodeownersCommentOnlyHeaderHasNoOwner(t *testing.T) {
	co := ParseCodeowners("CODEOWNERS", []byte("[Docs] # todo\n/docs/\n"))
	if owned, required := co.Owns("docs/a.md"); owned || required {
		t.Errorf("Owns: owned %v required %v, want neither", owned, required)
	}
	if co.Holds("docs/a.md") {
		t.Error("Holds: docs/a.md is held, want no owner")
	}
}
