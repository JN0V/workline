package contract

import (
	"slices"
	"testing"
)

// A fixture role, joined to a test registry, is found by name, each step
// on its own.
func TestAFixtureRoleIsFoundByName(t *testing.T) {
	r := NewRegistry()
	r.Join("fixture", Role{Pre: func(string, string) int { return 10 }})
	pre, ok := r.Step("fixture", "pre")
	if !ok || pre("run", "repo") != 10 {
		t.Fatalf("pre = %v, %v", pre, ok)
	}
	if _, ok := r.Step("fixture", "post"); ok {
		t.Error("a step the role does not offer is found")
	}
	if _, ok := r.Step("fixture", "draw"); ok {
		t.Error("a step that is not pre nor post is found")
	}
	if _, ok := r.Step("absent", "pre"); ok {
		t.Error("a role never joined is found")
	}
	if got := r.Names(); !slices.Equal(got, []string{"fixture"}) {
		t.Errorf("Names = %v", got)
	}
	if len(Builtins().Names()) != 0 {
		t.Error("a test registry leaks into the shipped one")
	}
}

func TestARoleJoinsOnce(t *testing.T) {
	r := NewRegistry()
	r.Join("x", Role{})
	defer func() {
		if recover() == nil {
			t.Error("a role joined twice is taken")
		}
	}()
	r.Join("x", Role{})
}

// The code the engine merges is asked of the role that offers it, found
// by the registry; none offered, none found.
func TestTheCodeIsAskedOfTheRoleOfferingIt(t *testing.T) {
	r := NewRegistry()
	r.Join("other", Role{})
	if c, n := r.Code(); c != nil || n != "" {
		t.Fatalf("Code = %v, %q with no role offering it", c, n)
	}
	r.Join("coder", Role{Code: &Code{Guarded: func(string, string) func(string) string {
		return func(p string) string { return "gate-file" }
	}}})
	c, n := r.Code()
	if c == nil || n != "coder" || c.Guarded("repo", "at")("ci.yml") != "gate-file" {
		t.Errorf("Code = %v, %q", c, n)
	}
}
