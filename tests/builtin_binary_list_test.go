package tests

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// shippedRoles lists the roles whose folder holds code/, the built-in
// code the engine runs for their pre and post, sorted.
func shippedRoles(t *testing.T) []string {
	t.Helper()
	dirs, err := filepath.Glob(filepath.Join("..", "roles", "*", "code"))
	if err != nil {
		t.Fatal(err)
	}
	var coded []string
	for _, d := range dirs {
		coded = append(coded, filepath.Base(filepath.Dir(d)))
	}
	slices.Sort(coded)
	return coded
}

// binaryRoles lists the roles the binary's own list (cmd/workline/builtins.go,
// or the list it imports) compiles in, read from the source: the roles each
// listed package joins to the registry in its init(). It stops, naming the
// file, when a test file of this folder keeps a list of its own: such a
// list fills the registry of the tests, whatever the binary forgets.
func binaryRoles(t *testing.T) []string {
	t.Helper()
	const module = "github.com/JN0V/workline/"
	join := regexp.MustCompile(`contract\.Join\("([a-z-]+)"`)
	// blank gives the module's packages a Go file imports for their init alone.
	blank := func(file string) []string {
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, imp := range parsed.Imports {
			p := strings.Trim(imp.Path.Value, `"`)
			if imp.Name != nil && imp.Name.Name == "_" && strings.HasPrefix(p, module) {
				out = append(out, p)
			}
		}
		return out
	}
	// scan reads a package's own Go files: the roles it joins, the packages it imports blank.
	scan := func(path string) (roles, blanks []string) {
		files, err := filepath.Glob(filepath.Join("..", strings.TrimPrefix(path, module), "*.go"))
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range files {
			if strings.HasSuffix(f, "_test.go") {
				continue
			}
			data, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			for _, m := range join.FindAllStringSubmatch(string(data), -1) {
				roles = append(roles, m[1])
			}
			blanks = append(blanks, blank(f)...)
		}
		return roles, blanks
	}

	own, err := filepath.Glob("*_test.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range own {
		for _, p := range blank(f) {
			if roles, _ := scan(p); len(roles) > 0 {
				t.Fatalf("tests/%s imports %s, which joins %v, for itself: the tests keep a list of compiled-in roles of their own, so a role the binary's list forgets is not seen; check cmd/workline/builtins.go, or share its list", f, p, roles)
			}
		}
	}

	var got []string
	seen := map[string]bool{}
	var walk func(path string)
	walk = func(path string) {
		if seen[path] {
			return
		}
		seen[path] = true
		roles, blanks := scan(path)
		got = append(got, roles...)
		for _, b := range blanks {
			walk(b)
		}
	}
	for _, p := range blank(filepath.Join("..", "cmd", "workline", "builtins.go")) {
		walk(p)
	}
	slices.Sort(got)
	return slices.Compact(got)
}

// A role the binary forgets
//
// Given a role's import removed from the binary's list only,
// When a contributor runs `go test ./...`,
// Then a test fails and names that role.
//
// Every shipped role with code/ is one the binary's list compiles in;
// the failure names each one the list leaves out.
func TestTheBinaryForgetsNoRoleWithCode(t *testing.T) {
	in := binaryRoles(t)
	for _, role := range shippedRoles(t) {
		if !slices.Contains(in, role) {
			t.Errorf("the role %q ships code/ (roles/%s/code), but cmd/workline/builtins.go does not import it: its pre and post would run as the shared ones", role, role)
		}
	}
}

// Both lists whole
//
// Given every compiled-in role in the binary's list,
// When a contributor runs `go test ./...`,
// Then that test passes.
//
// The binary's list compiles in exactly the shipped roles with code/.
func TestTheBinaryListHoldsExactlyTheRolesWithCode(t *testing.T) {
	if got, want := binaryRoles(t), shippedRoles(t); !slices.Equal(got, want) {
		t.Errorf("the binary's list compiles in %v; the shipped roles with code/ are %v", got, want)
	}
}
