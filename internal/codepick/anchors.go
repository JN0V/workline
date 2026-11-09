package codepick

import (
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/codenav"
)

// The files an issue is about by its words, not only by a name defined
// once (#1152): the component its title names (`WebUI: …` is WebUI.h),
// and the code holding a route it writes (`/api/components/enable`). A
// name defined in many places (`begin()`, `shutdown()`) is read there,
// the others counted, however few they are.

// stemsMax bounds the files one word of the title anchors: past it, the
// word names no component (`main`).
const stemsMax = 3

// routesMax bounds the files a route's lines are read in: past it, the
// route is everywhere (`/`), it anchors nothing.
const routesMax = 3

var (
	// titlePrefix is the word a title opens on before a colon: the
	// component it is about, `WebUI: switching on…`, `codepick: …`.
	titlePrefix = regexp.MustCompile(`^\s*([A-Za-z_][\w-]*)\s*:`)
	// routeRe is a route a text writes: a path from the root, two parts
	// at least, `/api/components/enable`; a file's path is not one.
	routeRe = regexp.MustCompile("(?:^|[\\s`'\"(])(/[A-Za-z_][\\w.-]*(?:/[\\w.{}:-]+)+)")
)

// commitTypes open a title as a commit's do (`fix: …`): a kind of
// change, no component.
var commitTypes = map[string]bool{"fix": true, "feat": true, "docs": true, "test": true, "refactor": true,
	"perf": true, "build": true, "ci": true, "chore": true, "revert": true, "style": true, "bug": true}

// titleStems are the files of the project's own code whose name, the
// extension off, is a word of the title written as code (`WebUI`,
// `RemoteConsole`, `NTP`) or the word it opens on before a colon: what
// the issue is about. A word naming more than stemsMax files names none.
func (p *Picker) titleStems(a Ask) []string {
	title, _, _ := strings.Cut(a.Text, "\n")
	var words []string
	if m := titlePrefix.FindStringSubmatch(title); m != nil && !commitTypes[strings.ToLower(m[1])] && !p.notName(m[1]) {
		words = append(words, m[1])
	}
	code, _, _ := NamedIn(title)
	for _, w := range code {
		if !p.notName(w) && !slices.Contains(words, w) {
			words = append(words, w)
		}
	}
	var out []string
	for _, w := range words {
		var fs []string
		for _, f := range p.files {
			base := path.Base(f)
			if strings.TrimSuffix(base, path.Ext(base)) == w && p.own(a, f) {
				fs = append(fs, f)
			}
		}
		if len(fs) <= stemsMax {
			out = append(out, fs...)
		}
	}
	return out
}

// own says whether a file is the project's own code: a language known,
// not a test, an example or a tool, nor skipped.
func (p *Picker) own(a Ask, f string) bool {
	return codenav.Of(f) != nil && !a.Tests.Aside(f) && !slices.Contains(a.Skip, f)
}

// routes are the routes a text writes, each once.
func routes(text string) []string {
	var out []string
	for _, m := range routeRe.FindAllStringSubmatch(text, -1) {
		r := strings.TrimRight(m[1], ".,;:")
		if path.Ext(r) == "" && !slices.Contains(out, r) {
			out = append(out, r)
		}
	}
	return out
}

// routeLines are the pieces of the project's own code a route's lines
// lie in, and their files: the function around each, else, a comment
// naming it, the definitions it heads, else the lines around. A route
// in more than routesMax files gives none.
func (p *Picker) routeLines(a Ask, rs []string) (pieces []routePiece, files []string) {
	hits := p.grep(rs)
	for _, r := range rs {
		var in []string
		var got []routePiece
		for _, h := range hits[r] {
			if !p.own(a, h.file) {
				continue
			}
			if !slices.Contains(in, h.file) {
				in = append(in, h.file)
			}
			lines := p.Lines(h.file)
			if codenav.IsComment(h.line) {
				if gs := headed(h.file, lines, h.n); len(gs) > 0 {
					for _, g := range gs {
						got = append(got, routePiece{h.file, g.From, g.To, "`" + g.Name + "`, under a comment naming `" + r + "`"})
					}
					continue
				}
			}
			if g, ok := codenav.Enclosing(h.file, lines, h.n); ok && g.To-g.From < funcMax {
				got = append(got, routePiece{h.file, g.From, g.To, "`" + g.Name + "`, naming `" + r + "`"})
				continue
			}
			got = append(got, routePiece{h.file, max(h.n-around, 1), min(h.n+around, len(lines)), "naming `" + r + "`"})
		}
		if len(in) > 0 && len(in) <= routesMax {
			pieces = append(pieces, got...)
			for _, f := range in {
				if !slices.Contains(files, f) {
					files = append(files, f)
				}
			}
		}
	}
	return pieces, files
}

// routePiece is a piece of code a route names.
type routePiece struct {
	file     string
	from, to int
	why      string
}

// headedMax is the lines under a comment its definitions start in.
const headedMax = 12

// headed are the functions defined in the lines under a comment, up to
// headedMax: what the comment says, `// Logic for API /api/x` above the
// function that does it, a type between them aside.
func headed(file string, lines []string, at int) []codenav.Func {
	var out []codenav.Func
	for n := at + 1; n <= min(at+headedMax, len(lines)); n++ {
		if g, ok := codenav.Enclosing(file, lines, n); ok && g.From >= at && g.To-g.From < funcMax {
			if !slices.ContainsFunc(out, func(o codenav.Func) bool { return o.From == g.From }) {
				out = append(out, g)
			}
		}
	}
	return out
}
