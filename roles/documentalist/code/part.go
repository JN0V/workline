package documentalist

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

// A source may name a part of a file after `#` (ADR-0053): a Markdown
// heading's anchor, as GitHub gives it; a Go declaration, `Name` or
// `Type.Method`; a YAML key, its path dotted from the top, `jobs.test`. A doc
// is then suspect only when that part changes, and a task reads that part,
// not the whole file.

// partOf is the part of a source file anchor names, as text. An empty anchor
// is the whole file (a Markdown doc's body). found is false when the part is
// not there; err says a file that cannot be read for parts.
func partOf(file, content, anchor string) (text string, found bool, err error) {
	switch {
	case strings.HasSuffix(file, ".md"):
		text, found = Section(content, anchor)
		return text, found, nil
	case anchor == "":
		return content, true, nil
	case strings.HasSuffix(file, ".go"):
		return goPart(file, content, anchor)
	case strings.HasSuffix(file, ".yml"), strings.HasSuffix(file, ".yaml"):
		return yamlPart(content, anchor)
	}
	return "", false, fmt.Errorf("%s#%s: a part is named only in Markdown (a heading), Go (a declaration) or YAML (a key)", file, anchor)
}

// isPart says whether a source names a part of a file other than a doc's:
// such a source is read and compared as that part.
func isPart(file, anchor string) bool {
	return anchor != "" && !strings.HasSuffix(file, ".md")
}

// partLang is the fence a task shows a part in.
func partLang(file string) string {
	switch path.Ext(file) {
	case ".go":
		return "go"
	case ".yml", ".yaml":
		return "yaml"
	}
	return ""
}

// goPart is a Go declaration with its doc comment: a function, a type, a
// constant or a variable by its name, a method as `Type.Method`. In a
// grouped declaration, only the spec naming it, unless the group is that one
// spec.
func goPart(file, content, anchor string) (string, bool, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, content, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return "", false, fmt.Errorf("%s does not parse as Go: %v", file, err)
	}
	recv, name, method := strings.Cut(anchor, ".")
	if !method {
		name, recv = recv, ""
	}
	span := func(doc *ast.CommentGroup, n ast.Node) (string, bool, error) {
		from := n.Pos()
		if doc != nil {
			from = doc.Pos()
		}
		return content[fset.Position(from).Offset:fset.Position(n.End()).Offset], true, nil
	}
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Name.Name == name && receiver(d) == recv {
				return span(d.Doc, d)
			}
		case *ast.GenDecl:
			if recv != "" {
				continue
			}
			for _, s := range d.Specs {
				var names []*ast.Ident
				var doc *ast.CommentGroup
				switch s := s.(type) {
				case *ast.TypeSpec:
					names, doc = []*ast.Ident{s.Name}, s.Doc
				case *ast.ValueSpec:
					names, doc = s.Names, s.Doc
				}
				for _, id := range names {
					if id.Name != name {
						continue
					}
					if len(d.Specs) == 1 {
						return span(d.Doc, d)
					}
					return span(doc, s)
				}
			}
		}
	}
	return "", false, nil
}

// receiver is a method's receiver type name, pointer and type parameters
// left out; "" for a function.
func receiver(d *ast.FuncDecl) string {
	if d.Recv == nil || len(d.Recv.List) == 0 {
		return ""
	}
	t := d.Recv.List[0].Type
	for {
		switch x := t.(type) {
		case *ast.StarExpr:
			t = x.X
		case *ast.IndexExpr:
			t = x.X
		case *ast.IndexListExpr:
			t = x.X
		case *ast.Ident:
			return x.Name
		default:
			return ""
		}
	}
}

