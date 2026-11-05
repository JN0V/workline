package codepick

import (
	"fmt"
	"path"
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
		if len(f.Given) == 0 && len(f.Decls) == 0 { // the test's includes too: what a new test needs to build
			continue
		}
		for _, im := range p.imports(f.Path, tests) {
			add(im.file, im.why)
		}
		lang := codenav.Of(f.Path)
		for _, g := range f.Given {
			if f.Test || lang == nil {
				break
			}
			// The types it derives from or embeds, by its language's
			// pattern (codenav.Languages).
			for _, l := range f.Lines[g.From-1 : min(g.To, len(f.Lines))] {
				for _, w := range lang.BasesIn(l) {
					if !slices.Contains(bases, w) {
						bases = append(bases, w)
					}
				}
			}
		}
	}
	if len(bases) > 0 {
		hits := p.grep(bases)
		for _, b := range bases {
			for _, h := range hits[b] {
				if l := codenav.Of(h.file); l != nil && l.Defines(b).MatchString(h.line) && (len(tests) == 0 || !pathglob.Any(tests, h.file)) {
					add(h.file, fmt.Sprintf("defines `%s`, a type the code derives from", b))
					break
				}
			}
		}
	}
	return out
}
