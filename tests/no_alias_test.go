package tests

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// forwards lists the package-level aliases and forwarding names in the Go
// files under root's internal/ and roles/ (tests aside): a type alias of
// another workline package's type, or a var or const that is only another
// workline package's name. The moves of ADR-0070 leave none: every caller
// names the real home (the owner's rule, 2026-11-20).
func forwards(t *testing.T, root string) []string {
	t.Helper()
	var found []string
	for _, top := range []string{"internal", "roles"} {
		dir := filepath.Join(root, top)
		_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			f, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			local := map[string]bool{} // the import names of workline packages
			for _, im := range f.Imports {
				p, _ := strconv.Unquote(im.Path.Value)
				if !strings.HasPrefix(p, modulePath+"/") {
					continue
				}
				name := p[strings.LastIndex(p, "/")+1:]
				if im.Name != nil {
					name = im.Name.Name
				}
				local[name] = true
			}
			isForward := func(e ast.Expr) bool {
				s, ok := e.(*ast.SelectorExpr)
				if !ok {
					return false
				}
				x, ok := s.X.(*ast.Ident)
				return ok && local[x.Name]
			}
			rel, _ := filepath.Rel(root, path)
			for _, decl := range f.Decls {
				g, ok := decl.(*ast.GenDecl)
				if !ok {
					continue
				}
				for _, spec := range g.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						if s.Assign.IsValid() && isForward(s.Type) {
							found = append(found, filepath.ToSlash(rel)+": type "+s.Name.Name)
						}
					case *ast.ValueSpec:
						if len(s.Values) == 1 && len(s.Names) == 1 && isForward(s.Values[0]) {
							found = append(found, filepath.ToSlash(rel)+": "+s.Names[0].Name)
						}
					}
				}
			}
			return nil
		})
	}
	return found
}

// kept are the aliases that stay, each with why: not a move's leftover,
// a boundary the code holds on purpose.
var kept = map[string]string{
	// A role never imports internal/packs (packs_test, "no verdict reads
	// a pack"): the kit's type is what the tester and the developer name.
	"internal/kit/testdef/testdef.go: type Home": "the kit's own name of a pack's test home",
}

// Given the repository, When it is scanned, Then no Go file of internal/
// nor roles/ forwards a name to another workline package, but the kept.
func TestNoForwardingAliasesAreLeft(t *testing.T) {
	for _, f := range forwards(t, "..") {
		if _, ok := kept[f]; !ok {
			t.Errorf("%s forwards another package's name: name the real home in its callers", f)
		}
	}
}
