package tests

import (
	"io/fs"
	"slices"
	"testing"

	"github.com/JN0V/workline"
	"github.com/JN0V/workline/internal/contract"

	// The compiled-in roles, the list the binary imports.
	_ "github.com/JN0V/workline/roles"
)

// Every shipped role with code/ finds its pre and post in the registry,
// and ships no pre nor post file: the engine runs the built-in code
// itself, a file beside it never runs (docs/spec/layout.md). The registry
// holds no role that has no code/ in roles/.
func TestEveryBuiltinStepIsInTheRegistry(t *testing.T) {
	coded := shippedRoles(t)
	for _, name := range coded {
		for _, step := range []string{"pre", "post"} {
			if _, ok := contract.Builtins().Step(name, step); !ok {
				t.Errorf("roles/%s/code: no built-in %s in the registry", name, step)
			}
			if _, err := fs.Stat(workline.Roles, "roles/"+name+"/"+step); err == nil {
				t.Errorf("roles/%s/%s is shipped beside the role's built-in code, which the engine runs instead: remove it", name, step)
			}
		}
	}
	if got := contract.Builtins().Names(); !slices.Equal(got, coded) {
		t.Errorf("the registry holds %v; the shipped roles with code/ are %v", got, coded)
	}
}
