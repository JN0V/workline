package contract

import "testing"

// The project's tests are asked of the role offering them, one role at
// most; none offered, none found.
func TestTheTestsAreAskedOfTheRoleOfferingThem(t *testing.T) {
	r := NewRegistry()
	if tt, n := r.Tests(); tt != nil || n != "" {
		t.Fatalf("Tests = %v, %q with no role offering them", tt, n)
	}
	r.Join("runner", Role{Tests: &Tests{}})
	if tt, n := r.Tests(); tt == nil || n != "runner" {
		t.Errorf("Tests = %v, %q", tt, n)
	}
	defer func() {
		if recover() == nil {
			t.Error("a second role offering the tests is taken")
		}
	}()
	r.Join("other", Role{Tests: &Tests{}})
}
