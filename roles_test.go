package workline

import (
	"io/fs"
	"testing"
)

// loaded names what the engine reads at a role's top level
// (docs/spec/role-contract.md, "Layout"): beside it, only README.md, for
// the role's users, and docs/, the pages about the role.
var loaded = map[string]bool{
	"role.yaml": true, "persona.md": true, "instruction.md": true, "policy.md": true,
	"pre": true, "post": true, "knowledge": true, "lenses": true, "skills": true,
}

func TestRoleTopLevelHoldsOnlyWhatTheEngineLoads(t *testing.T) {
	roles, err := fs.ReadDir(Roles, "roles")
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range roles {
		if !r.IsDir() {
			t.Errorf("roles/%s: a file beside the roles", r.Name())
			continue
		}
		entries, err := fs.ReadDir(Roles, "roles/"+r.Name())
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			name := e.Name()
			switch {
			case name == "README.md" && !e.IsDir(), name == "docs" && e.IsDir(), loaded[name]:
			default:
				t.Errorf("roles/%s/%s: the engine does not load it; a page about the role goes to roles/%s/docs/",
					r.Name(), name, r.Name())
			}
		}
	}
}
