// Package codepick gives a role the code an issue names, found the same
// way in any language (ADR-0063 §8): the product owner, the tester and
// the reviewer read an issue's code by it, not each by its own cut. What
// is given, in order, under the role's budget, the cut said:
//
//   - the lines an issue names after a file (`a.go:12`, "lines 6353 to
//     6353"), with the function they lie in, else the lines around;
//   - for each name it writes as code, `git grep -w`: each definition,
//     whole (codenav, docs/research/code-navigation.md); those in the files
//     it names first, then the project's own, its tests' and examples'
//     copies last;
//   - the files it names, whole when they fit, else their first lines;
//   - the other lines naming a name, the rarer the name the sooner: the
//     function they lie in, else the lines around;
//   - room for one existing test, the nearest to the code named;
//   - one step away, as paths only: the files the code given includes or
//     imports, and those defining the types it derives from.
package codepick

import (
	"fmt"
	"os/exec"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/codenav"
	"github.com/JN0V/workline/internal/pathglob"
	"github.com/JN0V/workline/internal/work"
)

const (
	// around is the lines given either side of a line named when no
	// function holds it.
	around = 15
	// near is the lines given either side of a line naming a name, no
	// function found around it.
	near = 3
	// funcMax is the longest function given for a line that names a name:
	// past it, the lines around.
	funcMax = 80
	// usesMax is the places naming one name, outside its definitions,
	// given at most: a name used everywhere gives a few, not every file.
	usesMax = 6
	// useMax is the longest function given for a place naming a name:
	// past it, the lines around.
	useMax = 40
	// defsMax is the definitions of one name given at most, the
	// project's own; the rest counted.
	defsMax = 12
	// asideMax is the copies of a definition in tests, examples and
	// tools given at most, the rest counted.
	asideMax = 2
	// commentMax is the lines of a definition's comment given above it.
	commentMax = 20
	// namedMax bounds the files a text names.
	namedMax = 12
	// linkedMax bounds the paths one step away.
	linkedMax = 12
	// testShare: 1 in testShare of the budget kept for a test when the
	// code fills it (#1059).
	testShare = 5
	// usesShare: 1 in usesShare of the budget at most for the places
	// naming a name that are not its definition.
	usesShare = 3
)

// Ask is what a role asks for.
type Ask struct {
	Text  string   // what names the code: the title, the Need, the Steps, the Scope, a person's comments
	Paths []string // files, folders or patterns named outright, the Scope's or the sources: given first
	Skip  []string // files never given: the file an issue was imported from
	Tests []string // test-paths: one existing test given room, test files' definitions ranked as copies
	// Budget is the lines given in all; BudgetName the setting it comes
	// from, said in the cut.
	Budget     int
	BudgetName string
}

// Piece is a span of a file's lines, from 1, and why it is given.
type Piece struct {
	From, To int
	Why      string
}

// File is a file of the pick and what of it is given.
type File struct {
	Path   string
	Lines  []string // the whole file
	Given  []Piece  // in order, merged; one from 1 to its end: the file whole
	Cut    int      // a piece the budget ran out in goes on to this line; 0: none
	Named  bool     // the text names the file itself
	Test   bool     // an existing test, given to write alike
	Unread []Piece  // what it names in the file, past the budget
	Above  bool     // what it names given for an earlier ask of the same task
}

// Whole says whether the file is given whole.
func (f File) Whole() bool {
	return len(f.Given) == 1 && f.Given[0].From == 1 && f.Given[0].To == len(f.Lines)
}

// GivenLines are the lines given of the file.
func (f File) GivenLines() int {
	n := 0
	for _, p := range f.Given {
		n += p.To - p.From + 1
	}
	return n
}

// Pick is what a role is given.
type Pick struct {
	Files   []File   // the code, then the test, in order
	Names   []string // the names read as code
	Many    []string // names with more definitions than given: "`f` (9 definitions, 2 given)"
	Linked  []string // one step away, as paths: "`a.h` (included)"
	Left    int      // the budget left
	Kept    int      // the lines kept for the test, the code filling the budget
	Unshown []string // names whose other places were past usesMax or the budget: "`f` (4 more places)"
	Ask     Ask
}

// Paths are the files given, here or above, in order, the test aside.
func (p Pick) Paths() []string {
	var out []string
	for _, f := range p.Files {
		if !f.Test && (len(f.Given) > 0 || f.Above) {
			out = append(out, f.Path)
		}
	}
	return out
}

