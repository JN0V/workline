package packs

// Rule B's tripwire (ADR-0065, "Rule B: no ecosystem pattern in the
// engine"): a Go string literal of internal/ holding a pack's marker file,
// extension or path pattern fails the tests, unless a `// converges:`
// comment on its line says why, or the baseline lists it. A tripwire, not
// a proof: the pull request's review catches what it misses.

import (
	"bufio"
	"fmt"
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
	"unicode"
)

// ruleBBaseline lists today's offenders, a file, a token and how many
// literals of the file hold it. It may only shrink.
const ruleBBaseline = "testdata/rule-b-baseline.txt"

// patternToken is one thing a literal must not hold: open is true when the
// pattern goes on with a wildcard after it, so any character may follow.
type patternToken struct {
	text string
	open bool
}

// ruleBTokens are the markers, extensions and path patterns of every pack
// but the generic one, less what the generic pack's test paths and homes
// give: `test` or `_test.` would flag many plain Go strings.
func ruleBTokens(all map[string]Pack) []patternToken {
	generic := map[string]bool{}
	if g, ok := all[Generic]; ok {
		for _, t := range packTokens(Pack{TestPaths: g.TestPaths, TestHomes: g.TestHomes}) {
			generic[t.text] = true
		}
	}
	seen := map[string]bool{}
	var out []patternToken
	for _, name := range Names(all) {
		if name == Generic {
			continue
		}
		for _, t := range packTokens(all[name]) {
			if generic[t.text] || seen[t.text] {
				continue
			}
			seen[t.text] = true
			out = append(out, t)
		}
	}
	slices.SortFunc(out, func(a, b patternToken) int { return strings.Compare(a.text, b.text) })
	return out
}

// packTokens is one pack's tokens: its marker files and extensions as
// written, and the literal parts of its test paths and test homes.
func packTokens(p Pack) []patternToken {
	var out []patternToken
	for _, f := range p.Detect.Files {
		out = append(out, patternToken{text: f})
	}
	for _, e := range p.Detect.Extensions {
		out = append(out, patternToken{text: e})
	}
	globs := slices.Clone(p.TestPaths)
	for _, h := range p.TestHomes {
		globs = append(globs, h.Tests, h.Code)
	}
	globs = append(globs, p.Aside...)
	for _, g := range globs {
		out = append(out, globTokens(g)...)
	}
	return out
}

var globClass = regexp.MustCompile(`\[[^\]]*\]`)

// globTokens is a glob's literal parts worth a token: a run of two folders
// or more (`src/test`, `src/androidTest` before a wildcard), and a part
// holding a dot (`_test.go`, `conftest.py`). A single folder name (`test`,
// `src`) is too plain a word to flag.
func globTokens(glob string) []patternToken {
	var out []patternToken
	var run []string
	flush := func(open bool) {
		if len(run) >= 2 {
			out = append(out, patternToken{text: strings.Join(run, "/"), open: open})
		}
		run = nil
	}
	for _, seg := range strings.Split(glob, "/") {
		seg = globClass.ReplaceAllString(seg, "*")
		if !strings.ContainsAny(seg, "*?") {
			if seg == "" {
				flush(false)
				continue
			}
			run = append(run, seg)
			if strings.Contains(seg, ".") && len(run) == 1 {
				out = append(out, patternToken{text: seg})
			}
			continue
		}
		parts := strings.FieldsFunc(seg, func(r rune) bool { return r == '*' || r == '?' })
		if prefix, _, _ := strings.Cut(seg, "*"); prefix != "" && !strings.Contains(prefix, "?") {
			run = append(run, prefix)
			flush(true)
		} else {
			flush(false)
		}
		for i, part := range parts {
			if strings.Contains(part, ".") {
				out = append(out, patternToken{text: part, open: i < len(parts)-1 || strings.HasSuffix(seg, "*")})
			}
		}
	}
	flush(false)
	return out
}

// holds says whether s holds the token as a token: not glued to a letter
// or a digit on a side where the token itself starts or ends with one, so
// `.kt` is in `task.kt.bak` and `Main.kt`, never in `a.ktx`.
func holds(s string, t patternToken) bool {
	for i := 0; ; {
		j := strings.Index(s[i:], t.text)
		if j < 0 {
			return false
		}
		start, end := i+j, i+j+len(t.text)
		if glued(s, start-1, t.text[0]) || (!t.open && glued(s, end, t.text[len(t.text)-1])) {
			i = start + 1
			continue
		}
		return true
	}
}