// yamlPart is a YAML key's entry, its path dotted from the top of the
// document (`jobs.test`): from the comment lines right above the key to the
// line before what follows it and the comments right above that, blank
// lines at its end left out. A job's comment says what it does: it is part
// of it.
func yamlPart(content, anchor string) (string, bool, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(content), &doc); err != nil {
		return "", false, fmt.Errorf("does not parse as YAML: %v", err)
	}
	if len(doc.Content) == 0 {
		return "", false, nil
	}
	lines := strings.Split(content, "\n")
	node, end := doc.Content[0], len(lines)
	var key *yaml.Node
	// A key may hold dots itself (GitLab's `.workline:bot`): the longest key
	// the path starts with is taken at each level.
	for rest := anchor; rest != ""; {
		if node.Kind != yaml.MappingNode {
			return "", false, nil
		}
		at := -1
		for i := 0; i+1 < len(node.Content); i += 2 {
			k := node.Content[i].Value
			if (rest == k || strings.HasPrefix(rest, k+".")) && (at < 0 || len(k) > len(node.Content[at].Value)) {
				at = i
			}
		}
		if at < 0 {
			return "", false, nil
		}
		if at+2 < len(node.Content) {
			end = commentsAbove(lines, node.Content[at+2].Line) - 1
		}
		key, node = node.Content[at], node.Content[at+1]
		rest = strings.TrimPrefix(strings.TrimPrefix(rest, key.Value), ".")
	}
	part := lines[commentsAbove(lines, key.Line)-1 : max(end, key.Line)]
	for len(part) > 1 && strings.TrimSpace(part[len(part)-1]) == "" {
		part = part[:len(part)-1]
	}
	return strings.Join(part, "\n"), true, nil
}

// commentsAbove is the first line, 1-based, of the comment lines right
// above line n, n itself when there are none.
func commentsAbove(lines []string, n int) int {
	for n > 1 && strings.HasPrefix(strings.TrimSpace(lines[n-2]), "#") {
		n--
	}
	return n
}

// partAt is the part a source names at a revision; found is false when the
// file or the part is not there.
func partAt(dir, rev, file, anchor string) (string, bool, error) {
	content, err := git(dir, "show", rev+":"+file)
	if err != nil {
		return "", false, nil
	}
	return partOf(file, content, anchor)
}

// partChanged is changed for a source naming a part of a file other than a
// doc: the commits since checked that changed that part. With names, the
// change counts only as the whole-file rule counts it (ADR-0014, amended
// 2026-10-21): the doc names the file, or a line of the part removed or
// changed holds a name or a value the doc writes. A part gone is said.
func partChanged(dir, checked, rev, file, anchor string, names map[string]bool) (string, error) {
	now, found, err := partAt(dir, rev, file, anchor)
	if err != nil {
		return "", err
	}
	if !found {
		if _, e := git(dir, "show", rev+":"+file); e != nil {
			return "", fmt.Errorf("%s no longer exists", file)
		}
		return "part #" + anchor + " of " + file + " is gone", nil
	}
	old, was, err := partAt(dir, checked, file, anchor)
	was = was && err == nil
	if was && old == now {
		return "", nil
	}
	named := names == nil || names[strings.ToLower(file)] || names[strings.ToLower(path.Base(file))] ||
		slices.ContainsFunc(removedLines(old, now), func(l string) bool { return holdsName(l, names) })
	if was && !named {
		return "", nil
	}
	out, err := git(dir, "log", "--format=%H %h %s", checked+".."+rev, "--", file)
	if err != nil {
		return "", err
	}
	var kept []string
	for _, l := range strings.Split(out, "\n") {
		full, rest, _ := strings.Cut(l, " ")
		if full == "" {
			continue
		}
		before, _, _ := partAt(dir, full+"^", file, anchor)
		after, _, _ := partAt(dir, full, file, anchor)
		if before != after {
			kept = append(kept, rest)
		}
	}
	if len(kept) == 0 {
		return fmt.Sprintf("(part #%s of %s is not as it was at %s)", anchor, file, checked), nil
	}
	return strings.Join(kept, "\n"), nil
}

// removedLines are the lines of old that now no longer holds, as many times
// as old holds more of them: a part's lines removed or changed.
func removedLines(old, now string) []string {
	left := map[string]int{}
	for _, l := range strings.Split(now, "\n") {
		left[l]++
	}
	var out []string
	for _, l := range strings.Split(old, "\n") {
		if left[l] > 0 {
			left[l]--
			continue
		}
		out = append(out, l)
	}
	return out
}
