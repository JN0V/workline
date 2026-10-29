package tester

// The curator removes, never rewrites or merges (ADR-0055, amendment
// 2026-10-29): a test stays tied to its issue's scenario, and the proof
// covers a removal only. removesOnly holds each file the curator touched
// to that: whole tests gone, nothing written.

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/JN0V/workline/internal/verdict"
)

// removesOnly refuses a proposal that writes test code: each touched file,
// as laid in tree, holds nothing its base did not (removal-writes-code).
func removesOnly(repo, base, tree string, touched []string) (*verdict.Verdict, error) {
	var fs []verdict.Finding
	for _, file := range touched {
		now, err := os.ReadFile(filepath.Join(tree, filepath.FromSlash(path.Clean(file))))
		if errors.Is(err, os.ErrNotExist) {
			continue // deleted whole
		} else if err != nil {
			return nil, err
		}
		was, isNew := "", false
		if listed, err := git(repo, "ls-tree", "--name-only", base, "--", file); err != nil {
			return nil, err
		} else if strings.TrimSpace(listed) == "" {
			isNew = true // a new file: written, not removed
		} else if was, err = git(repo, "show", base+":"+file); err != nil {
			return nil, err
		}
		if what, ok := keepsBase(file, string(now), was, isNew); !ok {
			fs = append(fs, verdict.Finding{Rule: "removal-writes-code", Where: file,
				Message: "the curator removes whole tests, never rewrites or merges them: " + what})
		}
	}
	if len(fs) == 0 {
		return nil, nil
	}
	return &verdict.Verdict{Status: verdict.Block, Summary: "the removal is not proven", Findings: fs}, nil
}

// keepsBase reports whether now holds only what was held, whole tests
// gone; else what it wrote. A Go file is read by its declarations: each
// one now holds is one of the base's, unchanged but for whitespace, so a
// test merged into another by deleting a header is told. Any other file
// by its lines: now's lines are the base's, in order, some dropped —
// weaker, since lines dropped from two tests may join them into one.
func keepsBase(file, now, was string, isNew bool) (string, bool) {
	if isNew {
		return "a new file", false
	}
	if strings.HasSuffix(file, ".go") {
		if w, ok := goDecls(was); ok {
			n, ok := goDecls(now)
			if !ok {
				return "the file no longer parses", false
			}
			return keepsDecls(n, w)
		}
	}
	return subsequence(now, was)
}

// decls are a Go file's package name, its imports and its other top-level
// declarations, each with its whitespace collapsed.
type decls struct {
	pkg     string
	imports map[string]bool
	others  []string
}

func goDecls(src string) (decls, bool) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return decls{}, false
	}
	d := decls{pkg: f.Name.Name, imports: map[string]bool{}}
	text := func(n ast.Node) string {
		return strings.Join(strings.Fields(src[fset.Position(n.Pos()).Offset:fset.Position(n.End()).Offset]), " ")
	}
	for _, decl := range f.Decls {
		if g, ok := decl.(*ast.GenDecl); ok && g.Tok == token.IMPORT {
			for _, s := range g.Specs {
				d.imports[text(s)] = true
			}
			continue
		}
		d.others = append(d.others, text(decl))
	}
	return d, true
}

// keepsDecls: now's imports are among the base's, and each of now's other
// declarations is one of the base's, each base one matched once.
func keepsDecls(now, was decls) (string, bool) {
	if now.pkg != was.pkg {
		return "the package clause changed", false
	}
	for imp := range now.imports {
		if !was.imports[imp] {
			return fmt.Sprintf("the import %s is not the base's", imp), false
		}
	}
	left := map[string]int{}
	for _, d := range was.others {
		left[d]++
	}
	for _, d := range now.others {
		if left[d] == 0 {
			return fmt.Sprintf("%.80q is not one of the base's declarations", d), false
		}
		left[d]--
	}
	return "", true
}

// subsequence reports whether now's lines are was's, in order, some
// dropped (blank lines, trailing spaces and carriage returns aside); else
// the first line that is not.
func subsequence(now, was string) (string, bool) {
	old := strings.Split(was, "\n")
	i := 0
	for _, l := range strings.Split(now, "\n") {
		l = strings.TrimRight(l, " \t\r")
		if l == "" {
			continue
		}
		for i < len(old) && strings.TrimRight(old[i], " \t\r") != l {
			i++
		}
		if i == len(old) {
			return fmt.Sprintf("%q is not the base's", l), false
		}
		i++
	}
	return "", true
}