// Named are the files the pick holds, given or not, the test aside.
func (p Pick) Named() []string {
	var out []string
	for _, f := range p.Files {
		if !f.Test {
			out = append(out, f.Path)
		}
	}
	return out
}

// Anchors are the files the tests nearest the code are looked for by:
// those the text names, else those holding what it names.
func (p Pick) Anchors() []string {
	var named, all []string
	for _, f := range p.Files {
		switch {
		case f.Test:
		case f.Named:
			named = append(named, f.Path)
			fallthrough
		default:
			all = append(all, f.Path)
		}
	}
	if len(named) > 0 {
		return named
	}
	return all
}

// Test is the test given, nil when none.
func (p Pick) Test() *File {
	for i := range p.Files {
		if p.Files[i].Test {
			return &p.Files[i]
		}
	}
	return nil
}

// Picker reads one commit of a repository, each file and each name once:
// a task holding several issues asks it once an issue, and a span given
// for one is not given again for the next.
type Picker struct {
	repo, rev string
	files     []string
	tracked   map[string]bool
	lines     map[string][]string
	hits      map[string][]hit
	given     map[string][]Piece
}

type hit struct {
	file string
	n    int
	line string
}

// New reads the files of a repository's commit rev ("" is HEAD).
func New(repo, rev string) (*Picker, error) {
	if rev == "" {
		rev = "HEAD"
	}
	out, err := exec.Command("git", "-C", repo, "ls-tree", "-r", "-z", "--name-only", rev).Output()
	if err != nil {
		return nil, fmt.Errorf("the files of %s: %v", rev, err)
	}
	p := &Picker{repo: repo, rev: rev, tracked: map[string]bool{}, lines: map[string][]string{},
		hits: map[string][]hit{}, given: map[string][]Piece{}}
	for _, f := range strings.Split(strings.TrimSuffix(string(out), "\x00"), "\x00") {
		if f != "" {
			p.files = append(p.files, f)
			p.tracked[f] = true
		}
	}
	return p, nil
}

// Files are the commit's files.
func (p *Picker) Files() []string { return p.files }

// Lines are a file's lines at the commit; nil when it holds none.
func (p *Picker) Lines(file string) []string {
	if l, ok := p.lines[file]; ok {
		return l
	}
	var l []string
	if p.tracked[file] {
		if out, err := exec.Command("git", "-C", p.repo, "show", p.rev+":"+file).Output(); err == nil {
			l = strings.Split(strings.TrimRight(string(out), "\n"), "\n")
		}
	}
	p.lines[file] = l
	return l
}

var (
	// lineRef is a line or a span after a file: `a.go:12`, `a.go:12-30`.
	lineRef = regexp.MustCompile(`([\w./-]+\.\w+):(\d+)(?:-(\d+))?\b`)
	// linesAfter is a line or a span named in words after a file, on the
	// same line: "docs/x.md, lines 6353 to 6353", "a.go line 12".
	linesAfter = regexp.MustCompile(`(?i)([\w./-]+\.\w+)\W{0,4}\s*(?:at\s+|on\s+)?lines?\s+(\d+)(?:\s*(?:to|-|–|through)\s*(\d+))?`)
	// pathLike is a path a text names: `src/a.go`, a folder `internal/x`,
	// a file named alone `csv.go`.
	pathLike = regexp.MustCompile(`[\w.-]+(?:/[\w.-]+)+|[\w-]+\.[A-Za-z]{1,5}\b`)
)

// notNames are words a text writes as code that name no definition worth
// searching: keywords and the language's own types.
var notNames = map[string]bool{}

func init() {
	for _, w := range strings.Fields(`if else elif while for foreach switch case return break continue
		nil null None true false True False undefined var let const val auto func function def fn fun
		new delete this self super int bool float double char long short byte void string String std
		static inline virtual override public private protected class struct enum namespace template
		typename unsigned signed size_t uint8_t uint16_t uint32_t uint64_t int8_t int16_t int32_t int64_t
		include import from package async await`) {
		notNames[w] = true
	}
}

var (
	// imported is the engine's mark of the file an issue was imported
	// from, and provenance the line it ends such an issue with.
	imported   = regexp.MustCompile(`<!-- workline:import=([^\s>]+?):[0-9a-f]+ -->`)
	provenance = regexp.MustCompile(`(?m)^Opened from .* by the [\w -]+ role\.\r?$`)
	engineMark = regexp.MustCompile(`<!-- workline:[^>]*-->`)
)

