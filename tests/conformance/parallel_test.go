package conformance

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// parsedRunner reads the runner's own source, to check how it schedules
// its cases.
func parsedRunner(t *testing.T) *ast.File {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), "runner_test.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func funcNamed(f *ast.File, name string) *ast.FuncDecl {
	for _, d := range f.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok && fn.Recv == nil && fn.Name.Name == name {
			return fn
		}
	}
	return nil
}

// Cases in parallel: Given the conformance suite, When it runs, Then each
// case's subtest calls t.Parallel(), so the cases run side by side.
func TestConformanceCasesRunInParallel(t *testing.T) {
	fn := funcNamed(parsedRunner(t), "TestConformance")
	if fn == nil {
		t.Fatal("TestConformance not found")
	}
	// The case's subtest is the last t.Run of the function.
	var last *ast.FuncLit
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Run" && len(call.Args) == 2 {
			if lit, ok := call.Args[1].(*ast.FuncLit); ok {
				last = lit
			}
		}
		return true
	})
	if last == nil {
		t.Fatal("no t.Run of a case found in TestConformance")
	}
	found := false
	ast.Inspect(last.Body, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Parallel" {
				found = true
			}
		}
		return true
	})
	if !found {
		t.Error("a case's subtest does not call t.Parallel()")
	}
}

// Shared engines are safe: Given two cases with the same build tags run at
// once, When they start, Then engineFor takes a lock around the engines map,
// so the tagged engine is built once, with no race.
func TestEngineForLocksEngines(t *testing.T) {
	fn := funcNamed(parsedRunner(t), "engineFor")
	if fn == nil {
		t.Fatal("engineFor not found")
	}
	locked := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Lock" {
				locked = true
			}
		}
		return true
	})
	if !locked {
		t.Error("engineFor reads and writes the engines map without taking a lock")
	}
}
