package documentalist

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/JN0V/workline/internal/verdict"
)

// A replaced value left (ADR-0014, step 2): a version a fix replaces, still
// said in the same doc — a badge's link beside its text (DomoticsCore #113)
// — or in another doc declaring the same source file — siblings left at the
// old version (16c660b). Reported, never fixed by the engine: an old version
// may be said on purpose ("since 1.4.0"), a person reads it.

// fixedDoc is one doc a judged patch changes: each run of lines it removes,
// beside the run it adds in their place.
type fixedDoc struct {
	path   string
	blocks []changeBlock
}

type changeBlock struct{ removed, added []string }

// changeBlocks splits a file's hunks into runs of lines removed and added.
func changeBlocks(f fileDiff) []changeBlock {
	var out []changeBlock
	var cur changeBlock
	flush := func() {
		if len(cur.removed)+len(cur.added) > 0 {
			out = append(out, cur)
		}
		cur = changeBlock{}
	}
	for _, h := range f.hunks {
		for _, l := range h.lines {
			switch {
			case l == "":
				flush()
			case l[0] == '-':
				if len(cur.added) > 0 {
					flush()
				}
				cur.removed = append(cur.removed, l[1:])
			case l[0] == '+':
				cur.added = append(cur.added, l[1:])
			default:
				flush()
			}
		}
		flush()
	}
	return out
}

// Versions: three parts by default (1.4.1, v1.4.1); a project writing them
// otherwise says how (`versions.pattern`: pip's calendar `26.2`), and names
// the files saying its version (`versions.files`). Set once a run, from the
// settings (useVersions).
var versions struct {
	pattern *regexp.Regexp
	files   []string
}

// useVersions takes the project's `versions`; a pattern that does not
// compile is an error, never a check quietly left out.
func useVersions(s Settings) error {
	versions.pattern, versions.files = nil, s.Versions.Files
	if s.Versions.Pattern == "" {
		return nil
	}
	re, err := regexp.Compile(s.Versions.Pattern)
	if err != nil {
		return fmt.Errorf("versions.pattern %q: %v", s.Versions.Pattern, err)
	}
	versions.pattern = re
	return nil
}

// versionsIn finds where a line says a version: three parts, or the
// project's pattern, never part of a longer number (26.2 in 26.2.1).
func versionsIn(l string) [][]int {
	var out [][]int
	if versions.pattern == nil {
		for _, at := range version.FindAllStringIndex(l, -1) {
			if strings.Count(l[at[0]:at[1]], ".") == 2 {
				out = append(out, at)
			}
		}
		return out
	}
	digit := func(i int) bool { return i >= 0 && i < len(l) && l[i] >= '0' && l[i] <= '9' }
	for _, at := range versions.pattern.FindAllStringIndex(l, -1) {
		if at[0] == at[1] || digit(at[0]-1) || at[0] > 1 && l[at[0]-1] == '.' && digit(at[0]-2) ||
			digit(at[1]) || at[1] < len(l) && l[at[1]] == '.' && digit(at[1]+1) {
			continue
		}
		out = append(out, at)
	}
	return out
}

// releaseVersions counts the versions lines say.
func releaseVersions(lines ...string) map[string]int {
	out := map[string]int{}
	for _, l := range lines {
		for _, at := range versionsIn(l) {
			out[l[at[0]:at[1]]]++
		}
	}
	return out
}

// replacedVersions are the versions a fix takes out for others, each with
// the versions written in its place: said fewer times in the lines it adds
// than in those it removes, where the lines it adds in their place say a
// version they did not. A line removed is paired with the line added in its
// place when a run replaces as many lines as it removes; otherwise the runs
// are paired whole. A fix writing no new version replaces nothing.
func replacedVersions(f fixedDoc) map[string]map[string]bool {
	var removed, added []string
	out := map[string]map[string]bool{}
	pair := func(rm, ad []string) {
		was, now := releaseVersions(rm...), releaseVersions(ad...)
		for v, n := range was {
			if now[v] >= n {
				continue
			}
			for w := range now {
				if was[w] == 0 {
					if out[v] == nil {
						out[v] = map[string]bool{}
					}
					out[v][w] = true
				}
			}
		}
	}
	for _, b := range f.blocks {
		removed, added = append(removed, b.removed...), append(added, b.added...)
		if len(b.removed) == len(b.added) {
			for i := range b.removed {
				pair(b.removed[i:i+1], b.added[i:i+1])
			}
		} else {
			pair(b.removed, b.added)
		}
	}
	// What a line still says elsewhere in the fix was not taken out.
	was, now := releaseVersions(removed...), releaseVersions(added...)
	for v := range out {
		if now[v] >= was[v] {
			delete(out, v)
		}
	}
	return out
}