// clean leaves out of an ask's text what the engine wrote there: the file
// an issue was imported from is where it was written, never its code
// (#548, #1063).
func clean(a Ask) Ask {
	for _, m := range imported.FindAllStringSubmatch(a.Text, -1) {
		a.Skip = append(slices.Clone(a.Skip), m[1])
	}
	a.Text = engineMark.ReplaceAllString(provenance.ReplaceAllString(a.Text, ""), "")
	return a
}

// Tracked says whether the commit holds a file.
func (p *Picker) Tracked(f string) bool { return p.tracked[f] }

// namedFiles are the files an ask names: outright, then in its text, a
// file named alone (`csv.go`) when one file of the commit has that name.
func (p *Picker) namedFiles(a Ask) []string {
	var named []string
	add := func(f string) {
		if f != "" && !slices.Contains(a.Skip, f) && !slices.Contains(named, f) && len(named) < namedMax && p.tracked[f] {
			named = append(named, f)
		}
	}
	for _, w := range a.Paths {
		for _, f := range p.wordFiles(strings.TrimSpace(work.LineOff(w)), a.Tests) {
			add(f)
		}
	}
	for _, m := range pathLike.FindAllString(a.Text, -1) {
		w := strings.TrimRight(strings.TrimPrefix(strings.Trim(m, "`'\""), "./"), ".,;:/")
		if f := work.InRepo(w, p.files); f != "" && (strings.Contains(w, "/") || path.Ext(w) != "") {
			add(f)
		}
	}
	return named
}

// codeNames are the names a text writes as code worth a search: an
// acronym (`CSV`, `MQTT`) names a format or a protocol as often as code,
// a constant holds an underscore.
func codeNames(text string) []string {
	code, _, _ := NamedIn(text)
	var names []string
	for _, n := range code {
		if !notNames[n] && len(n) > 2 && identRe.MatchString(n) && (strings.ToUpper(n) != n || strings.Contains(n, "_")) {
			names = append(names, n)
		}
	}
	return names
}

// defines is the function a hit's line defines by the name, its comment
// given up to commentMax lines above it.
func (p *Picker) defines(h hit, name string, pats map[*codenav.Language]*regexp.Regexp) (codenav.Func, bool) {
	l := codenav.Of(h.file)
	if l == nil {
		return codenav.Func{}, false
	}
	if pats[l] == nil {
		pats[l] = l.Pattern(name)
	}
	if l.Defined(h.line, pats[l]) != name {
		return codenav.Func{}, false
	}
	g, ok := codenav.At(h.file, p.Lines(h.file), h.n, name)
	g.From = max(g.From, h.n-commentMax) // its comment, not a page of them
	return g, ok
}

// Where are the files an ask names, and those defining a name it writes
// as code, tests, examples and tools aside: what an issue is about, with
// no line planned.
func (p *Picker) Where(a Ask) []string {
	a = clean(a)
	out := p.namedFiles(a)
	names := codeNames(a.Text)
	hits := p.grep(names)
	for _, n := range names {
		pats := map[*codenav.Language]*regexp.Regexp{}
		for _, h := range hits[n] {
			if slices.Contains(out, h.file) || slices.Contains(a.Skip, h.file) || aside(h.file) || (len(a.Tests) > 0 && pathglob.Any(a.Tests, h.file)) {
				continue
			}
			if _, ok := p.defines(h, n, pats); ok {
				out = append(out, h.file)
			}
		}
	}
	return out
}

