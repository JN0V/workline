// Package codepick gives a role the code an issue names, found the same
// way in any language (ADR-0063 §8): the product owner, the tester and
// the reviewer read an issue's code by it. What it gives, in what order
// and under which budget: docs/spec/code-picker.md.
package codepick

import (
	"bytes"
	"fmt"
	"os/exec"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/codenav"
	"github.com/JN0V/workline/internal/pathglob"
	"github.com/JN0V/workline/internal/testdef"
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
	// callsMax is the functions one definition given calls, defined in
	// its file, given at most (#1129).
	callsMax = 3
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
	// fileBytes is what a file given weighs in bytes beside its lines:
	// its heading, why each piece, a role's lines on it (its last
	// commits): ten files of a line each are no ten lines (#1152).
	fileBytes = 400
)

// Ask is what a role asks for.
type Ask struct {
	Text  string   // what names the code: the title, the Need, the Steps, the Scope, a person's comments
	Paths []string // files, folders or patterns named outright, the Scope's or the sources: given first
	Skip  []string // files never given: the file an issue was imported from
	// Tests: the project's test files (testdef, ADR-0065), never its own
	// code, their definitions ranked as copies; zero, the generic ones.
	// TestRoom: one existing test given room, the tester's.
	Tests    testdef.Def
	TestRoom bool
	// Needs are the files the agent said it lacked at its last run: each
	// given whole, its first lines when it does not fit, right after the
	// definitions, before any file is given whole for being named.
	Needs []string
	// Depth is the steps of includes and imports followed from the test
	// and the code given, the test's first, each file given as its
	// declarations while they fit; a file named or asked for that does
	// not fit whole is given so too. 0: none, the paths listed only.
	Depth int
	// Budget is the lines given in all; BudgetName the setting it comes
	// from, said in the cut.
	Budget     int
	BudgetName string
	// Bytes bounds the code given in bytes too, its lines' (#1152):
	// lines of 40 bytes or of 120 weigh alike in lines, not in a task.
	// 0: the lines bound it alone.
	Bytes int
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
	// Decls are the lines shown when the file is given as its
	// declarations: its bodies left out, the pieces given shown whole.
	Decls    []string
	DeclsWhy string
}

// Whole says whether the file is given whole.
func (f File) Whole() bool {
	return len(f.Given) == 1 && f.Given[0].From == 1 && f.Given[0].To == len(f.Lines)
}

// GivenLines are the lines given of the file.
func (f File) GivenLines() int {
	if len(f.Decls) > 0 {
		return len(f.Decls)
	}
	return f.givenPieces()
}

// givenPieces are the lines of the pieces given.
func (f File) givenPieces() int {
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
	Short   bool     // something named was cut or left out, past the budget

	byLines, byBytes bool // which bound cut it: Ask.Budget's lines, Ask.Bytes
	Ask              Ask
}

// Paths are the files given, here or above, in order, the test aside.
func (p Pick) Paths() []string {
	var out []string
	for _, f := range p.Files {
		if !f.Test && (len(f.Given) > 0 || len(f.Decls) > 0 || f.Above) {
			out = append(out, f.Path)
		}
	}
	return out
}

// WholePaths are the files given whole, here, the test aside: those a
// role read all of (#1440).
func (p Pick) WholePaths() []string {
	var out []string
	for _, f := range p.Files {
		if !f.Test && len(f.Decls) == 0 && f.Whole() {
			out = append(out, f.Path)
		}
	}
	return out
}

