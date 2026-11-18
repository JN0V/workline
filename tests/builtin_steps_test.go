package tests

import (
	"io/fs"
	"regexp"
	"slices"
	"testing"

	"github.com/JN0V/workline"
	"github.com/JN0V/workline/internal/contract"

	// The compiled-in roles, as cmd/workline lists them.
	_ "github.com/JN0V/workline/internal/builtin/committer"
	_ "github.com/JN0V/workline/internal/builtin/documentalist"
	_ "github.com/JN0V/workline/internal/builtin/inspector"
	_ "github.com/JN0V/workline/internal/builtin/productowner"
	_ "github.com/JN0V/workline/internal/builtin/reviewer"
	_ "github.com/JN0V/workline/internal/developer"
	_ "github.com/JN0V/workline/internal/tester"
)

// Every shipped step that calls `workline builtin <role> <step>` finds
// that step in the registry, and the registry holds no role no step calls.
func TestEveryBuiltinStepIsInTheRegistry(t *testing.T) {
	call := regexp.MustCompile(`builtin ([a-z-]+) (pre|post)\b`)
	var called []string
	err := fs.WalkDir(workline.Roles, "roles", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || (d.Name() != "pre" && d.Name() != "post") {
			return err
		}
		data, err := fs.ReadFile(workline.Roles, path)
		if err != nil {
			return err
		}
		for _, m := range call.FindAllStringSubmatch(string(data), -1) {
			if _, ok := contract.Builtins().Step(m[1], m[2]); !ok {
				t.Errorf("%s calls `builtin %s %s`: no such step in the registry", path, m[1], m[2])
			}
			if !slices.Contains(called, m[1]) {
				called = append(called, m[1])
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(called)
	if got := contract.Builtins().Names(); !slices.Equal(got, called) {
		t.Errorf("the registry holds %v; the shipped steps call %v", got, called)
	}
}