// Pick finds the code an ask names and plans what is given of it.
func (p *Picker) Pick(a Ask) Pick {
	a = clean(a)
	pk := Pick{Ask: a}
	budget := a.Budget
	if budget <= 0 {
		budget = 1500
	}
	byPath := map[string]*File{}
	var order []string
	file := func(f string) *File {
		if byPath[f] == nil {
			byPath[f] = &File{Path: f, Lines: p.Lines(f)}
			order = append(order, f)
		}
		return byPath[f]
	}
	skip := func(f string) bool { return slices.Contains(a.Skip, f) }
	isTest := func(f string) bool { return len(a.Tests) > 0 && pathglob.Any(a.Tests, f) }
	named := p.namedFiles(a)
	for _, f := range named {
		file(f).Named = true
	}

	type want struct {
		file     string
		from, to int
		why      string
		cut      bool // may be given in part
	}
	var wants []want
	// 1. The lines named after a file.
	spans := map[string][][2]int{}
	for _, re := range []*regexp.Regexp{lineRef, linesAfter} {
		for _, m := range re.FindAllStringSubmatch(a.Text, -1) {
			f := work.InRepo(m[1], p.files)
			if f == "" || skip(f) {
				continue
			}
			from, _ := strconv.Atoi(m[2])
			to := from
			if m[3] != "" {
				to, _ = strconv.Atoi(m[3])
			}
			spans[f] = append(spans[f], [2]int{min(from, to), max(from, to)})
		}
	}
	for _, f := range sortedKeys(spans) {
		lines := p.Lines(f)
		for _, s := range spans[f] {
			if s[0] < 1 || s[0] > len(lines) {
				continue // a line past the file's end: written for another version of it
			}
			s[1] = min(s[1], len(lines))
			file(f).Named = true
			for _, n := range []int{s[0], s[1]} {
				if g, ok := codenav.Enclosing(f, lines, n); ok && g.To-g.From < funcMax {
					wants = append(wants, want{f, g.From, g.To, fmt.Sprintf("`%s`, around the line named", g.Name), true})
				} else {
					wants = append(wants, want{f, max(n-around, 1), min(n+around, len(lines)), "the lines named", true})
				}
			}
			wants = append(wants, want{f, s[0], s[1], "the lines named", true})
		}
	}
	// 2. The names, by their hits.
	names := codeNames(a.Text)
	pk.Names = names
	hits := p.grep(names)
	type found struct {
		defs, aside []want
		uses        []want
	}
	byName := map[string]*found{}
	for _, n := range names {
		byName[n] = &found{}
	}
	for _, n := range names {
		fd := byName[n]
		pats := map[*codenav.Language]*regexp.Regexp{}
		seen := map[string]bool{}
		for _, h := range hits[n] {
			if skip(h.file) {
				continue
			}
			l := codenav.Of(h.file)
			lines := p.Lines(h.file)
			if l != nil {
				if g, ok := p.defines(h, n, pats); ok {
					w := want{h.file, g.From, g.To, fmt.Sprintf("`%s`, defined", n), true}
					if isTest(h.file) || aside(h.file) {
						fd.aside = append(fd.aside, w)
					} else {
						fd.defs = append(fd.defs, w)
					}
					continue
				}
				if g, ok := codenav.Enclosing(h.file, lines, h.n); ok && g.To-g.From < useMax {
					key := fmt.Sprintf("%s:%d", h.file, g.From)
					if g.Name != n && !seen[key] { // its own comment: the definition, counted above
						seen[key] = true
						fd.uses = append(fd.uses, want{h.file, g.From, g.To, fmt.Sprintf("`%s`, naming `%s`", g.Name, n), false})
					}
					continue
				}
			}
			fd.uses = append(fd.uses, want{h.file, max(h.n-near, 1), min(h.n+near, len(lines)), fmt.Sprintf("naming `%s`", n), false})
		}
	}
	// A place in the project's own code first, a doc last.
	for _, fd := range byName {
		slices.SortStableFunc(fd.uses, func(x, y want) int { return useRank(x.file) - useRank(y.file) })
	}
	// The rarer a name, the more it says: its pieces come first.
	rank := slices.Clone(names)
	slices.SortStableFunc(rank, func(x, y string) int { return len(hits[x]) - len(hits[y]) })
	inNamed := func(w want) bool { return slices.Contains(named, w.file) }
	// 2a. Definitions in the files named, then the project's own. A name
	// defined in a file named is that one: one of the same name
	// elsewhere is another function, counted, not given (#425).
	here := map[string]bool{}
	for _, n := range rank {
		here[n] = slices.ContainsFunc(append(slices.Clone(byName[n].defs), byName[n].aside...), inNamed)
	}
	for _, keepNamed := range []bool{true, false} {
		for _, n := range rank {
			for i, w := range byName[n].defs {
				if inNamed(w) == keepNamed && (inNamed(w) || i < defsMax && !here[n]) {
					wants = append(wants, w)
				}
			}
		}
	}
	// 2b. A name no function defines (a member, a constant): the lines
	// naming it in the files named, a few, as its definition stands
	// among them (#1056).
	usedIn := map[string]int{}
	for _, n := range rank {
		if len(byName[n].defs)+len(byName[n].aside) > 0 {
			continue
		}
		for _, w := range byName[n].uses {
			if inNamed(w) && usedIn[n] < usesMax {
				usedIn[n]++
				wants = append(wants, w)
			}
		}
	}
	// 2c. Copies in tests, examples and tools, a few.
	for _, n := range rank {
		fd := byName[n]
		given := 0
		for i, w := range append(slices.Clone(fd.defs), fd.aside...) {
			if inNamed(w) || !here[n] && (i < min(len(fd.defs), defsMax) || i >= len(fd.defs) && i-len(fd.defs) < asideMax) {
				given++
			}
		}
		for i, w := range fd.aside {
			if inNamed(w) || !here[n] && i < asideMax {
				wants = append(wants, w)
			}
		}
		switch total := len(fd.defs) + len(fd.aside); {
		case given == total:
		case here[n]:
			pk.Many = append(pk.Many, fmt.Sprintf("`%s` (defined in the files named; %d more of the same name elsewhere, not given)", n, total-given))
		default:
			pk.Many = append(pk.Many, fmt.Sprintf("`%s` (%d definitions, %d of them not given, most in tests, examples or tools)", n, total, total-given))
		}
	}
	// The test kept room for, before the code fills the budget.
	var test string
	kept := 0
	if len(a.Tests) > 0 {
		// Near the files named; none named, near the definitions found.
		anchor := slices.Clone(named)
		for _, w := range wants {
			if len(named) == 0 && !slices.Contains(anchor, w.file) {
				anchor = append(anchor, w.file)
			}
		}
		ts := NearestTests(p.files, a.Tests, anchor)
		if len(ts) == 0 && len(anchor) > 0 { // none near: the project's first, its style
			ts = slices.DeleteFunc(slices.Clone(p.files), func(f string) bool { return !pathglob.Any(a.Tests, f) })
		}
		if len(ts) > 0 {
			test = ts[0]
			kept = min(budget/testShare, len(p.Lines(test)))
		}
	}
	// The test's share is kept while the code is planned, given back
	// after: said only when the code wanted it.
	left := budget - kept
	short := false // something named was cut or left out
	// give plans a piece; false when nothing of it fits.
	give := func(w want) bool {
		f := file(w.file)
		if covered(p.given[w.file], w.from, w.to) || covered(f.Given, w.from, w.to) {
			return true
		}
		n := w.to - w.from + 1
		if n > left {
			short = true
			if !w.cut || left <= 0 {
				if w.cut {
					f.Unread = append(f.Unread, Piece{w.from, w.to, w.why})
				}
				return false
			}
			f.Cut = w.to
			w.to = w.from + left - 1
			n = left
		}
		f.Given = merge(f.Given, Piece{w.from, w.to, w.why})
		left -= n
		return true
	}
	for _, w := range wants {
		give(w)
	}
	// 3. The files named, whole when what they add fits; else, with
	// nothing of them given, their first lines.
	for _, f := range named {
		g := file(f)
		switch extra := len(g.Lines) - g.GivenLines(); {
		case len(p.given[f]) > 0 && len(g.Given) == 0:
			g.Above = true // given for an earlier issue: not twice
		case extra <= left:
			left -= extra
			g.Given, g.Cut, g.Unread = []Piece{{1, len(g.Lines), "named"}}, 0, nil
		case !g.Whole():
			short = true
		}
	}
	for _, f := range named {
		g := file(f)
		if len(g.Given) == 0 && len(g.Unread) == 0 && !g.Above && left > 0 && len(g.Lines) > 0 {
			k := min(len(g.Lines), left)
			g.Given = []Piece{{1, k, "its first lines"}}
			left -= k
		}
	}
	// 4. The other places naming a name, the rarer first, a few each,
	// those in the files named first, a third of the budget at most:
	// what defines the code comes before what uses it.
	usesLeft := budget / usesShare
	for _, n := range rank {
		dropped := 0
		for _, inFile := range []bool{true, false} {
			for _, w := range byName[n].uses {
				if inNamed(w) != inFile || covered(file(w.file).Given, w.from, w.to) {
					continue
				}
				size := w.to - w.from + 1
				if usedIn[n] >= usesMax || size > usesLeft || !give(w) {
					dropped++
					continue
				}
				usedIn[n]++
				usesLeft -= size
			}
		}
		if dropped > 0 {
			pk.Unshown = append(pk.Unshown, fmt.Sprintf("`%s` (%d more places)", n, dropped))
		}
	}
	// 5. The test, apart from the code: its own file.
	var tf *File
	if test != "" {
		left += kept
		if short {
			pk.Kept = kept
		}
		tf = &File{Path: test, Lines: p.Lines(test), Test: true}
		if k := min(len(tf.Lines), left, max(kept, budget/testShare)); k > 0 {
			tf.Given = []Piece{{1, k, "a test to write alike"}}
			left -= k
		}
	}
	for _, f := range order {
		g := byPath[f]
		if g.Test {
			continue
		}
		if len(g.Given) == 0 && !g.Named && len(g.Unread) == 0 {
			continue
		}
		pk.Files = append(pk.Files, *g)
		p.given[f] = append(p.given[f], g.Given...)
	}
	if tf != nil {
		pk.Files = append(pk.Files, *tf)
	}
	pk.Linked = p.linked(pk, a.Tests)
	pk.Left = left
	return pk
}

