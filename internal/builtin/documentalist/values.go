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

// releaseVersions counts the three-part versions lines say.
func releaseVersions(lines ...string) map[string]int {
	out := map[string]int{}
	for _, l := range lines {
		for _, v := range version.FindAllString(l, -1) {
			if strings.Count(v, ".") == 2 {
				out[v]++
			}
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

// linesSaying lists the doc's body lines saying a version, outside derived
// blocks and ranges.
func linesSaying(content, v string) []int {
	skip := derivedLines(content)
	var out []int
	for _, l := range scan(content) {
		if skip[l.n] {
			continue
		}
		for _, at := range version.FindAllStringIndex(l.text, -1) {
			if l.text[at[0]:at[1]] == v && !versionRange.MatchString(l.text[:at[0]]) {
				out = append(out, l.n)
				break
			}
		}
	}
	return out
}

// sourceFilesOf lists the files of this repository a doc names in its
// sources, folders left out: siblings share a file, not a folder.
func sourceFilesOf(d *Doc, files map[string]bool) map[string]bool {
	out := map[string]bool{}
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

// releaseSet is the set of three-part versions a text says.
func releaseSet(text string) map[string]bool {
	out := map[string]bool{}
	for v := range releaseVersions(text) {
		out[v] = true
	}
	return out
}
