package main

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// Every function of the package shaped as a command, func([]string) int,
// is in the dispatcher's table, so `workline --help` lists it: a command
// added without a line of help fails here.
func TestEveryCommandIsDispatched(t *testing.T) {
	inTable := map[string]bool{}
	for _, c := range commands {
		full := runtime.FuncForPC(reflect.ValueOf(c.run).Pointer()).Name()
		inTable[full[strings.LastIndex(full, ".")+1:]] = true
	}
	pkgs, err := parser.ParseDir(token.NewFileSet(), ".", func(fi os.FileInfo) bool { return !strings.HasSuffix(fi.Name(), "_test.go") }, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range pkgs["main"].Files {
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Name.Name == "dispatch" || !commandShaped(fn.Type) {
				continue
			}
			if !inTable[fn.Name.Name] {
				t.Errorf("%s looks like a command but is not in the commands table (cli.go)", fn.Name.Name)
			}
		}
	}
}

func commandShaped(ft *ast.FuncType) bool {
	if ft.Params == nil || len(ft.Params.List) != 1 || len(ft.Params.List[0].Names) > 1 || ft.Results == nil || len(ft.Results.List) != 1 {
		return false
	}
	arr, ok := ft.Params.List[0].Type.(*ast.ArrayType)
	if !ok || arr.Len != nil {
		return false
	}
	el, ok1 := arr.Elt.(*ast.Ident)
	res, ok2 := ft.Results.List[0].Type.(*ast.Ident)
	return ok1 && ok2 && el.Name == "string" && res.Name == "int"
}

// The help lists every command of the table a person runs, names the
// internal ones, and each command answers --help with its usage, exit 0.
func TestHelpCoversTheTable(t *testing.T) {
	var top bytes.Buffer
	topHelp(&top)
	for _, c := range commands {
		if c.hidden {
			if !strings.Contains(top.String(), "Internal") || !strings.Contains(top.String(), c.name) {
				t.Errorf("internal command %s is not named in the help", c.name)
			}
			continue
		}
		if !strings.Contains(top.String(), "\n  "+c.name+" ") {
			t.Errorf("%s is not listed in workline --help", c.name)
		}
	}
	for _, c := range commands {
		out, code := captureStdout(t, func() int { return dispatch([]string{c.name, "--help"}) })
		if code != 0 || !strings.HasPrefix(out, "usage: workline ") {
			t.Errorf("workline %s --help: exit %d, %q", c.name, code, out)
		}
		if c.name != "help" && !strings.HasPrefix(out, "usage: workline "+c.name) {
			t.Errorf("workline %s --help prints another command's usage: %q", c.name, out)
		}
	}
}

func captureStdout(t *testing.T, f func() int) (string, int) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	read := make(chan []byte)
	go func() { out, _ := io.ReadAll(r); read <- out }()
	old := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = old }()
	code := f()
	w.Close()
	return string(<-read), code
}
