package developer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The old name is written in two halves, so this file never holds it
// whole: the search below for the old name must find nothing.
const warnTestOldName = "TestReadMergeTests" + "FindingSentToTesterIsNotOpen"
const warnTestNewName = "TestReadMergeWarnFindingNeverHoldsWhoeverGotIt"

// warnTestDecls parses merge_warn_test.go and returns the functions it
// defines under a name.
func warnTestDecls(t *testing.T, name string) []*ast.FuncDecl {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "merge_warn_test.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	var found []*ast.FuncDecl
	for _, d := range file.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok && fn.Name.Name == name {
			found = append(found, fn)
		}
	}
	return found
}

// Sent or not: given the test that holds a warn finding with the
// developer's record naming the tester, when its name and comment are
// read, then it is named for what it checks (a warn finding never holds,
// whoever got it), the name it was given for a tests finding sent to the
// tester is found nowhere, and its comment says that a finding from the
// lenses that warn is never counted open, even when the record says it
// went to the tester.
func TestWarnFindingTestIsNamedForWhatItChecks(t *testing.T) {
	err := filepath.WalkDir("..", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") {
			return err
		}
		src, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if strings.Contains(string(src), warnTestOldName) {
			t.Errorf("%s still names %s: it checks no tester hand-off", p, warnTestOldName)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	decls := warnTestDecls(t, warnTestNewName)
	if len(decls) != 1 {
		t.Fatalf("merge_warn_test.go defines %s %d times, want once", warnTestNewName, len(decls))
	}
	if decls[0].Doc == nil {
		t.Fatalf("%s has no comment saying what it checks", warnTestNewName)
	}
	doc := strings.ToLower(decls[0].Doc.Text())
	for _, want := range []string{"warn", "never", "open", "tester"} {
		if !strings.Contains(doc, want) {
			t.Errorf("the comment of %s does not say %q: %q", warnTestNewName, want, doc)
		}
	}
}

// Still guards the fix: given the renamed test, when its setup is read,
// then it keeps the developer's record naming the tester (its Tester and
// Rounds lines), reads the merge as the merge pass does, and asserts on
// the findings counted open: so a merge read that counts a warn finding
// open fails it, whoever the finding was sent to.
func TestWarnFindingTestKeepsTheSentToTesterRecord(t *testing.T) {
	decls := warnTestDecls(t, warnTestNewName)
	if len(decls) != 1 {
		t.Fatalf("merge_warn_test.go defines %s %d times, want once", warnTestNewName, len(decls))
	}
	keys := map[string]bool{}
	var reads, open bool
	ast.Inspect(decls[0].Body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.KeyValueExpr:
			if id, ok := x.Key.(*ast.Ident); ok {
				keys[id.Name] = true
			}
		case *ast.CallExpr:
			if id, ok := x.Fun.(*ast.Ident); ok && id.Name == "mergeReadWith" {
				reads = true
			}
		case *ast.SelectorExpr:
			if x.Sel.Name == "Open" {
				open = true
			}
		}
		return true
	})
	for _, k := range []string{"Tester", "Rounds"} {
		if !keys[k] {
			t.Errorf("%s does not set %s in the developer's record: the sent-to-tester case is lost", warnTestNewName, k)
		}
	}
	if !reads {
		t.Errorf("%s does not read the merge with mergeReadWith", warnTestNewName)
	}
	if !open {
		t.Errorf("%s does not check the findings counted open", warnTestNewName)
	}
}
