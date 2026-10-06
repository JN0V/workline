package reviewer

// What a judge reads of the code around a finding (#127): the function its
// cause lies in, whole; then the functions the finding names, those that
// function calls and those calling it, whole, up to judge-lines-max lines
// all together, the rest named. Found with no build (docs/research/
// code-navigation.md): Go by its own parser; other languages by a
// definition pattern for their extension, and braces, indentation or an
// `end` for a function's bounds; references by word search at the head.

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/pathglob"
)

// language is how functions are found in a family of files.
type language struct {
	name string
	exts []string
	def  string // a definition's pattern, NAME standing for its name
	body string // go, braces, indent (Python), end (Ruby, Lua)
}

var languages = []language{
	{"Go", []string{".go"}, `^func\s+(?:\([^)]*\)\s*)?(NAME)\s*[\[(]`, "go"},
	{"shell", []string{".sh", ".bash", ".zsh", ".ksh"}, `^\s*(?:function\s+(NAME)\b|(NAME)\s*\(\s*\))`, "braces"},
	{"Python", []string{".py"}, `^\s*(?:async\s+)?def\s+(NAME)\s*\(`, "indent"},
	{"Ruby", []string{".rb"}, `^\s*def\s+(?:self\.)?(NAME)\b`, "end"},
	{"Lua", []string{".lua"}, `^\s*(?:local\s+)?function\s+(?:[\w.:]*[.:])?(NAME)\s*\(`, "end"},
	{"C-like", []string{".c", ".h", ".cc", ".cpp", ".cxx", ".hpp", ".hh", ".java", ".cs", ".js", ".jsx", ".mjs", ".cjs", ".ts", ".tsx", ".rs", ".kt", ".kts", ".swift", ".php", ".scala", ".dart", ".groovy"},
		`(?:\b(?:function|fn|fun|func|def)\s+(NAME)\b|\b(?:const|let|var)\s+(NAME)\s*=\s*(?:async\s+)?(?:function\b|\([^)]*\)\s*=>|[A-Za-z_$][\w$]*\s*=>)|^\s*(?:[\w$:<>,*&\[\]~.]+\s+)*(NAME)\s*\([^;{]*(?:\{.*)?$)`, "braces"},
}

// notCalls are words a pattern would take for a function's name: the
// keywords that open a block with a parenthesis.
var notCalls = map[string]bool{"if": true, "for": true, "while": true, "switch": true, "catch": true, "return": true,
	"else": true, "do": true, "new": true, "throw": true, "await": true, "case": true, "typeof": true, "sizeof": true,
	"elif": true, "until": true, "with": true, "foreach": true, "func": true, "function": true, "fn": true, "fun": true,
	"def": true, "go": true, "defer": true, "select": true, "range": true, "make": true, "len": true, "append": true}

func languageOf(file string) *language {
	ext := strings.ToLower(path.Ext(file))
	for i := range languages {
		if slices.Contains(languages[i].exts, ext) {
			return &languages[i]
		}
	}
	return nil
}

func (l *language) pattern(name string) *regexp.Regexp {
	n := `[A-Za-z_$][\w$]*`
	if name != "" {
		n = regexp.QuoteMeta(name)
	}
	return regexp.MustCompile(strings.ReplaceAll(l.def, "NAME", n))
}

// defined is the name a line defines, by the language's pattern; "" none.
func (l *language) defined(line string, re *regexp.Regexp) string {
	m := re.FindStringSubmatch(line)
	if m == nil {
		return ""
	}
	for _, g := range m[1:] {
		if g != "" {
			if notCalls[g] {
				return ""
			}
			return g
		}
	}
	return ""
}

// fn is a function found: its file, its lines (its doc comment included),
// its name, and what it is to the finding.
type fn struct {
	file     string
	from, to int
	name     string
	why      string
	calls    []string // the names it calls, in order
}

func (f fn) lines() int  { return f.to - f.from + 1 }
func (f fn) key() string { return fmt.Sprintf("%s:%d", f.file, f.from) }

// reacher finds functions in the repository at one commit, each file read
// once.
type reacher struct {
	repo, head string
	skip       []string // test files: what the tests judge reads apart
	files      map[string][]string
}

func (r *reacher) lines(file string) []string {
	if l, ok := r.files[file]; ok {
		return l
	}
	text, _ := fileAt(r.repo, r.head, file)
	l := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	r.files[file] = l
	return l
}

