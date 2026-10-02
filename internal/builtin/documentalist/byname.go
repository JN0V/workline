package documentalist

import (
	"path"
	"sort"
	"strings"
)

// Sources by name: adopting a repository with no agent, a doc's sources
// are proposed when its name, or the name of a folder holding it up to
// `docs/`, is a code folder's or file's. A proposal for a person, never
// written: on six public repositories it matched 335 of backstage's 524
// docs (named after its plugins), and 0 to 6 elsewhere
// (docs/research/documentalist-genericity.md).

// vagueStems say nothing of what a doc describes: its folder does.
var vagueStems = map[string]bool{"index": true, "readme": true, "overview": true, "intro": true,
	"introduction": true}

// notCode are the files no doc's sources name by their name: docs, images.
var notCode = map[string]bool{".md": true, ".mdx": true, ".rst": true, ".txt": true, ".adoc": true,
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".svg": true, ".ico": true,
	".webp": true, ".pdf": true}

// nameKey is a name as names are matched: case, `-` and `_` alike, a
// leading `_` aside.
func nameKey(s string) string {
	return strings.TrimLeft(strings.ReplaceAll(strings.ToLower(s), "-", "_"), "_")
}

// codeByName indexes the repository's code folders and files by name: a
// folder by its own, a file by its name up to the first dot. Folders end
// with a slash.
func codeByName(files map[string]bool, globs []string) map[string][]string {
	seen := map[string]bool{}
	out := map[string][]string{}
	add := func(name, p string) {
		if !seen[p] {
			seen[p] = true
			out[nameKey(name)] = append(out[nameKey(name)], p)
		}
	}
	for f := range files {
		if notCode[strings.ToLower(path.Ext(f))] || matchAny(globs, f) || hidden(f) || isTest(f) {
			continue
		}
		base := path.Base(f)
		if i := strings.Index(base, "."); i > 0 {
			base = base[:i]
		}
		add(base, f)
		for d := path.Dir(f); d != "."; d = path.Dir(d) {
			add(path.Base(d), d+"/")
		}
	}
	return out
}

// isTest says whether a file is a test, or test data: a doc describes the
// code they test (prettier's docs/api.md is not its tests' API.js).
func isTest(f string) bool {
	for _, part := range strings.Split(path.Dir(f), "/") {
		if notDocFolders[strings.ToLower(part)] {
			return true
		}
	}
	base := strings.ToLower(path.Base(f))
	return strings.Contains(base, ".test.") || strings.Contains(base, ".spec.") ||
		strings.Contains(base, "_test.") || strings.HasPrefix(base, "test_")
}

func hidden(f string) bool {
	for _, part := range strings.Split(f, "/") {
		if strings.HasPrefix(part, ".") {
			return true
		}
	}
	return false
}

// byName proposes a doc's sources: its stem, then each folder holding it
// up to a `docs` or `doc` folder, matched to the code of that name, the
// shallowest; nil when none matches, or more than three are as shallow.
func byName(doc string, code map[string][]string) []string {
	var names []string
	stem := strings.TrimSuffix(path.Base(doc), path.Ext(doc))
	if !vagueStems[strings.ToLower(stem)] {
		names = append(names, stem)
	}
	for d := path.Dir(doc); d != "."; d = path.Dir(d) {
		if b := strings.ToLower(path.Base(d)); b == "docs" || b == "doc" {
			break
		}
		names = append(names, path.Base(d))
	}
	for _, n := range names {
		cands := code[nameKey(n)]
		if len(cands) == 0 {
			continue
		}
		depth := func(p string) int { return strings.Count(strings.TrimSuffix(p, "/"), "/") }
		best := -1
		var out []string
		for _, c := range cands {
			switch d := depth(c); {
			case best < 0 || d < best:
				best, out = d, []string{c}
			case d == best:
				out = append(out, c)
			}
		}
		if len(out) > 3 {
			return nil
		}
		sort.Strings(out)
		return out
	}
	return nil
}
