package tests

import "testing"

// Given roles/builtin.go, the list of the compiled-in roles, importing
// each role's code, When the check scans it, Then it passes: a file
// directly under roles/ is no role's (docs/spec/layout.md); a role's
// code importing another's still fails.
func TestImportsTheRolesListIsNoRole(t *testing.T) {
	root := sampleTree(t, map[string]string{
		"roles/builtin.go":          goFile("roles", judgeCode) + "import _ \"" + testerCode + "\"\n",
		"roles/developer/code/a.go": goFile("code", testerCode),
	})
	found, err := checkImports(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0][:len("roles/developer/code/a.go")] != "roles/developer/code/a.go" {
		t.Errorf("want the developer's import alone, got %q", found)
	}
}
