package tests

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/JN0V/workline/internal/contract"

	// The compiled-in roles, the list the binary imports.
	_ "github.com/JN0V/workline/roles"
)

// Every shipped role with code/ finds its pre and post in the registry;
// the registry holds no role that has no code/ in roles/.
func TestEveryBuiltinStepIsInTheRegistry(t *testing.T) {
	coded := shippedRoles(t)
	for _, name := range coded {
		for _, step := range []string{"pre", "post"} {
			if _, ok := contract.Builtins().Step(name, step); !ok {
				t.Errorf("roles/%s/code: no built-in %s in the registry", name, step)
			}
		}
	}
	if got := contract.Builtins().Names(); !slices.Equal(got, coded) {
		t.Errorf("the registry holds %v; the shipped roles with code/ are %v", got, coded)
	}
}

// No shipped role holds a pre or post file, with code/ or without: its
// steps are its built-in code, else the shared ones (docs/spec/layout.md).
func TestNoShippedRoleHoldsAPreOrPostFile(t *testing.T) {
	for _, step := range []string{"pre", "post"} {
		found, err := filepath.Glob(filepath.Join("..", "roles", "*", step))
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range found {
			rel, _ := filepath.Rel("..", f)
			t.Errorf("%s: a shipped role holds no %s file; its built-in code or the shared step runs: remove it", filepath.ToSlash(rel), step)
		}
	}
}
