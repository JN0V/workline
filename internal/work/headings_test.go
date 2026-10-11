package work

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// The headings the engine ships with, one a meaning, until the new names
// of ADR-0039 go in front of them.
func TestShippedHeadings(t *testing.T) {
	want := map[Meaning]string{Need: "Need", Example: "Example", Steps: "Steps to reproduce",
		Validation: "Validation", Verification: "Verification", Scope: "Scope"}
	if got := Headings.Meanings(); !slices.Equal(got, []Meaning{Need, Example, Steps, Validation, Verification, Scope}) {
		t.Errorf("the layout = %v", got)
	}
	for m, h := range want {
		if Name(m) != h || MeaningOf(h) != m || !slices.Equal(Headings.Of(m), []string{h}) {
			t.Errorf("%s: written %q, read %v; want %q", m, Name(m), Headings.Of(m), h)
		}
	}
	for _, m := range Headings.Meanings() {
		if Headings.Builder(m) != (m == Verification || m == Scope) {
			t.Errorf("%s: builder's = %v", m, Headings.Builder(m))
		}
	}
	if MeaningOf("Context") != "" || MeaningOf("need") != "" {
		t.Error("a heading the table does not hold, or not as it holds it, names no meaning")
	}
}

// A meaning taking a new name is written under it and still read under the
// old one; a body holding both reads them as one section.
func TestTableWithANewName(t *testing.T) {
	tb := Headings.With(Validation, "Accepted when")
	if tb.Name(Validation) != "Accepted when" || tb.Meaning("Validation") != Validation || tb.Meaning("Accepted when") != Validation {
		t.Fatalf("With: written %q, read %v", tb.Name(Validation), tb.Of(Validation))
	}
	if Name(Validation) != "Validation" {
		t.Error("With changed the engine's table")
	}
	body := "## Need\n\nA.\n\n## Accepted when\n\nB.\n\n## Validation\n\n_No response_\n\n## Validation\n\nC.\n\n## Context\n\nD."
	got := tb.Read(body)
	if s := got[Validation]; s.Heading != "Accepted when" || s.Text != "\nB.\n\n\nC.\n\n" {
		t.Errorf("Read = %+v", s)
	}
	if len(got) != 2 || got[Need].Text != "\nA.\n\n" {
		t.Errorf("Read = %+v; want need and validation only", got)
	}
	if n, ok := tb.In([]string{"Need", "Validation"}, Validation); !ok || n != "Validation" {
		t.Errorf("In = %q, %v", n, ok)
	}
	if _, ok := tb.In([]string{"Context"}, ""); ok {
		t.Error("no meaning is in no list")
	}
}

// Ready reads the required sections through the table: under a new name,
// an old one, or both, an item is ready alike; a missing one is named as
// the table writes it.
func TestReadyFollowsTheTable(t *testing.T) {
	defer func(was Table) { Headings = was }(Headings)
	Headings = Headings.With(Validation, "Accepted when")
	for _, v := range []string{"## Accepted when\n\nB.", "## Validation\n\nB.", "## Accepted when\n\n_No response_\n\n## Validation\n\nB."} {
		if missing := check("## Need\n\nA.\n\n" + v + "\n\n## Verification\n\nC.\n\n## Scope\n\nD."); len(missing) > 0 {
			t.Errorf("%q: %v", v, missing)
		}
	}
	missing := check("## Need\n\nA.\n\n## Verification\n\nC.\n\n## Scope\n\nD.")
	if len(missing) != 1 || missing[0].Where != "Accepted when" || missing[0].Message != "## Accepted when is missing or empty" {
		t.Errorf("missing = %+v", missing)
	}
}

// The table is the one place the engine names a section's heading: no
// other Go file of the engine holds one in a string, prose included, so a
// new name is one change to the table.
func TestHeadingsNamedOnlyInTheTable(t *testing.T) {
	var words []string
	for _, r := range Headings {
		for _, h := range r.Headings {
			words = append(words, regexp.QuoteMeta(h))
		}
	}
	heading := regexp.MustCompile(`\b(` + strings.Join(words, "|") + `)\b`)
	root := filepath.Join("..", "..")
	for _, dir := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") ||
				filepath.ToSlash(path) == "../../internal/work/headings.go" {
				return err
			}
			src, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, path, src, parser.SkipObjectResolution)
			if err != nil {
				return err
			}
			ast.Inspect(f, func(n ast.Node) bool {
				lit, ok := n.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return true
				}
				text, err := strconv.Unquote(lit.Value)
				if err != nil {
					text = lit.Value
				}
				if h := heading.FindString(text); h != "" {
					t.Errorf("%s: names the heading %q: read it from the table (work.Name)", fset.Position(lit.Pos()), h)
				}
				return true
			})
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
