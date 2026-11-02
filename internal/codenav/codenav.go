// Package codenav finds the functions of a file with no build
// (docs/research/code-navigation.md): Go by its own parser; other
// languages by a definition pattern for their extension, and braces,
// indentation or an `end` for a function's bounds. The reviewer's judge
// reads a finding's functions by it (#127), the tester the functions a
// Scope names in a file too long to give whole (#908).
package codenav

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"regexp"
	"slices"
	"strings"
)

// Language is how functions are found in a family of files.
type Language struct {
	Name string
	Exts []string
	def  string // a definition's pattern, NAME standing for its name
	Body string // go, braces, indent (Python), end (Ruby, Lua)
}

// Languages are the families known, in the order they are tried.
var Languages = []Language{
	{"Go", []string{".go"}, `^func\s+(?:\([^)]*\)\s*)?(NAME)\s*[\[(]`, "go"},
	{"shell", []string{".sh", ".bash", ".zsh", ".ksh"}, `^\s*(?:function\s+(NAME)\b|(NAME)\s*\(\s*\))`, "braces"},
	{"Python", []string{".py"}, `^\s*(?:async\s+)?def\s+(NAME)\s*\(`, "indent"},
	{"Ruby", []string{".rb"}, `^\s*def\s+(?:self\.)?(NAME)\b`, "end"},
	{"Lua", []string{".lua"}, `^\s*(?:local\s+)?function\s+(?:[\w.:]*[.:])?(NAME)\s*\(`, "end"},
	{"C-like", []string{".c", ".h", ".cc", ".cpp", ".cxx", ".hpp", ".hh", ".java", ".cs", ".js", ".jsx", ".mjs", ".cjs", ".ts", ".tsx", ".rs", ".kt", ".kts", ".swift", ".php", ".scala", ".dart", ".groovy"},
		`(?:\b(?:function|fn|fun|func|def)\s+(NAME)\b|\b(?:const|let|var)\s+(NAME)\s*=\s*(?:async\s+)?(?:function\b|\([^)]*\)\s*=>|[A-Za-z_$][\w$]*\s*=>)|^\s*(?:[\w$:<>,*&\[\]~.]+\s+)*(NAME)\s*\([^;{]*(?:\{.*)?$)`, "braces"},
}

// NotCalls are words a pattern would take for a function's name: the
// keywords that open a block with a parenthesis.
var NotCalls = map[string]bool{"if": true, "for": true, "while": true, "switch": true, "catch": true, "return": true,
	"else": true, "do": true, "new": true, "throw": true, "await": true, "case": true, "typeof": true, "sizeof": true,
	"elif": true, "until": true, "with": true, "foreach": true, "func": true, "function": true, "fn": true, "fun": true,
	"def": true, "go": true, "defer": true, "select": true, "range": true, "make": true, "len": true, "append": true}

// Of is the language of a file by its extension; nil when none is known.
func Of(file string) *Language {
	ext := strings.ToLower(path.Ext(file))
	for i := range Languages {
		if slices.Contains(Languages[i].Exts, ext) {
			return &Languages[i]
		}
	}
	return nil
}

// Pattern matches a line defining the function name; "" any function.
func (l *Language) Pattern(name string) *regexp.Regexp {
	n := `[A-Za-z_$][\w$]*`
	if name != "" {
		n = regexp.QuoteMeta(name)
	}
	return regexp.MustCompile(strings.ReplaceAll(l.def, "NAME", n))
}

// Defined is the name a line defines, by the language's pattern; "" none.
func (l *Language) Defined(line string, re *regexp.Regexp) string {
	m := re.FindStringSubmatch(line)
	if m == nil {
		return ""
	}
	for _, g := range m[1:] {
		if g != "" {
			if NotCalls[g] {
				return ""
			}
			return g
		}
	}
	return ""
}

// Func is a function found: its lines, from 1, its doc comment included;
// its name; the names it calls, in order.
type Func struct {
	From, To int
	Name     string
	Calls    []string
}

// Enclosing is the function a file's line lies in, the file's lines
// given; ok false when none is found.
func Enclosing(file string, lines []string, at int) (Func, bool) {
	l := Of(file)
	if l == nil {
		return Func{}, false
	}
	if l.Body == "go" {
		if f, ok, parsed := goEnclosing(file, lines, at); parsed {
			return f, ok
		}
	}
	re := l.Pattern("")
	for n := min(at, len(lines)); n >= 1; n-- {
		name := l.Defined(lines[n-1], re)
		if name == "" {
			continue
		}
		if f, ok := bounds(l, lines, n, name); ok && f.To >= at {
			return f, true
		}
	}
	return Func{}, false
}

// At is the function name defined on a file's line n.
func At(file string, lines []string, n int, name string) (Func, bool) {
	l := Of(file)
	if l == nil {
		return Func{}, false
	}
	if l.Body == "go" {
		if f, ok, parsed := goEnclosing(file, lines, n); parsed {
			return f, ok && f.Name == name
		}
	}
	return bounds(l, lines, n, name)
}

// Named are the functions name defines in a file, the file's lines given,
// in their order; none when the file's language is not known.
func Named(file string, lines []string, name string) []Func {
	l := Of(file)
	if l == nil {
		return nil
	}
	re := l.Pattern(name)
	var out []Func
	for n, line := range lines {
		if l.Defined(line, re) != name {
			continue
		}
		if f, ok := At(file, lines, n+1, name); ok {
			out = append(out, f)
		}
	}
	return out
}