// useRank orders the places naming a name: the project's own code, then
// its tests', examples' and tools', then docs and files no language is
// known for.
func useRank(f string) int {
	switch {
	case codenav.Of(f) == nil:
		return 2
	case aside(f):
		return 1
	}
	return 0
}

// aside says whether a file is beside the project's own code: its tests,
// examples, tools.
func aside(f string) bool { return asideFolder.MatchString(f) }

var asideFolder = regexp.MustCompile(`(?i)(^|/)(tests?|examples?|tools|mocks?|testdata|spec)/`)

// covered says whether pieces hold every line from to.
func covered(ps []Piece, from, to int) bool {
	for _, p := range ps {
		if p.From <= from && p.To >= to {
			return true
		}
	}
	return false
}

// merge adds a piece to sorted pieces, joining those that touch or lie a
// line apart: a blank between, given.
func merge(ps []Piece, q Piece) []Piece {
	ps = append(ps, q)
	slices.SortFunc(ps, func(a, b Piece) int { return a.From - b.From })
	var out []Piece
	for _, p := range ps {
		if k := len(out) - 1; k >= 0 && p.From <= out[k].To+2 {
			if p.To > out[k].To {
				out[k].To = p.To
			}
			if !strings.Contains(out[k].Why, p.Why) {
				out[k].Why += "; " + p.Why
			}
			continue
		}
		out = append(out, p)
	}
	return out
}

