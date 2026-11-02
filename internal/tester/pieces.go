package tester

// A Scope's file past what is left of code-lines-max is given by the
// functions the Scope names in it, whole, and those its line references
// lie in, not by its first lines: a function deep in a long file was cut
// and the tester sent the issue back (#908). The rest of the file is
// named, not shown.

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/codenav"
	"github.com/JN0V/workline/internal/work"
)

var (
	scopeWordRe = regexp.MustCompile("`([^`\\s]+)`")
	lineRefRe   = regexp.MustCompile(`^(.+):(\d+)(?:-(\d+))?$`)
	symbolRe    = regexp.MustCompile(`^[A-Za-z_][\w]*$`)
)

// lineAround is the lines given either side of a line a Scope names when
// no function is found around it.
const lineAround = 15

// anchors are what a Scope points at in its files: the names it writes in
// backticks that are no file nor path (`personIssue`, `Store.Post`,
// `run()`), and the lines or spans of lines it names after a file
// (`engine.go:2690`, `engine.go:100-400`), by file.
func anchors(scope string, files []string) (names []string, spans map[string][][2]int) {
	spans = map[string][][2]int{}
	for _, m := range scopeWordRe.FindAllStringSubmatch(scope, -1) {
		w := m[1]
		if r := lineRefRe.FindStringSubmatch(w); r != nil {
			if f := work.InRepo(r[1], files); f != "" {
				from, _ := strconv.Atoi(r[2])
				to := from
				if r[3] != "" {
					to, _ = strconv.Atoi(r[3])
				}
				spans[f] = append(spans[f], [2]int{min(from, to), max(from, to)})
			}
			continue
		}
		// A file, a pattern, or a path to a file not there yet
		// (`new/foo.go`): no function's name.
		if len(work.Files(w, files)) > 0 || strings.Contains(w, "/") || codenav.Of(w) != nil {
			continue
		}
		w = strings.TrimSuffix(w, "()")
		if i := strings.LastIndex(w, "."); i >= 0 {
			w = w[i+1:]
		}
		if symbolRe.MatchString(w) && !slices.Contains(names, w) {
			names = append(names, w)
		}
	}
	return names, spans
}

// pieces are the spans of a file's lines, from 1, that hold the functions
// names defines in it, and the lines at names with the functions their
// ends lie in, in order, merged.
func pieces(file string, lines []string, names []string, at [][2]int) [][2]int {
	var out [][2]int
	for _, n := range names {
		for _, f := range codenav.Named(file, lines, n) {
			out = append(out, [2]int{f.From, f.To})
		}
	}
	for _, s := range at {
		if s[0] < 1 || s[0] > len(lines) {
			continue // a line past the file's end: written for another version of it
		}
		s[1] = min(s[1], len(lines))
		out = append(out, s)
		for _, n := range []int{s[0], s[1]} {
			if f, ok := codenav.Enclosing(file, lines, n); ok {
				out = append(out, [2]int{f.From, f.To})
			} else {
				out = append(out, [2]int{max(n-lineAround, 1), min(n+lineAround, len(lines))})
			}
		}
	}
	slices.SortFunc(out, func(a, b [2]int) int { return a[0] - b[0] })
	var merged [][2]int
	for _, p := range out {
		if k := len(merged) - 1; k >= 0 && p[0] <= merged[k][1]+2 { // one line between: a blank, given
			merged[k][1] = max(merged[k][1], p[1])
			continue
		}
		merged = append(merged, p)
	}
	return merged
}

func spanLines(ps [][2]int) int {
	n := 0
	for _, p := range ps {
		n += p[1] - p[0] + 1
	}
	return n
}

// codeFile is a Scope's file and what of it is given.
type codeFile struct {
	path   string
	lines  []string
	pieces [][2]int // the spans the Scope names in it
	given  [][2]int // what is given: the whole file is one span from 1
	cut    [2]int   // a piece the budget ran out in: given to [0], it goes on to [1]
	unread [][2]int // the pieces past the budget, named
}

// planCode shares budget lines among the Scope's files: all whole when
// they fit; else the pieces the Scope names in each first, then, in
// order, a file whole when what it adds fits, a file with no piece by
// its first lines. Left is what remains for the tests beside.
func planCode(files []codeFile, budget int) int {
	total := 0
	for _, f := range files {
		total += len(f.lines)
	}
	if total <= budget {
		for i := range files {
			files[i].given = [][2]int{{1, len(files[i].lines)}}
		}
		return budget - total
	}
	left := budget
	for i := range files {
		for _, p := range files[i].pieces {
			if left <= 0 {
				files[i].unread = append(files[i].unread, p)
				continue
			}
			if end := p[0] + left - 1; end < p[1] {
				files[i].cut = [2]int{end, p[1]}
				p[1] = end
			}
			files[i].given = append(files[i].given, p)
			left -= p[1] - p[0] + 1
		}
	}
	for i := range files {
		f := &files[i]
		if len(f.pieces) == 0 {
			if left > 0 {
				f.given = [][2]int{{1, min(len(f.lines), left)}}
				left -= f.given[0][1]
			}
			continue
		}
		if extra := len(f.lines) - spanLines(f.given); extra <= left {
			f.given, f.cut, f.unread = [][2]int{{1, len(f.lines)}}, [2]int{}, nil
			left -= extra
		}
	}
	return left
}

// writeCodeFile writes what is given of a file: whole, its first lines,
// or the pieces the Scope names, the rest named.
func writeCodeFile(b *strings.Builder, f codeFile) {
	n := len(f.lines)
	switch {
	case len(f.given) == 1 && f.given[0] == [2]int{1, n}:
		fmt.Fprintf(b, "\n### `%s`\n\n````\n%s\n````\n", f.path, strings.Join(f.lines, "\n"))
	case len(f.pieces) == 0:
		k := f.given[0][1]
		fmt.Fprintf(b, "\n### `%s`\n\n````\n%s\n````\n(cut: the first %d lines given)\n", f.path, strings.Join(f.lines[:k], "\n"), k)
	default:
		var spans []string
		for _, p := range f.given {
			spans = append(spans, fmt.Sprintf("%d to %d", p[0], p[1]))
		}
		fmt.Fprintf(b, "\n### `%s` (%d lines: lines %s given, the functions and lines the %s names; the rest not shown)\n",
			f.path, n, strings.Join(spans, ", "), work.Name(work.Scope))
		for _, p := range f.given {
			fmt.Fprintf(b, "\nLines %d to %d:\n\n````\n%s\n````\n", p[0], p[1], strings.Join(f.lines[p[0]-1:p[1]], "\n"))
			if p[1] == f.cut[0] {
				fmt.Fprintf(b, "(cut, past code-lines-max: what it names goes on to line %d)\n", f.cut[1])
			}
		}
		for _, p := range f.unread {
			fmt.Fprintf(b, "\nNot given, past code-lines-max: lines %d to %d, which the %s names.\n", p[0], p[1], work.Name(work.Scope))
		}
	}
}