// goEnclosing finds, with Go's parser, the function declared around line at;
// parsed false when the file does not parse.
func goEnclosing(file string, lines []string, at int) (Func, bool, bool) {
	fset := token.NewFileSet()
	src := strings.Join(lines, "\n") + "\n"
	af, err := parser.ParseFile(fset, file, src, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return Func{}, false, false
	}
	for _, d := range af.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}
		from, to := fset.Position(fd.Pos()).Line, fset.Position(fd.End()).Line
		if fd.Doc != nil {
			from = fset.Position(fd.Doc.Pos()).Line
		}
		if at < from || at > to {
			continue
		}
		f := Func{From: from, To: to, Name: fd.Name.Name}
		seen := map[string]bool{fd.Name.Name: true}
		ast.Inspect(fd, func(n ast.Node) bool {
			c, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			fun := c.Fun
			if ix, ok := fun.(*ast.IndexExpr); ok {
				fun = ix.X
			}
			var name string
			switch e := fun.(type) {
			case *ast.Ident:
				name = e.Name
			case *ast.SelectorExpr:
				name = e.Sel.Name
			}
			if name != "" && !seen[name] {
				seen[name] = true
				f.Calls = append(f.Calls, name)
			}
			return true
		})
		return f, true, true
	}
	return Func{}, false, true
}

var (
	callRe = regexp.MustCompile(`([A-Za-z_$][\w$]*)\s*\(`)
	wordRe = regexp.MustCompile(`([A-Za-z_][\w-]*)`)
)

// bounds reads a function from its definition on line def: its body by the
// language's rule, the comment just over it, and the names it calls.
func bounds(l *Language, lines []string, def int, name string) (Func, bool) {
	to := 0
	switch l.Body {
	case "indent", "end":
		to = indentEnd(lines, def, l.Body == "end")
	default:
		to = braceEnd(lines, def, l.Name == "shell")
	}
	if to == 0 {
		return Func{}, false
	}
	from := def
	for from > 1 && isComment(lines[from-2]) {
		from--
	}
	return Func{From: from, To: to, Name: name, Calls: CallsIn(l, lines[def:to], name)}, true
}

// CallsIn are the names some lines call, in order, self aside: a word
// before a parenthesis; in shell and Ruby any word, a function being
// called there with none.
func CallsIn(l *Language, lines []string, self string) []string {
	var out []string
	seen := map[string]bool{self: true}
	re := callRe
	if l.Name == "shell" || l.Name == "Ruby" {
		re = wordRe
	}
	for _, line := range lines {
		for _, m := range re.FindAllStringSubmatch(line, -1) {
			w := m[1]
			if !seen[w] && !NotCalls[w] {
				seen[w] = true
				out = append(out, w)
			}
		}
	}
	return out
}

func isComment(line string) bool {
	t := strings.TrimSpace(line)
	return strings.HasPrefix(t, "//") || strings.HasPrefix(t, "/*") || strings.HasPrefix(t, "*") ||
		(strings.HasPrefix(t, "#") && !strings.HasPrefix(t, "#!") && !strings.HasPrefix(t, "#include"))
}

// braceEnd is the line a function's braces close on: its first brace within
// six lines of its definition, before any `;`; 0 when there is none.
func braceEnd(lines []string, def int, shell bool) int {
	depth, opened := 0, false
	for n := def; n <= len(lines); n++ {
		line := StripCode(lines[n-1], shell)
		for _, c := range line {
			switch c {
			case ';':
				if !opened && !shell {
					return 0
				}
			case '{':
				depth++
				opened = true
			case '}':
				depth--
				if opened && depth == 0 {
					return n
				}
			}
		}
		if !opened && n-def >= 6 {
			return 0
		}
	}
	return 0
}

// StripCode drops a line's strings and comment, roughly: their braces are
// not the code's. A single quote opens a string or a character only when
// it closes on the line, an apostrophe left open hiding nothing; a
// backslash escapes, except in shell single quotes and Go's raw strings.
func StripCode(line string, shell bool) string {
	var b strings.Builder
	rs := []rune(line)
	for i := 0; i < len(rs); i++ {
		c := rs[i]
		switch {
		case c == '"' || c == '\'' || (c == '`' && !shell):
			raw := c == '`' || (c == '\'' && shell)
			end := -1
			for j := i + 1; j < len(rs) && end < 0; j++ {
				switch {
				case rs[j] == '\\' && !raw:
					j++
				case rs[j] == c:
					end = j
				}
			}
			switch {
			case end >= 0:
				i = end
			case c == '\'':
				b.WriteRune(c)
			default:
				return b.String() // a string going on past the line: its rest is not code
			}
		case c == '#' && shell && (i == 0 || rs[i-1] == ' ' || rs[i-1] == '\t'):
			return b.String()
		case c == '/' && !shell && i+1 < len(rs) && rs[i+1] == '/':
			return b.String()
		default:
			b.WriteRune(c)
		}
	}
	return b.String()
}

// indentEnd is a block's last line by its indentation: the lines after its
// header (up to the first ending in `:` for Python) indented further; with
// end, up to the `end` at the definition's indentation.
func indentEnd(lines []string, def int, end bool) int {
	indent := func(s string) int { return len(s) - len(strings.TrimLeft(s, " \t")) }
	d := indent(lines[def-1])
	start := def
	if !end {
		for start <= len(lines) && start-def < 10 && !strings.HasSuffix(strings.TrimSpace(strings.SplitN(lines[start-1], "#", 2)[0]), ":") {
			start++
		}
		if start > len(lines) || start-def >= 10 {
			return 0
		}
	}
	last := start
	for n := start + 1; n <= len(lines); n++ {
		t := strings.TrimSpace(lines[n-1])
		if t == "" {
			continue
		}
		if indent(lines[n-1]) <= d {
			if end && indent(lines[n-1]) == d && (t == "end" || strings.HasPrefix(t, "end ") || strings.HasPrefix(t, "end)")) {
				return n
			}
			break
		}
		last = n
	}
	return last
}
