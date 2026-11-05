package codepick

import (
	"maps"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/codenav"
)

// declare gives a file as its declarations, its pieces given shown
// whole, when they fit room lines and say less than the file whole;
// false when its language has no outline or they do not fit.
func (p *Picker) declare(f *File, why string, room int) bool {
	d := p.decls(f)
	if d == nil || len(d) > room || len(d) >= len(f.Lines) {
		return false
	}
	f.Decls, f.DeclsWhy = d, why
	f.Cut, f.Unread = 0, nil
	return true
}

// decls are a file's declarations, its pieces given shown whole; nil
// when its language has no outline.
func (p *Picker) decls(f *File) []string {
	o := codenav.Outline(f.Path, f.Lines)
	if o == nil {
		return nil
	}
	full := func(n int) bool {
		for _, g := range f.Given {
			if n >= g.From && n <= g.To {
				return true
			}
		}
		return false
	}
	// Blank lines out, but in the pieces shown whole: lines are the budget.
	return slices.DeleteFunc(o.Render(f.Lines, full), func(l string) bool { return strings.TrimSpace(l) == "" })
}

// closure gives the project's files the test and the code given include
// or import, a.Depth steps deep, each as its declarations while they fit:
// the test's own first, as it shows what builds today; then the others of
// every step together, the most used first. The rest stay paths, one
// step away. The budget left is returned.
func (p *Picker) closure(a Ask, test string, roots, order []string, byPath map[string]*File, file func(string) *File, left int) int {
	var code []string
	var given []*File
	for _, f := range order {
		// The files named outright or asked for, and those defining a
		// name: a place merely naming one is not the code the test calls.
		if g := byPath[f]; (len(g.Given) > 0 || len(g.Decls) > 0) && (slices.Contains(roots, f) || slices.ContainsFunc(g.Given, func(p Piece) bool { return strings.Contains(p.Why, ", defined") })) {
			code, given = append(code, f), append(given, g)
		}
	}
	usage := p.usage(a, test, given)
	seen := map[string]bool{test: true}
	for f := range byPath {
		seen[f] = true
	}
	why := map[string]string{}
	// reach are the files some files include or import, not seen yet.
	reach := func(from []string) []string {
		var out []string
		for _, f := range from {
			for _, im := range p.imports(f, a.Tests) {
				if !seen[im.file] && !slices.Contains(a.Skip, im.file) && len(p.Lines(im.file)) > 0 {
					seen[im.file] = true
					out = append(out, im.file)
					why[im.file] = im.why + " by `" + f + "`"
				}
			}
		}
		return out
	}
	var first []string
	if test != "" {
		first = reach([]string{test})
	}
	rest := reach(code)
	// A file two steps away is worth its lines when the declarations one
	// step away use its names: each step's use holds the last one's.
	use := map[string]map[string]bool{}
	for _, f := range append(slices.Clone(first), rest...) {
		use[f] = usage
	}
	front := append(slices.Clone(first), rest...)
	for d := 1; d < a.Depth && len(front) > 0; d++ {
		more := maps.Clone(usage)
		for _, f := range front {
			addWords(more, strings.Join(p.decls(&File{Path: f, Lines: p.Lines(f)}), "\n"))
		}
		front = reach(front)
		for _, f := range front {
			use[f] = more
		}
		rest, usage = append(rest, front...), more
	}
	// cost is the lines a file takes: its declarations, else whole.
	sizes := map[string]int{}
	cost := func(f string) int {
		if _, ok := sizes[f]; !ok {
			sizes[f] = len(p.Lines(f))
			if d := p.decls(&File{Path: f, Lines: p.Lines(f)}); d != nil {
				sizes[f] = min(len(d), sizes[f])
			}
		}
		return sizes[f]
	}
	first = rankUsed(first, use, p.Lines, cost, true)
	rest = rankUsed(rest, use, p.Lines, cost, false)
	for _, f := range append(first, rest...) {
		g := &File{Path: f, Lines: p.Lines(f)}
		switch {
		case cost(f) > left:
			continue
		case cost(f) >= len(g.Lines):
			g.Given = []Piece{{1, len(g.Lines), why[f]}}
		case p.declare(g, why[f], left):
		default:
			continue
		}
		left -= g.GivenLines()
		h := file(f)
		h.Given, h.Decls, h.DeclsWhy = g.Given, g.Decls, g.DeclsWhy
	}
	return left
}

// askedWhy is why a file the agent asked for is given.
const askedWhy = "asked for at the last run"

var identRe4 = regexp.MustCompile(`[A-Za-z_]\w{3,}`)

// addWords adds a text's names to words, each as written and lowercased.
func addWords(words map[string]bool, s string) {
	for _, w := range identRe4.FindAllString(s, -1) {
		words[w] = true
		words[strings.ToLower(w)] = true
	}
}

// usage are the words the issue, the test and the code given write:
// what a file included must declare to be worth its lines.
func (p *Picker) usage(a Ask, test string, code []*File) map[string]bool {
	words := map[string]bool{}
	add := func(s string) { addWords(words, s) }
	add(a.Text)
	for _, f := range code {
		for _, g := range f.Given { // the lines given, not the file's other code
			add(strings.Join(f.Lines[g.From-1:min(g.To, len(f.Lines))], "\n"))
		}
	}
	if test != "" {
		add(strings.Join(p.Lines(test), "\n"))
	}
	return words
}

// rankUsed orders files, the most used a line first, stably: a point for
// each name a file declares that the issue, the test or the code given
// write and no other of the files declares, a point for each word of its
// name they write (`Platform_Stub.h` for a test on the stub). With
// unused false, a file none of whose names are used is left out.
func rankUsed(files []string, use map[string]map[string]bool, lines func(string) []string, cost func(string) int, unused bool) []string {
	ids := map[string]map[string]bool{}
	count := map[string]int{}
	for _, f := range files {
		ids[f] = map[string]bool{}
		for _, w := range identRe4.FindAllString(strings.Join(lines(f), "\n"), -1) {
			if !ids[f][w] {
				ids[f][w] = true
				count[w]++
			}
		}
	}
	used, score := map[string]int{}, map[string]int{}
	for _, f := range files {
		usage := use[f]
		for w := range ids[f] {
			if count[w] == 1 && usage[w] {
				used[f]++
			}
		}
		score[f] = used[f]
		for _, w := range nameWords(f) {
			if usage[w] {
				score[f]++
			}
		}
	}
	out := slices.DeleteFunc(slices.Clone(files), func(f string) bool { return !unused && used[f] == 0 })
	density := func(f string) int { return score[f] * 1000 / max(cost(f), 1) }
	slices.SortStableFunc(out, func(x, y string) int { return density(y) - density(x) })
	return out
}

var camel = regexp.MustCompile(`[A-Z]+[a-z0-9]*|[a-z0-9]+`)

// nameWords are the words of a file's name, lowercased, 3 letters at
// least: `Platform_Stub.h` is platform, stub.
func nameWords(f string) []string {
	var out []string
	base := strings.TrimSuffix(path.Base(f), path.Ext(f))
	for _, w := range camel.FindAllString(base, -1) {
		if w = strings.ToLower(w); len(w) >= 3 && !slices.Contains(out, w) {
			out = append(out, w)
		}
	}
	return out
}