// glued says whether s[at] is a letter or a digit and the token's own
// edge, edge, is one too.
func glued(s string, at int, edge byte) bool {
	if at < 0 || at >= len(s) {
		return false
	}
	return wordByte(edge) && wordByte(s[at])
}

func wordByte(b byte) bool {
	return b >= 0x80 || unicode.IsLetter(rune(b)) || unicode.IsDigit(rune(b))
}

// unescapeRegex drops a regex's escapes, so `\.kt$` reads `.kt$`.
func unescapeRegex(s string) string {
	return strings.ReplaceAll(s, `\`, "")
}

// ruleBHit is a literal holding a token; lit is its line holding it.
type ruleBHit struct {
	file  string
	line  int
	token string
	lit   string
}

var convergesComment = regexp.MustCompile(`^//\s*converges:\s*\S`)

// ruleBScan is the hits of one Go file's string literals, import paths and
// struct tags left out, and a literal with a `// converges: <reason>`
// comment on its first or last line exempted.
func ruleBScan(name string, src []byte, tokens []patternToken) ([]ruleBHit, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, name, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	exempt := map[int]bool{}
	for _, cg := range f.Comments {
		for _, c := range cg.List {
			if convergesComment.MatchString(c.Text) {
				exempt[fset.Position(c.Slash).Line] = true
			}
		}
	}
	skip := map[*ast.BasicLit]bool{}
	for _, imp := range f.Imports {
		skip[imp.Path] = true
	}
	var hits []ruleBHit
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.Field:
			if n.Tag != nil {
				skip[n.Tag] = true
			}
		case *ast.BasicLit:
			if n.Kind != token.STRING || skip[n] {
				return true
			}
			first, last := fset.Position(n.Pos()).Line, fset.Position(n.End()).Line
			if exempt[first] || exempt[last] {
				return true
			}
			s, err := strconv.Unquote(n.Value)
			if err != nil {
				s = n.Value
			}
			var lit []ruleBHit
			ls := strings.Split(s, "\n")
			for _, t := range tokens {
				at := tokenLine(s, t)
				if at < 0 {
					at = tokenLine(unescapeRegex(s), t)
				}
				if at < 0 {
					continue
				}
				line := first
				if strings.HasPrefix(n.Value, "`") {
					line += at
				}
				lit = append(lit, ruleBHit{file: name, line: line, token: t.text, lit: strings.TrimSpace(ls[at])})
			}
			// One token inside another the literal holds counts once, as
			// the longer: `build.gradle.kts`, not `.kts` too.
			for _, h := range lit {
				if !slices.ContainsFunc(lit, func(o ruleBHit) bool {
					return o.token != h.token && strings.Contains(o.token, h.token)
				}) {
					hits = append(hits, h)
				}
			}
		}
		return true
	})
	return hits, nil
}

// tokenLine is the line of s, from 0, holding the token, or -1.
func tokenLine(s string, t patternToken) int {
	for i, l := range strings.Split(s, "\n") {
		if holds(l, t) {
			return i
		}
	}
	return -1
}

// ruleBKey is a baseline line's key: a file and a token.
type ruleBKey struct{ file, token string }

