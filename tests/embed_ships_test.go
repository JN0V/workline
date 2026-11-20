package tests

import (
	"io/fs"
	"path"
	"strings"
	"testing"

	"github.com/JN0V/workline"
)

// The binary ships what each role's run reads, named one by one
// (docs/spec/layout.md, "A role folder"): never a role's code/, docs/,
// cases/ nor README.md, and every role keeps its role.yaml.
func TestTheBinaryShipsNoRoleCodeNorDocs(t *testing.T) {
	roles := map[string]bool{}
	err := fs.WalkDir(workline.Roles, "roles", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		parts := strings.Split(p, "/")
		if len(parts) < 3 {
			t.Errorf("%s: embedded, but no role's file", p)
			return nil
		}
		switch {
		case parts[2] == "code", parts[2] == "docs", parts[2] == "cases", parts[2] == "README.md", path.Ext(p) == ".go":
			t.Errorf("%s: embedded, though no run reads it", p)
		case parts[2] == "role.yaml":
			roles[parts[1]] = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"committer", "documentalist", "reviewer", "inspector", "tester", "developer", "product-owner", "auditor", "judge"} {
		if !roles[name] {
			t.Errorf("roles/%s/role.yaml is not embedded", name)
		}
	}
}
