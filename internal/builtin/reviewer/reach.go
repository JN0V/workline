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
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/codenav"
	"github.com/JN0V/workline/internal/codepick"
	"github.com/JN0V/workline/internal/pathglob"
)

// language is how functions are found in a family of files.
type language = codenav.Language

func languageOf(file string) *language { return codenav.Of(file) }

var (
	languages = codenav.Languages
	notCalls  = codenav.NotCalls
	stripCode = codenav.StripCode
	callsIn   = codenav.CallsIn
)

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
	f, ok := codenav.Enclosing(file, r.lines(file), at)
	return asFn(file, f, ok), ok
}

// at is the function defined on a file's line n.
func (r *reacher) at(file string, n int, name string) (fn, bool) {
	f, ok := codenav.At(file, r.lines(file), n, name)
	return asFn(file, f, ok), ok
}

func asFn(file string, f codenav.Func, ok bool) fn {
	if !ok {
		return fn{}
	}
	return fn{file: file, from: f.From, to: f.To, name: f.Name, calls: f.Calls}
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
		for _, e := range l.Exts {
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
			res[l] = l.Pattern("")
		}
		name := l.Defined(line, res[l])
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
	for _, e := range l.Exts {
		args = append(args, "*"+e)
	}
	hits, err := git(r.repo, append(args, ":(exclude)vendor/**", ":(exclude)**/node_modules/**", ":(exclude)third_party/**")...)
	if err != nil {
		return nil
	}
	re := l.Pattern(f.name)
	var out []fn
	seen := map[string]bool{f.key(): true}
	for _, h := range strings.Split(strings.TrimSpace(hits), "\n") {
		file, n, line, ok := hit(h, r.head)
		if !ok || pathglob.Any(r.skip, file) || (file == f.file && n >= f.from && n <= f.to) || l.Defined(line, re) != "" || !visible(f, file) {
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
	var same, near, far []fn
	for _, d := range defs {
		switch {
		case !visible(d, file):
		case d.file == file:
			same = append(same, d)
		case path.Dir(d.file) == path.Dir(file):
			near = append(near, d)
		default:
			far = append(far, d)
		}
	}
	switch {
	case len(same) > 0:
		return same[:1], false
	case len(near) > 0:
		return near[:min(len(near), 2)], false
	case len(far) <= 2:
		return far, false
	}
	return nil, true
}

// visible says whether a Go function can be named from a file: an
// unexported one only from its own package's folder. Other languages: yes.
func visible(f fn, from string) bool {
	if languageOf(f.file).Body != "go" || f.name == "" || strings.ToUpper(f.name[:1]) == f.name[:1] {
		return true
	}
	return path.Dir(f.file) == path.Dir(from)
}

// namedIn are the names a text points to, read as the product owner and
// the tester read an issue's (codepick.NamedIn).
func namedIn(text string) (code, capital, weak []string) { return codepick.NamedIn(text) }

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
				lang = l.Name
			}
			w := around(file, at, fmt.Sprintf("the lines around %s, no function found around it (%s)", what, lang))
			if l := languageOf(file); l != nil { // what those lines call stands for their function's calls
				w.calls = callsIn(l, r.lines(file)[w.from-1:w.to], "")
			}
			return w, true
		case shown[g.key()]:
			return g, true
		case g.lines() > left:
			around(file, at, fmt.Sprintf("the lines around %s, in `%s` (lines %d to %d, longer than judge-lines-max)", what, g.name, g.from, g.to))
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
	named, capital, weak := namedIn(f.Title + ". " + f.Why + ". " + f.Fix)
	add(append(named, capital...), nil, "named by the finding")
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
	add(weak, nil, "named by the finding") // last: a word opening a sentence may be English
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

// reaches is the line of a removal that a kept line's quote, lines p of
// path at head, depends on (#357): a line the change removed above it, in
// the same function, that tested or assigned a name the quote uses. A
// guard removed exposes its uses further down, past the lines beside it;
// a name, not the whole function, so a defect from before the change on
// other names, or above, stays outside it. 0: none. No agent decides it.
func (c Change) reaches(repo, head, path string, p Place) int {
	removed := c.Removed[path]
	if len(removed) == 0 {
		return 0
	}
	r := &reacher{repo: repo, head: head, files: map[string][]string{}}
	in, ok := r.enclosing(path, p.From)
	lines := r.lines(path)
	if !ok || p.To > in.to || p.To > len(lines) {
		return 0
	}
	l := languageOf(path)
	hash := l.HashComments // a `#` opens a comment
	used := map[string]bool{}
	for _, line := range lines[p.From-1 : p.To] {
		for _, n := range names(stripCode(line, hash)) {
			used[n] = true
		}
	}
	for _, rm := range removed {
		if rm.At < in.from || rm.At > in.to || p.From <= rm.At { // a removal of its function, above the quote
			continue
		}
		for _, n := range guarded(stripCode(rm.Text, hash)) {
			if used[n] {
				return rm.At
			}
		}
	}
	return 0
}

var (
	nameRe = regexp.MustCompile(`[A-Za-z_$][\w$]*`)
	// testRe opens a condition: the names after it are tested.
	testRe = regexp.MustCompile(`\b(?:if|elif|elsif|while|until|unless|for|foreach|switch|case|when|guard|assert)\b`)
	// setRe is an assignment: `:=`, `=` alone or after an operator, `++`,
	// `--`; `==`, `!=`, `<=` and `>=` are told apart by the character
	// before (setAt), `=>` by the one after.
	setRe = regexp.MustCompile(`:=|\+\+|--|(?:[-+*/%&|^]|<<|>>|&&|\|\||\?\?)?=(?:[^=>]|$)`)
	// indexRe is an index or a key, `[i]`: what it names is read, not set.
	indexRe = regexp.MustCompile(`\[[^\]]*\]`)
)

// setAt is where a line's first assignment starts; -1: none.
func setAt(code string) int {
	for _, m := range setRe.FindAllStringIndex(code, -1) {
		if code[m[0]] == '=' && m[0] > 0 && strings.ContainsRune("<>!=", rune(code[m[0]-1])) {
			continue // a comparison
		}
		return m[0]
	}
	return -1
}

// notNames are the words of code that name nothing a line tests or sets:
// keywords, constants, the types a declaration opens with.
var notNames = map[string]bool{}

func init() {
	for _, w := range strings.Fields(`if else elif elsif while until unless for foreach in of switch case when
		guard assert default return break continue goto do then fi esac and or not is
		nil null None true false True False undefined var let const val mut auto func function def fn fun
		lambda go defer select range new delete typeof instanceof sizeof await async yield this self super
		throw raise try catch except finally int string bool float double char long short byte void local my`) {
		notNames[w] = true
	}
	for w := range notCalls {
		notNames[w] = true
	}
}

// names are the names a line of code uses, in order, once each: a word
// neither a keyword, nor after a dot (a field), nor before a parenthesis
// (a call).
func names(code string) []string {
	var out []string
	for _, m := range nameRe.FindAllStringIndex(code, -1) {
		w := code[m[0]:m[1]]
		after := strings.TrimLeft(code[m[1]:], " \t")
		if notNames[w] || w == "_" || m[0] > 0 && code[m[0]-1] == '.' || strings.HasPrefix(after, "(") || slices.Contains(out, w) {
			continue
		}
		out = append(out, w)
	}
	return out
}

// guarded are the names a removed line tested or assigned (#357): those
// after the keyword opening a condition (`if p < 1`), and those an
// assignment sets (`total += v`, `a, b := f()`, `items[i] = x`); a
// declaration's type among them, which cannot be told from its name
// without a parser.
func guarded(code string) []string {
	var out []string
	if m := testRe.FindStringIndex(code); m != nil {
		out = append(out, names(code[m[1]:])...)
	}
	if at := setAt(code); at >= 0 {
		out = append(out, names(indexRe.ReplaceAllString(code[:at], ""))...)
	}
	return out
}