func sortedSet(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// versionRange ends the text before a version given as a range — `>=1.4.1`,
// `^1.4.1`, `~> 1.4` — which says what a dependant accepts, not the version
// now (measured on DomoticsCore: `>=1.13.0` beside Core 1.13.1).
var versionRange = regexp.MustCompile(`(?:>=?|<=?|\^|~>?)\s*v?$`)

// versionMarker ends the text before a version that names when something
// came, changed or went — "_First available in v1.9.0_", "Added in",
// "since", "until", "prior to", "New in version", "Deprecated in",
// "default changed from `avoid` to `always` in", Sphinx's `versionadded::`
// and MyST's `{versionadded}`, a name between or not ("prior to pip 18.0")
// — history, not the version now: 39 of the 43 lines reported on prettier
// (docs/research/documentalist-genericity.md).
var versionMarker = regexp.MustCompile(`(?i)(?:^|[^a-z0-9])(?:since|until|prior\s+to|available\s+(?:in|since|from)|(?:added|new|introduced|removed)\s+in|changed\b[^.;]*?\s+in|deprecated(?:\s+(?:in|since))?|version(?:added|changed|removed))(?:\s+version|\s+[a-z][a-z0-9_-]*)?[\s:*_({}]*v?$`)

// linesSaying lists the doc's body lines saying a version, outside derived
// blocks and ranges.
func linesSaying(content, v string) []int {
	skip := derivedLines(content)
	var out []int
	for _, l := range scan(content) {
		if skip[l.n] {
			continue
		}
		for _, at := range versionsIn(l.text) {
			if before := l.text[:at[0]]; l.text[at[0]:at[1]] == v && !versionRange.MatchString(before) && !versionMarker.MatchString(before) {
				out = append(out, l.n)
				break
			}
		}
	}
	return out
}

// sourceFilesOf lists the files of this repository a doc names in its
// sources, folders left out: siblings share a file, not a folder. The files
// saying the project's version (`versions.files`) are every doc's.
func sourceFilesOf(d *Doc, files map[string]bool) map[string]bool {
	out := map[string]bool{}
	for _, p := range versions.files {
		if files[p] {
			out[p] = true
		}
	}
	for _, src := range d.Sources {
		if name, p, _ := splitSource(src); name == "" && files[p] {
			out[p] = true
		}
	}
	return out
}

func lineList(ns []int) string {
	s := make([]string, len(ns))
	for i, n := range ns {
		s[i] = fmt.Sprint(n)
	}
	if len(ns) == 1 {
		return "line " + s[0]
	}
	return "lines " + strings.Join(s, ", ")
}

// valuesLeft reports, for the fixes judged, each version they replace that
// is still said in the doc fixed, or in another doc declaring one of its
// source files — a file that now says a version written in its place, so
// that 1.4.0 replaced in the Core's row is not looked for in the docs of a
// component that shares nothing but the number. docs holds every doc as it
// will be once the fixes apply; read gives a source file's content.
// History docs — a changelog, decision records, tried and research records
// — are left out: there an old version is the record.
func valuesLeft(fixes []fixedDoc, docs map[string]string, files map[string]bool, read func(string) string) []verdict.Finding {
	var out []verdict.Finding
	said := map[string]bool{}
	report := func(where, v, msg string) {
		if !said[where+" "+v] {
			said[where+" "+v] = true
			out = append(out, verdict.Finding{Rule: "value-left", Where: where, Message: msg})
		}
	}
	others := make([]string, 0, len(docs))
	for p := range docs {
		others = append(others, p)
	}
	sort.Strings(others)
	saysOne := map[string]map[string]bool{} // source file -> the versions it says
	for _, f := range fixes {
		if isHistory(f.path) {
			continue
		}
		replaced := replacedVersions(f)
		if len(replaced) == 0 {
			continue
		}
		var old []string
		for v := range replaced {
			old = append(old, v)
		}
		sort.Strings(old)
		for _, v := range old {
			if at := linesSaying(docs[f.path], v); len(at) > 0 {
				report(f.path, v, fmt.Sprintf("%s, which this fix replaces with %s, is still said on %s: change it there too, or say why it stays", v, strings.Join(sortedSet(replaced[v]), ", "), lineList(at)))
			}
		}
		d, _ := ParseDoc(f.path, []byte(docs[f.path]))
		if d == nil {
			continue
		}
		mine := sourceFilesOf(d, files)
		for _, p := range others {
			if p == f.path || isHistory(p) {
				continue
			}
			o, _ := ParseDoc(p, []byte(docs[p]))
			if o == nil {
				continue
			}
			theirs := sourceFilesOf(o, files)
			for _, v := range old {
				var shared []string
				for src := range theirs {
					if !mine[src] {
						continue
					}
					if saysOne[src] == nil {
						saysOne[src] = releaseSet(read(src))
					}
					for w := range replaced[v] {
						if saysOne[src][w] {
							shared = append(shared, src)
							break
						}
					}
				}
				if len(shared) == 0 {
					continue
				}
				sort.Strings(shared)
				if at := linesSaying(docs[p], v); len(at) > 0 {
					report(p, v, fmt.Sprintf("%s, which the fix of %s replaces with %s, is still said on %s; both declare %s, which says the new one", v, f.path, strings.Join(sortedSet(replaced[v]), ", "), lineList(at), strings.Join(shared, ", ")))
				}
			}
		}
	}
	return out
}

// releaseSet is the set of versions a text says.
func releaseSet(text string) map[string]bool {
	out := map[string]bool{}
	for v := range releaseVersions(text) {
		out[v] = true
	}
	return out
}