func readRuleBBaseline(t *testing.T) map[ruleBKey]int {
	t.Helper()
	f, err := os.Open(ruleBBaseline)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	out := map[ruleBKey]int{}
	sc := bufio.NewScanner(f)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 3 {
			t.Fatalf("%s:%d: want `<file> <token> <literals>`, got %q", ruleBBaseline, n, line)
		}
		c, err := strconv.Atoi(fields[2])
		if err != nil || c < 1 {
			t.Fatalf("%s:%d: %q is not a count", ruleBBaseline, n, fields[2])
		}
		out[ruleBKey{fields[0], fields[1]}] = c
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// No Go file of internal/ holds a pack's marker, extension or path
// pattern beyond the baseline: a new language's fix lands in a pack.
// Test files, testdata and the pack loader are left out.
func TestRuleBNoEcosystemPatternInTheEngine(t *testing.T) {
	tokens := ruleBTokens(Builtin())
	root := filepath.Join("..")
	var hits []ruleBHit
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "testdata" || p == filepath.Join(root, "packs") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		src, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		h, err := ruleBScan(filepath.ToSlash(filepath.Join("internal", rel)), src, tokens)
		hits = append(hits, h...)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	got := map[ruleBKey][]ruleBHit{}
	for _, h := range hits {
		k := ruleBKey{h.file, h.token}
		got[k] = append(got[k], h)
	}
	base := readRuleBBaseline(t)
	var fails []string
	for k, hs := range got {
		if len(hs) <= base[k] {
			continue
		}
		for _, h := range hs {
			fails = append(fails, fmt.Sprintf("%s:%d: %s holds %q: a language pack's pattern: put it in a pack (internal/packs/builtin/), or say why it converges with a `// converges: <reason>` comment on its line (ADR-0065, Rule B); baseline allows %d in this file", h.file, h.line, truncate(h.lit), h.token, base[k]))
		}
	}
	for k, n := range base {
		if c := len(got[k]); c < n {
			fails = append(fails, fmt.Sprintf("%s: %s %s %d is stale, %d literal(s) left: lower or remove its line, the baseline only shrinks", ruleBBaseline, k.file, k.token, n, c))
		}
	}
	slices.Sort(fails)
	for _, f := range fails {
		t.Error(f)
	}
	if t.Failed() && os.Getenv("WORKLINE_RULE_B_PRINT") != "" {
		var lines []string
		for k, hs := range got {
			lines = append(lines, fmt.Sprintf("%s %s %d", k.file, k.token, len(hs)))
		}
		slices.Sort(lines)
		t.Log("today's offenders:\n" + strings.Join(lines, "\n"))
	}
}

// truncate is a literal quoted on one line, its first 60 bytes.
func truncate(lit string) string {
	if len(lit) > 60 {
		lit = lit[:60] + "…"
	}
	return strconv.Quote(lit)
}

// The tokens come from the packs, the generic pack's left out.
func TestRuleBTokens(t *testing.T) {
	var texts []string
	for _, tok := range ruleBTokens(Builtin()) {
		texts = append(texts, tok.text)
	}
	for _, want := range []string{"build.gradle.kts", "go.mod", ".kt", ".py", "_test.go", "src/test", "src/main", "src/androidTest", "conftest.py"} {
		if !slices.Contains(texts, want) {
			t.Errorf("no token %q in %q", want, texts)
		}
	}
	for _, not := range []string{"test", "tests", "__tests__", "test_", ".test.", ".spec.", "src"} {
		if slices.Contains(texts, not) {
			t.Errorf("token %q is plain or the generic pack's: %q", not, texts)
		}
	}
}

// A token is held only when not glued to a letter or a digit on its own
// word edge.
func TestRuleBHolds(t *testing.T) {
	kt := patternToken{text: ".kt"}
	src := patternToken{text: "src/test"}
	android := patternToken{text: "src/androidTest", open: true}
	for _, c := range []struct {
		s    string
		t    patternToken
		want bool
	}{
		{"Main.kt", kt, true},
		{"task.kt.bak", kt, true},
		{"a.ktx", kt, false},
		{"app/src/test/x", src, true},
		{"app/src/testing/x", src, false},
		{"mysrc/test", src, false},
		{"src/androidTestDebug", android, true},
	} {
		if got := holds(c.s, c.t); got != c.want {
			t.Errorf("holds(%q, %q) = %v, want %v", c.s, c.t.text, got, c.want)
		}
	}
}

// The scan flags a literal and a regex, names the file and the line, and
// leaves out a reasoned exemption, an import path and a struct tag.
func TestRuleBScan(t *testing.T) {
	tokens := ruleBTokens(Builtin())
	for _, c := range []struct {
		name, src string
		want      []string // token@line
	}{
		{"literal", "package x\n\nvar a = \"build.gradle.kts\"\n", []string{"build.gradle.kts@3"}},
		{"regex", "package x\n\nimport \"regexp\"\n\nvar r = regexp.MustCompile(`\\w+\\.kt$`)\n", []string{".kt@5"}},
		{"raw string line", "package x\n\nvar p = `say\nopen Main.kt`\n", []string{".kt@4"}},
		{"exempt", "package x\n\nvar a = \".kt\" // converges: an example only\n", nil},
		{"comment without reason", "package x\n\nvar a = \".kt\" // converges:\n", []string{".kt@3"}},
		{"plain comment", "package x\n\nvar a = \".kt\" // the extension\n", []string{".kt@3"}},
		{"clean", "package x\n\nvar a = \"a.ktx and testing\"\n", nil},
		{"tag and import", "package x\n\nimport _ \"example.com/x.py\"\n\ntype T struct{ A string `yaml:\"go.mod\"` }\n", nil},
	} {
		hits, err := ruleBScan(c.name+".go", []byte(c.src), tokens)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		var got []string
		for _, h := range hits {
			got = append(got, fmt.Sprintf("%s@%d", h.token, h.line))
		}
		if !slices.Equal(got, c.want) {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}