// enclosing is the function a file's line lies in; ok false when none is found.
func (r *reacher) enclosing(file string, at int) (fn, bool) {
	l := languageOf(file)
	if l == nil {
		return fn{}, false
	}
	lines := r.lines(file)
	if l.body == "go" {
		if f, ok, parsed := goEnclosing(file, lines, at); parsed {
			return f, ok
		}
	}
	re := l.pattern("")
	for n := min(at, len(lines)); n >= 1; n-- {
		name := l.defined(lines[n-1], re)
		if name == "" {
			continue
		}
		if f, ok := bounds(l, file, lines, n, name); ok && f.to >= at {
			return f, true
		}
	}
	return fn{}, false
}

// at is the function defined on a file's line n.
func (r *reacher) at(file string, n int, name string) (fn, bool) {
	l := languageOf(file)
	lines := r.lines(file)
	if l.body == "go" {
		if f, ok, parsed := goEnclosing(file, lines, n); parsed {
			return f, ok && f.name == name
		}
	}
	return bounds(l, file, lines, n, name)
}

// goEnclosing finds, with Go's parser, the function declared around line at;
// parsed false when the file does not parse.
func goEnclosing(file string, lines []string, at int) (fn, bool, bool) {
	fset := token.NewFileSet()
	src := strings.Join(lines, "\n") + "\n"
	af, err := parser.ParseFile(fset, file, src, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return fn{}, false, false
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
		f := fn{file: file, from: from, to: to, name: fd.Name.Name}
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
				f.calls = append(f.calls, name)
			}
			return true
		})
		return f, true, true
	}
	return fn{}, false, true
}

var (
	callRe = regexp.MustCompile(`([A-Za-z_$][\w$]*)\s*\(`)
	wordRe = regexp.MustCompile(`([A-Za-z_][\w-]*)`)
)

// bounds reads a function from its definition on line def: its body by the
// language's rule, the comment just over it, and the names it calls.
func bounds(l *language, file string, lines []string, def int, name string) (fn, bool) {
	to := 0
	switch l.body {
	case "indent", "end":
		to = indentEnd(lines, def, l.body == "end")
	default:
		to = braceEnd(lines, def, l.name == "shell")
	}
	if to == 0 {
		return fn{}, false
	}
	from := def
	for from > 1 && isComment(lines[from-2]) {
		from--
	}
	return fn{file: file, from: from, to: to, name: name, calls: callsIn(l, lines[def:to], name)}, true
}