// Bytes are the bytes of the code and the test given.
func (p Pick) Bytes() int {
	n := 0
	for _, f := range p.Files {
		if b := f.bytes(); b > 0 {
			n += b + fileBytes
		}
	}
	return n
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
	notNames  map[string]bool     // the words never searched, by the commit's languages
	byDir     map[string][]string // the commit's files by folder
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
	if err := codenav.UseProject(repo); err != nil { // its packs' code navigation rows
		return nil, fmt.Errorf("the project's packs: %v", err)
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
		// A binary file (a NUL in its first 8 KB, as git tells one) gives
		// no lines: no image's bytes in a task.
		if out, err := exec.Command("git", "-C", p.repo, "show", p.rev+":"+file).Output(); err == nil && !bytes.Contains(out[:min(len(out), 8000)], []byte{0}) {
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

// commonWords are words a text writes as code that name no definition
// worth searching in most languages; each language's own are in its row
// of codenav.Languages, a language pack's (#1109).
var commonWords = strings.Fields(`if else while for foreach switch case return break continue
	null true false new this super class enum public private protected static const var
	int long float double char byte string String import package`)

// notName says whether a word written as code names nothing worth a
// search: a common word, or a keyword or type of a language the commit
// holds (`open` is searched in Python, not in Kotlin). A commit of no
// known language: every language's.
func (p *Picker) notName(w string) bool {
	if p.notNames == nil {
		p.notNames = map[string]bool{}
		for _, c := range commonWords {
			p.notNames[c] = true
		}
		langs := map[*codenav.Language]bool{}
		for _, f := range p.files {
			if l := codenav.Of(f); l != nil {
				langs[l] = true
			}
		}
		ls := codenav.Languages()
		for i := range ls {
			if l := &ls[i]; langs[l] || len(langs) == 0 {
				for _, k := range strings.Fields(l.Keywords) {
					p.notNames[k] = true
				}
			}
		}
	}
	return p.notNames[w]
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
		if f != "" && !slices.Contains(a.Skip, f) && !slices.Contains(named, f) && len(named) < namedMax && p.tracked[f] && len(p.Lines(f)) > 0 {
			named = append(named, f)
		}
	}
	// Each word's files in turn: two folders named share the bound, the
	// first not taking it whole.
	var each [][]string
	for _, w := range a.Paths {
		each = append(each, p.wordFiles(strings.TrimSpace(work.LineOff(w)), a.Tests.Patterns()))
	}
	for _, f := range inTurn(each) {
		add(f)
	}
	for _, m := range pathLike.FindAllString(a.Text, -1) {
		w := strings.TrimRight(strings.TrimPrefix(strings.Trim(m, "`'\""), "./"), ".,;:/")
		if f := work.InRepo(w, p.files); f != "" && (strings.Contains(w, "/") || path.Ext(w) != "") {
			add(f)
		}
	}
	return named
}

// inTurn are lists' items taken in turn, each list's first, then each
// one's second…
func inTurn(each [][]string) []string {
	var out []string
	for i := 0; slices.ContainsFunc(each, func(fs []string) bool { return i < len(fs) }); i++ {
		for _, fs := range each {
			if i < len(fs) {
				out = append(out, fs[i])
			}
		}
	}
	return out
}

// codeNames are the names a text writes as code worth a search: an
// acronym (`CSV`, `MQTT`) names a format or a protocol as often as code,
// a constant holds an underscore.
func (p *Picker) codeNames(text string) []string {
	code, _, _ := NamedIn(text)
	var names []string
	for _, n := range code {
		if !p.notName(n) && len(n) > 2 && identRe.MatchString(n) && (strings.ToUpper(n) != n || strings.Contains(n, "_")) {
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
	names := p.codeNames(a.Text)
	hits := p.grep(names)
	for _, n := range names {
		pats := map[*codenav.Language]*regexp.Regexp{}
		for _, h := range hits[n] {
			if slices.Contains(out, h.file) || slices.Contains(a.Skip, h.file) || a.Tests.Aside(h.file) {
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
	named := p.namedFiles(a)
	for _, f := range named {
		file(f).Named = true
	}
	// The files the issue is about by its words: the component its title
	// names (`WebUI: …`), the code holding a route it writes (#1152).
	routed, routeFiles := p.routeLines(a, routes(a.Text))
	about := map[string]bool{}
	for _, f := range append(p.titleStems(a), routeFiles...) {
		about[f] = true
	}

	type want struct {
		file     string
		from, to int
		why      string
		cut      bool // may be given in part
	}
	calls := map[string][]string{} // a definition's, by its file:line: the names it calls
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
	names := p.codeNames(a.Text)
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
					calls[fmt.Sprintf("%s:%d", h.file, g.From)] = g.Calls
					if a.Tests.Aside(h.file) {
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
		slices.SortStableFunc(fd.uses, func(x, y want) int { return useRank(a.Tests, x.file) - useRank(a.Tests, y.file) })
	}
	// The rarer a name, the more it says: its pieces come first.
	rank := slices.Clone(names)
	slices.SortStableFunc(rank, func(x, y string) int { return len(hits[x]) - len(hits[y]) })
	inNamed := func(w want) bool { return slices.Contains(named, w.file) }
	// The files the issue is about: those named, and those holding the
	// one definition of a name in the project's own code (`enableLED` in
	// `LED.h`). A name defined in more places than defsMax (`loop`) is
	// read there; one defined in a few is given in each (#1129).
	home := map[string]bool{}
	for _, n := range names {
		var in []string
		for _, w := range byName[n].defs {
			if !slices.Contains(in, w.file) {
				in = append(in, w.file)
			}
		}
		if len(in) == 1 {
			home[in[0]] = true
		}
	}
	atHome := func(n string, w want) bool {
		return inNamed(w) || home[w.file] && len(byName[n].defs) > defsMax
	}
	// 2a. Definitions in the files the issue is about, then the
	// project's own. A name defined in one of them is that one: one of
	// the same name elsewhere is another function, counted, not given
	// (#425, #1129).
	here := map[string]bool{}
	for _, n := range rank {
		here[n] = slices.ContainsFunc(append(slices.Clone(byName[n].defs), byName[n].aside...), func(w want) bool { return atHome(n, w) })
	}
	// The component the issue is about by its words orders, never hides
	// (#1152): its definitions first (`shutdown()` in the RemoteConsole
	// its title names), then the other files holding a name it writes,
	// then its folder's; the others last, while the budget lasts. A title
	// can name what a person sees, not where the fault is.
	var tops []string
	for f := range about {
		tops = append(tops, top(f))
	}
	for _, f := range named {
		tops = append(tops, top(f))
	}
	// nearness: 0 the files the issue is about, 1 those holding the one
	// definition of a name it writes, 2 its component's folder, 3 the
	// rest.
	nearness := func(w want) int {
		switch {
		case inNamed(w) || about[w.file]:
			return 0
		case home[w.file]:
			return 1
		case len(about) > 0 && top(w.file) != "" && slices.Contains(tops, top(w.file)):
			return 2
		}
		return 3
	}
	byNear := func(ws []want) []want {
		ws = slices.Clone(ws)
		slices.SortStableFunc(ws, func(x, y want) int { return nearness(x) - nearness(y) })
		return ws
	}
	defsGiven := map[string]int{}
	for _, n := range rank {
		for _, w := range byName[n].defs {
			if atHome(n, w) {
				defsGiven[n]++
				wants = append(wants, w)
			}
		}
	}
	for _, r := range routed { // the code a route names, right after
		wants = append(wants, want{r.file, r.from, r.to, r.why, true})
	}
	// A name defined in one file only (`DoWork` in `worker.go`), and the
	// definitions in the component the issue is about: before the
	// functions they call, before the files asked for and the places
	// naming a name.
	once := func(n string) bool {
		ds := byName[n].defs
		return len(ds) > 0 && !slices.ContainsFunc(ds, func(w want) bool { return w.file != ds[0].file })
	}
	early := func(n string, w want) bool { return !atHome(n, w) && !here[n] && (about[w.file] || once(n)) }
	for _, n := range rank {
		for _, w := range byName[n].defs {
			if early(n, w) {
				defsGiven[n]++
				wants = append(wants, w)
			}
		}
	}
	// The other definitions, after the functions those call (2b'): the
	// nearest first, while the budget lasts.
	var others []want
	for _, n := range rank {
		for _, w := range byNear(byName[n].defs) {
			if !atHome(n, w) && !early(n, w) && !here[n] && defsGiven[n] < defsMax {
				defsGiven[n]++
				others = append(others, w)
			}
		}
	}
	// 1b. The files the agent asked for, whole, or their first lines.
	// With a depth, a test whole while it fits, the others as named
	// files are given (3): their declarations, then whole if they fit.
	// A folder asked for is its files, as a folder named is (#1410).
	// Several folders take turns, namedMax of their files at most.
	var each [][]string
	inFolder, askedByName := map[string]bool{}, map[string]bool{}
	for _, w := range a.Needs {
		w = strings.TrimSpace(work.LineOff(w))
		if f := work.InRepo(w, p.files); f != "" {
			each, askedByName[f] = append(each, []string{f}), true
			continue
		}
		fs := p.wordFiles(w, a.Tests.Patterns())
		for _, f := range fs {
			inFolder[f] = true
		}
		each = append(each, fs)
	}
	for f := range askedByName {
		delete(inFolder, f) // asked for by its name: never bounded
	}
	var asked, askedTests []string
	seen, fromFolders := map[string]bool{}, 0
	for _, f := range inTurn(each) {
		if seen[f] || skip(f) || len(p.Lines(f)) == 0 || inFolder[f] && fromFolders >= namedMax {
			continue
		}
		seen[f] = true
		if inFolder[f] {
			fromFolders++
		}
		file(f).Named = true
		switch {
		case a.Depth > 0 && a.Tests.Test(f):
			askedTests = append(askedTests, f)
		case a.Depth > 0:
			asked = append(asked, f)
		default:
			wants = append(wants, want{f, 1, len(p.Lines(f)), askedWhy, true})
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
			if (inNamed(w) || about[w.file]) && usedIn[n] < usesMax {
				usedIn[n]++
				wants = append(wants, w)
			}
		}
	}
	// 2b'. The functions a definition in those files calls, defined
	// once in its file, the longest first, a few: `loop()` says little,
	// `updateEffects()` it calls says what it does, a one-line helper
	// less (#1129).
	for _, n := range rank {
		for _, w := range byName[n].defs {
			// A callee after its caller: only a definition given before.
			if !inNamed(w) && !home[w.file] && !about[w.file] || !atHome(n, w) && !early(n, w) {
				continue
			}
			lines := p.Lines(w.file)
			var callees []codenav.Func
			for _, c := range calls[fmt.Sprintf("%s:%d", w.file, w.from)] {
				// Two of the same name in the file (`String()` of two
				// types): which one is called is not known, none given.
				if g := codenav.Named(w.file, lines, c); !slices.Contains(names, c) && len(g) == 1 && g[0].To-g[0].From < funcMax {
					callees = append(callees, g[0])
				}
			}
			slices.SortStableFunc(callees, func(x, y codenav.Func) int { return (y.To - y.From) - (x.To - x.From) })
			for _, g := range callees[:min(len(callees), callsMax)] {
				wants = append(wants, want{w.file, g.From, g.To, fmt.Sprintf("`%s`, called by `%s`", g.Name, n), false})
			}
		}
	}
	wants = append(wants, others...)
	// 2c. Copies in tests, examples and tools, a few.
	for _, n := range rank {
		fd := byName[n]
		given, asides := defsGiven[n], 0
		for _, w := range byNear(fd.aside) {
			if atHome(n, w) || !here[n] && asides < asideMax {
				asides++
				given++
				wants = append(wants, w)
			}
		}
		switch total := len(fd.defs) + len(fd.aside); {
		case given == total:
		case here[n] && slices.ContainsFunc(append(slices.Clone(fd.defs), fd.aside...), inNamed):
			pk.Many = append(pk.Many, fmt.Sprintf("`%s` (defined in the files named; %d more of the same name elsewhere, not given)", n, total-given))
		case here[n]:
			pk.Many = append(pk.Many, fmt.Sprintf("`%s` (defined beside the other names the text writes; %d more of the same name elsewhere, not given)", n, total-given))
		default:
			pk.Many = append(pk.Many, fmt.Sprintf("`%s` (%d definitions, %d of them not given, most in tests, examples or tools)", n, total, total-given))
		}
	}
	// The test kept room for, before the code fills the budget.
	var test string
	kept := 0
	if a.TestRoom {
		// Near the files named; none named, near the definitions found.
		anchor := slices.Clone(named)
		for _, w := range wants {
			if len(named) == 0 && !slices.Contains(anchor, w.file) {
				anchor = append(anchor, w.file)
			}
		}
		ts := NearestTests(p.files, a.Tests.Patterns(), anchor)
		if len(ts) == 0 && len(anchor) > 0 { // none near: the project's first, its style
			ts = slices.DeleteFunc(slices.Clone(p.files), func(f string) bool { return !a.Tests.Test(f) })
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
	// byLines, byBytes: which bound cut it, the lines or the bytes, said
	// by its own setting (#1159).
	byLines, byBytes := false, false
	cutBy := func(bytes bool) {
		short = true
		if bytes {
			byBytes = true
		} else {
			byLines = true
		}
	}
	// bytesLeft are the bytes the code may still take; a lot when none
	// bound it.
	bytesLeft := func() int {
		if a.Bytes <= 0 {
			return 1 << 40
		}
		used := 0
		for _, g := range byPath {
			if b := g.bytes(); b > 0 {
				used += b + fileBytes
			}
		}
		return a.Bytes - used
	}
	// room is the bytes a file may still take: a file not given yet
	// weighs fileBytes more.
	room := func(f *File) int {
		if f.bytes() == 0 {
			return bytesLeft() - fileBytes
		}
		return bytesLeft()
	}
	// give plans a piece; false when nothing of it fits.
	give := func(w want) bool {
		f := file(w.file)
		if covered(p.given[w.file], w.from, w.to) || covered(f.Given, w.from, w.to) {
			return true
		}
		n := w.to - w.from + 1
		if inBytes := fitting(f.Lines, w.from, room(f)); n > min(left, inBytes) {
			fit := min(left, inBytes)
			cutBy(inBytes < left) // fewer lines fit the bytes than the lines left
			if !w.cut || fit <= 0 {
				if w.cut {
					f.Unread = append(f.Unread, Piece{w.from, w.to, w.why})
				}
				return false
			}
			f.Cut = w.to
			w.to = w.from + fit - 1
			n = fit
		}
		f.Given = merge(f.Given, Piece{w.from, w.to, w.why})
		left -= n
		return true
	}
	for _, w := range wants {
		give(w)
	}
	// 3. The files named, whole when what they add fits; else, with
	// nothing of them given, their first lines. With a depth, their
	// declarations first, the pieces given shown whole in them; then the
	// files the test and the code include or import (3b); then each
	// named file whole if what it adds still fits. The files the agent
	// asked for come first.
	own := slices.Clone(asked)
	for _, f := range named {
		if !slices.Contains(own, f) {
			own = append(own, f)
		}
	}
	why := func(f string) string {
		if slices.Contains(asked, f) {
			return askedWhy
		}
		return "named"
	}
	whole := func(g *File) {
		left -= len(g.Lines) - g.GivenLines()
		g.Given, g.Cut, g.Unread, g.Decls = []Piece{{1, len(g.Lines), why(g.Path)}}, 0, nil, nil
	}
	for _, f := range own {
		g := file(f)
		extra := len(g.Lines) - g.GivenLines()
		fits := extra <= left && g.wholeBytes()-g.bytes() <= room(g)
		switch {
		case len(p.given[f]) > 0 && len(g.Given) == 0:
			g.Above = true // given for an earlier issue: not twice
		case g.Whole():
		case slices.Contains(asked, f) && fits: // asked for by name: whole while it fits
			whole(g)
		case a.Depth > 0 && p.declare(g, why(f), left+g.GivenLines(), bytesLeft()):
			left -= g.GivenLines() - g.givenPieces()
		case fits:
			whole(g)
		default:
			cutBy(extra <= left) // the lines fit: the bytes cut it
		}
	}
	for _, f := range own {
		g := file(f)
		if len(g.Given) == 0 && len(g.Decls) == 0 && len(g.Unread) == 0 && !g.Above && left > 0 && len(g.Lines) > 0 {
			k := min(len(g.Lines), left, fitting(g.Lines, 1, room(g)))
			if k <= 0 {
				cutBy(true) // lines left: the bytes cut it
				continue
			}
			g.Given = []Piece{{1, k, "its first lines"}}
			left -= k
		}
	}
	// 3b. With a depth: the files asked for whole while they fit (asked
	// for by name, given so); the files the test and the code include or
	// import, as their declarations, the test's first: what a new test
	// needs to build; the tests asked for, their first lines at least;
	// the files named whole while they fit.
	if a.Depth > 0 {
		upgrade := func(fs []string) {
			for _, f := range fs {
				if g := file(f); len(g.Decls) > 0 {
					if len(g.Lines)-g.GivenLines() <= left {
						whole(g)
					} else {
						cutBy(false)
					}
				}
			}
		}
		upgrade(asked)
		roots := slices.Clone(asked)
		for _, w := range a.Paths {
			roots = append(roots, p.wordFiles(strings.TrimSpace(work.LineOff(w)), a.Tests.Patterns())...)
		}
		left = p.closure(a, test, roots, order, byPath, file, left, bytesLeft)
		for _, f := range askedTests {
			give(want{f, 1, len(p.Lines(f)), askedWhy, true})
		}
		upgrade(own)
	}
	// 4. The other places naming a name, the rarer first, a few each,
	// those in the files named first, a third of the budget at most:
	// what defines the code comes before what uses it.
	// The code before the docs, every name's (#1152); the component the
	// issue names first.
	usesLeft := budget / usesShare
	dropped := map[string]int{}
	for _, docs := range []bool{false, true} {
		for _, n := range rank {
			for _, w := range byNear(byName[n].uses) {
				if (useRank(a.Tests, w.file) == 2) != docs || covered(file(w.file).Given, w.from, w.to) {
					continue
				}
				size := w.to - w.from + 1
				if usedIn[n] >= usesMax || size > usesLeft || !give(w) {
					dropped[n]++
					continue
				}
				usedIn[n]++
				usesLeft -= size
			}
		}
	}
	for _, n := range rank {
		if dropped[n] > 0 {
			pk.Unshown = append(pk.Unshown, fmt.Sprintf("`%s` (%d more places)", n, dropped[n]))
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
		if g.Test || f == test && tf != nil && len(tf.Given) > 0 && !g.Named { // given once, as the test
			continue
		}
		if len(g.Given) == 0 && len(g.Decls) == 0 && !g.Named && len(g.Unread) == 0 {
			continue
		}
		pk.Files = append(pk.Files, *g)
		p.given[f] = append(p.given[f], g.Given...)
	}
	if tf != nil {
		pk.Files = append(pk.Files, *tf)
	}
	pk.Linked = p.linked(pk, a.Tests.Patterns())
	pk.Left, pk.Short, pk.byLines, pk.byBytes = left, short, byLines, byBytes
	return pk
}

// useRank orders the places naming a name: the project's own code, then
// its tests', examples' and tools', then docs and files no language is
// known for.
func useRank(d testdef.Def, f string) int {
	switch {
	case codenav.Of(f) == nil:
		return 2
	case d.Aside(f):
		return 1
	}
	return 0
}

// bytes are the bytes of the lines given of a file, each with its end
// of line.
func (f File) bytes() int {
	n := 0
	if len(f.Decls) > 0 {
		for _, l := range f.Decls {
			n += len(l) + 1
		}
		return n
	}
	for _, q := range f.Given {
		for i := q.From; i <= min(q.To, len(f.Lines)); i++ {
			n += len(f.Lines[i-1]) + 1
		}
	}
	return n
}

// wholeBytes are the bytes of a file whole.
func (f File) wholeBytes() int {
	n := 0
	for _, l := range f.Lines {
		n += len(l) + 1
	}
	return n
}

// fitting is how many of a file's lines from a line on fit in so many
// bytes.
func fitting(lines []string, from, bytes int) int {
	k := 0
	for i := from; i <= len(lines) && bytes >= len(lines[i-1])+1; i++ {
		bytes -= len(lines[i-1]) + 1
		k++
	}
	if from > len(lines) {
		return 1 << 30 // no lines to weigh: the lines' bound alone
	}
	return k
}

// top is the folder a file's path opens on, the component it is in in
// a project of components; "" for a file at the root.
func top(f string) string {
	t, _, ok := strings.Cut(f, "/")
	if !ok {
		return ""
	}
	return t
}

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
			words[n] = wordRe(n)
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

// wordRe matches a name as a word: a route (`/api/x`) opens on no
// word's edge.
func wordRe(n string) *regexp.Regexp {
	edge := func(c byte) string {
		if c == '_' || c >= '0' && c <= '9' || c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' {
			return `\b`
		}
		return ""
	}
	return regexp.MustCompile(edge(n[0]) + regexp.QuoteMeta(n) + edge(n[len(n)-1]))
}

// wordFiles are the files a word names: a file, the files a pattern
// matches, or a folder's own files, its tests aside when it holds code
// too (the tester's rule, #791, #801), then the code files below it,
// the nearest first, namedMax at most: a module's folder holds its build file, its code
// sits folders below (`core/x/src/main/kotlin/…`, #1410).
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
	var out, below []string
	for _, f := range p.files {
		switch {
		case len(tests) > 0 && pathglob.Any(tests, f):
		case path.Dir(f) == dir:
			out = append(out, f)
		case strings.HasPrefix(f, dir+"/") && codenav.Of(f) != nil:
			below = append(below, f)
		}
	}
	slices.SortStableFunc(below, func(x, y string) int { return strings.Count(x, "/") - strings.Count(y, "/") })
	return append(out, below[:min(len(below), namedMax)]...)
}

func sortedKeys(m map[string][][2]int) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}
