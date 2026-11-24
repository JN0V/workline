package main_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// narrowedDocs are the thirteen user docs whose `sources` the documentalist's
// agent could not judge: too wide, or judged without being vouched for.
var narrowedDocs = []string{
	"README.md",
	"docs/main-after-merge.md",
	"docs/triggers.md",
	"roles/developer/README.md",
	"roles/tester/README.md",
	"roles/documentalist/docs/push-local.md",
	"roles/product-owner/docs/outputs.md",
	"roles/product-owner/docs/run.md",
	"roles/product-owner/docs/tracking.md",
	"roles/product-owner/docs/writing.md",
	"docs/local-line-review.md",
	"docs/local-line-steps.md",
	"docs/github/propose-job.md",
}

// anchorRegion reads the symbol a `path#Symbol` source names, from the
// repository's root: the text of its declaration (a Go function, type,
// variable or constant; a YAML key, with what is under it). found is false
// when the file holds no such symbol.
func anchorRegion(path, sym string) (region string, found bool) {
	data, err := os.ReadFile(filepath.Join(userRepo, path))
	if err != nil {
		return "", false
	}
	lines := strings.Split(string(data), "\n")
	segs := strings.Split(sym, ".")
	last := segs[len(segs)-1]
	q := regexp.QuoteMeta(last)
	indent := func(l string) int { return len(l) - len(strings.TrimLeft(l, " ")) }
	if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
		keyAt := func(from int, k string) int {
			re := regexp.MustCompile(`^\s*(- )?["']?` + regexp.QuoteMeta(k) + `["']?:`)
			for i := from; i < len(lines); i++ {
				if re.MatchString(lines[i]) {
					return i
				}
			}
			return -1
		}
		start, from := -1, 0
		for _, k := range segs {
			if start = keyAt(from, k); start < 0 {
				break
			}
			from = start + 1
		}
		if start < 0 {
			start = keyAt(0, last)
		}
		if start < 0 {
			return "", false
		}
		base, end := indent(lines[start]), len(lines)
		for i := start + 1; i < len(lines); i++ {
			s := strings.TrimSpace(lines[i])
			if s == "" || strings.HasPrefix(s, "#") {
				continue
			}
			if n := indent(lines[i]); n < base || n == base && !strings.HasPrefix(s, "- ") {
				end = i
				break
			}
		}
		return strings.Join(lines[start:end], "\n"), true
	}
	if strings.HasSuffix(path, ".go") {
		decl := regexp.MustCompile(`^(func (\([^)]*\) )?` + q + `[\[(]|(type|var|const) ` + q + `\b)`)
		for i, l := range lines {
			if !decl.MatchString(l) {
				continue
			}
			end := i
			if s := strings.TrimSpace(l); strings.HasSuffix(s, "{") || strings.HasSuffix(s, "(") {
				for end = i + 1; end < len(lines) && lines[end] != "}" && lines[end] != ")"; end++ {
				}
			}
			if end >= len(lines) {
				end = len(lines) - 1
			}
			return strings.Join(lines[i:end+1], "\n"), true
		}
	}
	word := regexp.MustCompile(`\b` + q + `\b`)
	for i, l := range lines {
		if word.MatchString(l) {
			return strings.Join(lines[i:min(i+20, len(lines))], "\n"), true
		}
	}
	return "", false
}

// Thirteen docs judgeable
//
// Given each of the 13 docs,
// When `workline docs --ai none --since <root commit>` runs on the
// repository,
// Then none of the 13 is listed as suspect because its sources are too wide,
// because the agent could not vouch for every sentence, because it was judged
// in parts, or because a person must judge it.
func TestThirteenDocsAreJudgeable(t *testing.T) {
	out := userPageDocsRun(t, false)
	if !strings.Contains(out, "workline:") {
		t.Fatalf("workline docs said nothing:\n%s", out)
	}
	stuck := regexp.MustCompile(`too wide|too large|could not vouch|not vouched for|judged in parts|a person`)
	listed := regexp.MustCompile(`(?m)^\s+suspect (\S+?): .*$`)
	for _, m := range listed.FindAllStringSubmatch(out, -1) {
		for _, doc := range narrowedDocs {
			if m[1] == doc && stuck.MatchString(m[0]) {
				t.Errorf("%s is still not judgeable:\n%s", doc, m[0])
			}
		}
	}
}

// Sources cover the prose
//
// Given a doc with narrowed `sources`,
// When its sources are read,
// Then each is a symbol anchor `path#Symbol` that resolves in its file (a
// test fixture under tests/ may be named whole), never a folder nor a whole
// file, and the doc keeps at least one.
func TestNarrowedSourcesCoverTheProse(t *testing.T) {
	for _, doc := range narrowedDocs {
		sources, _, _ := userPageRead(t, doc)
		if len(sources) == 0 {
			t.Errorf("%s: no source left", doc)
		}
		for _, src := range sources {
			path, sym, anchored := strings.Cut(src, "#")
			if !anchored || sym == "" {
				if !strings.HasPrefix(path, "tests/") {
					t.Errorf("%s: source %q is a file or a folder, not a `path#Symbol` anchor", doc, src)
				}
				continue
			}
			if _, found := anchorRegion(path, sym); !found {
				t.Errorf("%s: source %q names a symbol its file does not hold", doc, src)
			}
		}
	}
}

// Never dropped
//
// Given a doc whose prose states a fact no symbol covers (a setting or rule
// a role's or the routing's YAML declares, named in backticks),
// When its sources are narrowed,
// Then the symbol anchor of some source holds that name: the prose was fixed
// or the source added, the source never dropped.
func TestNarrowingNeverDropsTheSourceOfAFact(t *testing.T) {
	var declared strings.Builder
	files, _ := filepath.Glob(filepath.Join(userRepo, "roles", "*", "role.yaml"))
	files = append(files, filepath.Join(userRepo, "defaults", "routing.yaml"))
	for _, f := range files {
		if data, err := os.ReadFile(f); err == nil {
			declared.Write(data)
			declared.WriteByte('\n')
		}
	}
	for _, doc := range narrowedDocs {
		sources, _, prose := userPageRead(t, doc)
		var covered strings.Builder
		for _, src := range sources {
			path, sym, anchored := strings.Cut(src, "#")
			if !anchored {
				continue // a file or a folder is no symbol: it covers no fact
			}
			if r, found := anchorRegion(path, sym); found {
				covered.WriteString(r + "\n")
			}
		}
		seen := map[string]bool{}
		for _, m := range userSpan.FindAllStringSubmatch(prose, -1) {
			name := strings.TrimSpace(m[1])
			if seen[name] || !userBare.MatchString(name) {
				continue
			}
			seen[name] = true
			key := regexp.MustCompile(`(?m)^\s*["']?` + regexp.QuoteMeta(name) + `["']?:`)
			if key.MatchString(declared.String()) && !strings.Contains(covered.String(), name) {
				t.Errorf("%s: it states `%s`, which no source's symbol holds: fix the prose or add the source", doc, name)
			}
		}
	}
}
