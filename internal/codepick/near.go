package codepick

import (
	"fmt"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/codenav"
	"github.com/JN0V/workline/internal/pathglob"
)

// NearestTests are the test files nearest some code, nearest first (the
// tester's rule, #1047): in a code file's folder, those named after it
// (ordinals.go: ordinals_test.go, test_ordinals.py) first; else those
// sharing the most leading folders with one, one at least, those whose
// name holds a code file's name first. None near: none.
func NearestTests(files, patterns, code []string) []string {
	dirs := map[string]bool{}
	stems := map[string]bool{}
	for _, p := range code {
		dirs[path.Dir(p)] = true
		stems[norm(strings.TrimSuffix(path.Base(p), path.Ext(p)))] = true
	}
	var tests, beside, nearby []string
	for _, f := range files {
		if f == "" || !pathglob.Any(patterns, f) {
			continue
		}
		tests = append(tests, f)
		base := norm(strings.TrimSuffix(path.Base(f), path.Ext(f)))
		switch {
		case !dirs[path.Dir(f)]:
		case stems[strings.TrimSuffix(strings.TrimPrefix(base, "test"), "test")]:
			beside = append(beside, f)
		default:
			nearby = append(nearby, f)
		}
	}
	if out := append(beside, nearby...); len(out) > 0 {
		return out
	}
	return kinTests(tests, code, stems)
}

// kinTests are the tests sharing the most leading folders with a code
// file, one at least, those whose name holds a code file's name first
// (`Mod/include/Mod.h` named, `Mod/test/test_mod_x/test_mod_x.cpp` beside
// it, not another module's tests).
func kinTests(tests, code []string, stems map[string]bool) []string {
	shared := func(a, b string) int {
		as, bs := strings.Split(path.Dir(a), "/"), strings.Split(path.Dir(b), "/")
		n := 0
		for n < len(as) && n < len(bs) && as[n] == bs[n] && as[n] != "." {
			n++
		}
		return n
	}
	score := map[string]int{}
	var kin []string
	for _, t := range tests {
		best := 0
		for _, p := range code {
			best = max(best, shared(t, p))
		}
		if best == 0 {
			continue
		}
		score[t] = 2 * best
		for stem := range stems {
			if stem != "" && strings.Contains(norm(path.Base(t)), stem) {
				score[t]++
				break
			}
		}
		kin = append(kin, t)
	}
	slices.SortStableFunc(kin, func(a, b string) int { return score[b] - score[a] })
	return kin
}

// norm keeps a text's letters and digits, lowercased.
func norm(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r > 127 {
			b.WriteRune(r)
		}
	}
	return b.String()
}

var (
	// includeRe is a file a line includes or imports by a path: C and
	// C++ `#include "a.h"` or `<a/b.h>`, JavaScript `from './a'` and
	// `require('./a')`, Python `from a.b import c` and `import a.b`.
	includeRe = regexp.MustCompile(`^\s*#\s*include\s*[<"]([^>"]+)[>"]|\bfrom\s+['"](\.{1,2}/[^'"]+)['"]|\brequire\(\s*['"](\.{1,2}/[^'"]+)['"]|^\s*from\s+([\w.]+)\s+import\b|^\s*import\s+([\w.]+)\s*$`)
	// baseRe is a type a class derives from: C++ `: public Base`, Python
	// `class A(Base):`, Java and TypeScript `extends Base`.
	baseRe = regexp.MustCompile(`\b(?:public|protected|private|extends|implements)\s+(?:virtual\s+)?([A-Z]\w*)|^\s*class\s+\w+\(([A-Z]\w*)`)
)

// linked are the files one step away from the code given, as paths: those
// it includes or imports, then those defining the types it derives from.
func (p *Picker) linked(pk Pick, tests []string) []string {
	given := map[string]bool{}
	for _, f := range pk.Files {
		given[f.Path] = true
	}
	var out []string
	seen := map[string]bool{}
	add := func(f, why string) {
		if f != "" && !given[f] && !seen[f] && len(out) < linkedMax {
			seen[f] = true
			out = append(out, fmt.Sprintf("`%s` (%s)", f, why))
		}
	}
	var bases []string
	for _, f := range pk.Files {
		if f.Test || len(f.Given) == 0 {
			continue
		}
		for _, l := range f.Lines {
			m := includeRe.FindStringSubmatch(l)
			if m == nil {
				continue
			}
			for k, w := range m[1:] {
				switch {
				case w == "":
				case k >= 3: // a Python module: a.b is a/b.py, or a/b/__init__.py
					w = strings.ReplaceAll(w, ".", "/")
					got := p.resolve(f.Path, w+".py")
					if got == "" {
						got = p.resolve(f.Path, w+"/__init__.py")
					}
					add(got, "imported")
				default:
					add(p.resolve(f.Path, w), "included or imported")
				}
			}
		}
		for _, g := range f.Given {
			for _, l := range f.Lines[g.From-1 : min(g.To, len(f.Lines))] {
				for _, m := range baseRe.FindAllStringSubmatch(l, -1) {
					for _, w := range m[1:] {
						if w != "" && !slices.Contains(bases, w) {
							bases = append(bases, w)
						}
					}
				}
			}
		}
	}
	if len(bases) > 0 {
		hits := p.grep(bases)
		for _, b := range bases {
			def := regexp.MustCompile(`\b(?:class|struct|interface|trait|type)\s+` + regexp.QuoteMeta(b) + `\b[^;]*$`)
			for _, h := range hits[b] {
				if codenav.Of(h.file) != nil && def.MatchString(h.line) && (len(tests) == 0 || !pathglob.Any(tests, h.file)) {
					add(h.file, fmt.Sprintf("defines `%s`, a type the code derives from", b))
					break
				}
			}
		}
	}
	return out
}

// resolve is the file an include or an import names from a file: beside
// it, from the root, or the one file whose path ends with it.
func (p *Picker) resolve(from, w string) string {
	var tries []string
	if strings.HasPrefix(w, ".") {
		base := path.Join(path.Dir(from), w)
		tries = append(tries, base)
		for _, e := range []string{".js", ".ts", ".jsx", ".tsx", ".mjs", "/index.js", "/index.ts"} {
			tries = append(tries, base+e)
		}
	} else {
		tries = append(tries, path.Join(path.Dir(from), w), w)
	}
	for _, t := range tries {
		if p.tracked[t] {
			return t
		}
	}
	if strings.HasPrefix(w, ".") {
		return ""
	}
	found := ""
	for _, f := range p.files {
		if strings.HasSuffix(f, "/"+w) {
			if found != "" {
				return "" // several: none named
			}
			found = f
		}
	}
	return found
}