// grep finds each name as a word at the commit: one search for all,
// vendored code aside; each hit kept for the names its line holds.
func (p *Picker) grep(names []string) map[string][]hit {
	var ask []string
	for _, n := range names {
		if _, ok := p.hits[n]; !ok {
			ask = append(ask, n)
			p.hits[n] = nil
		}
	}
	if len(ask) > 0 {
		args := []string{"-C", p.repo, "grep", "-n", "-I", "-w", "-F"}
		for _, n := range ask {
			args = append(args, "-e", n)
		}
		args = append(args, p.rev, "--", ".", ":(exclude)vendor/**", ":(exclude)**/node_modules/**", ":(exclude)third_party/**")
		out, _ := exec.Command("git", args...).Output() // none found: git grep fails
		words := map[string]*regexp.Regexp{}
		for _, n := range ask {
			words[n] = regexp.MustCompile(`\b` + regexp.QuoteMeta(n) + `\b`)
		}
		for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			l = strings.TrimPrefix(l, p.rev+":")
			f, rest, ok := strings.Cut(l, ":")
			if !ok {
				continue
			}
			num, text, ok := strings.Cut(rest, ":")
			n, err := strconv.Atoi(num)
			if !ok || err != nil {
				continue
			}
			for _, name := range ask {
				if words[name].MatchString(text) {
					p.hits[name] = append(p.hits[name], hit{f, n, text})
				}
			}
		}
	}
	out := map[string][]hit{}
	for _, n := range names {
		out[n] = p.hits[n]
	}
	return out
}

// wordFiles are the files a word names: a file, the files a pattern
// matches, or a folder's own files, its tests aside when it holds code
// too (the tester's rule, #791, #801).
func (p *Picker) wordFiles(w string, tests []string) []string {
	notTest := func(fs []string) []string {
		if code := slices.DeleteFunc(slices.Clone(fs), func(f string) bool { return len(tests) > 0 && pathglob.Any(tests, f) }); len(code) > 0 {
			return code
		}
		return fs
	}
	if fs := work.Files(w, p.files); len(fs) > 0 {
		return notTest(fs)
	}
	dir := path.Clean(strings.TrimSuffix(strings.ReplaceAll(w, `\`, "/"), "/"))
	if dir == "." || strings.HasPrefix(dir, "../") || path.IsAbs(dir) {
		return nil
	}
	var out []string
	for _, f := range p.files {
		if path.Dir(f) == dir && (len(tests) == 0 || !pathglob.Any(tests, f)) {
			out = append(out, f)
		}
	}
	return out
}

func sortedKeys(m map[string][][2]int) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}