// callsIn are the names some lines call, in order, self aside: a word
// before a parenthesis; in shell and Ruby any word, a function being
// called there with none.
func callsIn(l *language, lines []string, self string) []string {
	var out []string
	seen := map[string]bool{self: true}
	re := callRe
	if l.name == "shell" || l.name == "Ruby" {
		re = wordRe
	}
	for _, line := range lines {
		for _, m := range re.FindAllStringSubmatch(line, -1) {
			w := m[1]
			if !seen[w] && !notCalls[w] {
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
		line := stripCode(lines[n-1], shell)
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

// stripCode drops a line's strings and comment, roughly: their braces are
// not the code's.
func stripCode(line string, shell bool) string {
	var b strings.Builder
	var quote rune
	prev := ' '
	for _, c := range line {
		switch {
		case quote != 0:
			if c == quote && prev != '\\' {
				quote = 0
			}
		case c == '"' || (c == '`' && !shell):
			quote = c
		case c == '#' && shell && (prev == ' ' || prev == '\t' || b.Len() == 0):
			return b.String()
		case c == '/' && prev == '/' && !shell:
			return b.String()
		default:
			b.WriteRune(c)
		}
		prev = c
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

// defs finds where each name is defined, in the files of the languages
// given (nil: every language), at the head: one search for every name.
func (r *reacher) defs(names []string, langs []*language) map[string][]fn {
	out := map[string][]fn{}
	var ok []string
	for _, n := range names {
		if regexp.MustCompile(`^[A-Za-z_][\w]*$`).MatchString(n) && len(n) > 1 {
			ok = append(ok, n)
		}
	}
	if len(ok) == 0 {
		return out
	}
	if langs == nil {
		for i := range languages {
			langs = append(langs, &languages[i])
		}
	}
	args := []string{"grep", "-n", "-I", "-w", "-E", "-e", "(" + strings.Join(ok, "|") + ")", r.head, "--"}
	for _, l := range langs {
		for _, e := range l.exts {
			args = append(args, "*"+e)
		}
	}
	args = append(args, ":(exclude)vendor/**", ":(exclude)**/node_modules/**", ":(exclude)third_party/**")
	hits, err := git(r.repo, args...)
	if err != nil {
		return out // none found: git grep fails
	}
	res := map[*language]*regexp.Regexp{}
	for _, h := range strings.Split(strings.TrimSpace(hits), "\n") {
		file, n, line, okh := hit(h, r.head)
		if !okh || pathglob.Any(r.skip, file) {
			continue
		}
		l := languageOf(file)
		if l == nil {
			continue
		}
		if res[l] == nil {
			res[l] = l.pattern("")
		}
		name := l.defined(line, res[l])
		if name == "" || !slices.Contains(ok, name) {
			continue
		}
		if f, found := r.at(file, n, name); found {
			out[name] = append(out[name], f)
		}
	}
	return out
}

// hit reads a `git grep -n` line at a commit: file, line number, text.
func hit(h, head string) (string, int, string, bool) {
	h = strings.TrimPrefix(h, head+":")
	file, rest, ok := strings.Cut(h, ":")
	if !ok {
		return "", 0, "", false
	}
	num, text, ok := strings.Cut(rest, ":")
	n := 0
	if _, err := fmt.Sscan(num, &n); err != nil || !ok {
		return "", 0, "", false
	}
	return file, n, text, true
}

// callers are the functions, in the files of the cause's language, that
// name the function f as a word, f's own lines aside.
func (r *reacher) callers(f fn) []fn {
	l := languageOf(f.file)
	args := []string{"grep", "-n", "-I", "-w", "-F", "-e", f.name, r.head, "--"}
	for _, e := range l.exts {
		args = append(args, "*"+e)
	}
	hits, err := git(r.repo, append(args, ":(exclude)vendor/**", ":(exclude)**/node_modules/**", ":(exclude)third_party/**")...)
	if err != nil {
		return nil
	}
	re := l.pattern(f.name)
	var out []fn
	seen := map[string]bool{f.key(): true}
	for _, h := range strings.Split(strings.TrimSpace(hits), "\n") {
		file, n, line, ok := hit(h, r.head)
		if !ok || pathglob.Any(r.skip, file) || (file == f.file && n >= f.from && n <= f.to) || l.defined(line, re) != "" {
			continue
		}
		c, ok := r.enclosing(file, n)
		if !ok || seen[c.key()] {
			continue
		}
		seen[c.key()] = true
		out = append(out, c)
	}
	return out
}

// pick is the definitions of a name worth showing, nearest first: the one
// in the file, else those in its folder, else every one if few; many: it
// is named, none shown.
func pick(defs []fn, file string) (shown []fn, many bool) {
	var same, near []fn
	for _, d := range defs {
		switch {
		case d.file == file:
			same = append(same, d)
		case path.Dir(d.file) == path.Dir(file):
			near = append(near, d)
		}
	}
	switch {
	case len(same) > 0:
		return same[:1], false
	case len(near) > 0:
		return near[:min(len(near), 2)], false
	case len(defs) <= 2:
		return defs, false
	}
	return nil, true
}

// namedIn are the names a finding's words point to: a word in backticks,
// one followed by a parenthesis, one holding a capital or an underscore.
func namedIn(text string) []string {
	var out []string
	add := func(w string) {
		if len(w) > 2 && !slices.Contains(out, w) {
			out = append(out, w)
		}
	}
	for _, m := range regexp.MustCompile("`([^`]+)`").FindAllStringSubmatch(text, -1) {
		for _, w := range regexp.MustCompile(`[A-Za-z_]\w*`).FindAllString(m[1], -1) {
			add(w)
		}
	}
	for _, m := range regexp.MustCompile(`\b([A-Za-z_]\w*)(\()?`).FindAllStringSubmatch(text, -1) {
		w := m[1]
		if m[2] != "" || strings.Contains(w, "_") || strings.ToLower(w) != w {
			add(w)
		}
	}
	return out
}

// reach is what a judge reads of the code a finding stands on, at most
// limit lines: the function its cause lies in (else the lines around it),
// the symptom's, then the functions the finding names, those the cause's
// calls, and those calling it, each whole or named.
func reach(repo, head string, tests []string, f Finding, at int, symptomAt int, limit int) string {
	r := &reacher{repo: repo, head: head, skip: tests, files: map[string][]string{}}
	var b strings.Builder
	left := limit
	shown := map[string]bool{}
	var cut []string
	write := func(g fn, title string) {
		lines := r.lines(g.file)
		fmt.Fprintf(&b, "\n### %s, lines %d to %d: %s\n\n```\n", g.file, g.from, g.to, title)
		for n := g.from; n <= g.to && n <= len(lines); n++ {
			fmt.Fprintf(&b, "%d  %s\n", n, lines[n-1])
		}
		b.WriteString("```\n")
		shown[g.key()] = true
		left -= g.lines()
	}
	around := func(file string, at int, why string) fn {
		lines := r.lines(file)
		g := fn{file: file, from: max(at-15, 1), to: min(at+15, len(lines))}
		fmt.Fprintf(&b, "\n### %s, lines %d to %d: %s\n\n```\n", file, g.from, g.to, why)
		for n := g.from; n <= g.to; n++ {
			fmt.Fprintf(&b, "%d  %s\n", n, lines[n-1])
		}
		b.WriteString("```\n")
		left -= g.lines()
		return g
	}
	// The cause's own code, whole when it fits, else the lines around it.
	place := func(file string, at int, what string) (fn, bool) {
		g, ok := r.enclosing(file, at)
		switch {
		case limit <= 0:
			around(file, at, "the lines around "+what)
			return fn{}, false
		case !ok:
			lang := "no language it knows"
			if l := languageOf(file); l != nil {
				lang = l.name
			}
			w := around(file, at, fmt.Sprintf("the lines around %s, no function found around it (%s)", what, lang))
			if l := languageOf(file); l != nil { // what those lines call stands for their function's calls
				w.calls = callsIn(l, r.lines(file)[w.from-1:w.to], "")
			}
			return w, true
		case g.lines() > left:
			around(file, at, fmt.Sprintf("the lines around %s, in `%s` (lines %d to %d, longer than judge-lines-max)", what, g.name, g.from, g.to))
			return g, true
		case shown[g.key()]:
			return g, true
		}
		write(g, fmt.Sprintf("`%s`, the function %s lies in", g.name, what))
		return g, true
	}
	b.WriteString("\n## The code it stands on\n")
	if limit > 0 {
		fmt.Fprintf(&b, "\nAs it reads at %s: the function the cause lies in, whole; then the functions the finding names, those the cause's function calls and those calling it, up to %d lines all together (judge-lines-max). Found with no build: a call may be missed, a name taken for another of the same name.\n", short(head), limit)
	}
	cause, ok := place(f.Cause.Path, at, "the cause")
	if f.Symptom != nil && symptomAt > 0 {
		place(f.Symptom.Path, symptomAt, "the symptom")
	}
	if !ok || limit <= 0 {
		return b.String()
	}
	type want struct {
		g   fn
		why string
	}
	var wants []want
	var many []string
	add := func(names []string, langs []*language, why string) {
		found := r.defs(names, langs)
		for _, n := range names {
			ds, m := pick(found[n], f.Cause.Path)
			if m {
				many = append(many, fmt.Sprintf("`%s` (%d definitions)", n, len(found[n])))
			}
			for _, d := range ds {
				wants = append(wants, want{d, why})
			}
		}
	}
	add(namedIn(f.Title+" "+f.Why+" "+f.Fix), nil, "named by the finding")
	if l := languageOf(f.Cause.Path); l != nil {
		by := "called by the lines around the cause"
		if cause.name != "" {
			by = fmt.Sprintf("called by `%s`", cause.name)
		}
		add(cause.calls, []*language{l}, by)
		if cause.name != "" {
			for _, c := range r.callers(cause) {
				wants = append(wants, want{c, fmt.Sprintf("calls `%s`", cause.name)})
			}
		}
	}
	for _, w := range wants {
		if shown[w.g.key()] {
			continue
		}
		if w.g.lines() > left {
			cut = append(cut, fmt.Sprintf("`%s` (%s:%d-%d, %s)", w.g.name, w.g.file, w.g.from, w.g.to, w.why))
			shown[w.g.key()] = true
			continue
		}
		write(w.g, fmt.Sprintf("`%s`, %s", w.g.name, w.why))
	}
	if len(cut) > 0 {
		if len(cut) > 12 {
			cut = append(cut[:12], fmt.Sprintf("and %d more", len(cut)-12))
		}
		fmt.Fprintf(&b, "\nNot shown, past %d lines: %s. What would prove or refute the finding may lie in them.\n", limit, strings.Join(cut, "; "))
	}
	if len(many) > 0 {
		fmt.Fprintf(&b, "\nNot shown, defined in too many places to tell which: %s.\n", strings.Join(many, ", "))
	}
	return b.String()
}
