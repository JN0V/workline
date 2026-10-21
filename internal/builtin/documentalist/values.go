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

// tagsBefore, tagsAfter and quoteMarkers find markup, read as `*` or a
// backquote are: the tags beside a version — `<code>v1.2.0</code>`,
// `<version>1.2.0</version>` — and the markers opening a blockquote's
// line — `> v1.2.0`, `> > v1.2.0`. The `>` closing a tag or opening a
// quote is no range (#569). A marker has a space after it, so `>= 1.2.0`
// or `>1.2.0` opening a line stays a range. A tag is shaped as one, a name
// then attributes, or is a comment: `v<min && v>1.2.0` is a range.
const tag = `<(?:/?[a-zA-Z][a-zA-Z0-9:-]*(?:\s+[a-zA-Z_:][a-zA-Z0-9_:.-]*(?:\s*=\s*(?:"[^"]*"|'[^']*'|[^\s"'=<>\x60]+))?)*\s*/?|!--.*?--)>`

var (
	tagsBefore   = regexp.MustCompile(`(?:` + tag + `[ \t]*)+(v?)$`)
	tagsAfter    = regexp.MustCompile(`^(?:[ \t]*` + tag + `)+`)
	quoteMarkers = regexp.MustCompile(`^[ \t]*(?:>(?:[ \t]+|$))+`)
)

// versionMarker ends the text before a version that names when something
// came, changed or went — "_First available in v1.9.0_", "Added in",
// "since", "until", "prior to", "New in version", "Deprecated in",
// "default changed from `avoid` to `always` in", Sphinx's `versionadded::`
// and MyST's `{versionadded}`, a name between or not ("prior to pip 18.0")
// — history, not the version now: 39 of the 43 lines reported on prettier
// (docs/research/documentalist-genericity.md). The release a feature came
// in, "from v0.38.0", "the first release after v0.37.0", "fixed in", and
// a minimum, "at least", are history too (#346): true at every later
// release; these, right before the version, "version" alone between
// ("Install from tag v1.2.0", `FROM tool:v1.2.0` pin it). French:
// "depuis", "à partir de", "avant", "après". A version in code quotes or
// plain quotes is read the same, in prose as in a code block's comment.
var versionMarker = regexp.MustCompile(`(?i)(?:^|[^a-z0-9])(?:(?:since|until|prior\s+to|available\s+(?:in|since|from)|(?:added|new|introduced|removed|fixed|shipped|released)\s+in|changed\b[^.;]*?\s+in|deprecated(?:\s+(?:in|since))?|version(?:added|changed|removed))(?:\s+version|\s+[a-z][a-z0-9_-]*)?|(?:from|after|before|at\s+least|minimum|depuis|à\s+partir\s+d(?:e|u)|avant|après)(?:\s+version)?)[\s:*_({}\x60"']*v?$`)

// versionLater starts the text after a version given as a minimum —
// "1.4.1 or later", "and newer", "onwards", `1.4.1+` — which says what is
// needed, true at every later release, not the version now (#346). A `+`
// holding more, `v1.2.0+incompatible`, is a pin. French: "ou plus
// récent", "ou ultérieure", "et suivantes".
var versionLater = regexp.MustCompile(`(?i)^[*_\x60"')]*(?:\+(?:[^0-9a-z.]|$)|\s+(?:or|and)\s+(?:later|newer|above|higher|up)\b|\s+onwards?\b|\s+ou\s+(?:plus\s+récente?|ultérieure?|supérieure?)|\s+et\s+(?:suivantes?|plus)\b)`)

// saidAsNow says whether a version, the text before and after it given,
// is said as the version now — a pin, an example, a version stated — not
// as a range, nor after a history marker, nor as a minimum: in prose as in
// a code block's comment (`# since 1.4.1`).
func saidAsNow(before, after string) bool {
	return !versionRange.MatchString(before) && !versionMarker.MatchString(before) && !versionLater.MatchString(after)
}

// saidAsNowAt says whether the version at at in lines[i] is said as the
// version now (saidAsNow). In a code block, outside its comments, it is
// code the reader copies: a pin whatever word comes before it —
// `since: v1.2.0` is a key's value, `--from v1.2.0` a flag's (#571). The
// history markers are read in prose and in a comment, by the block's
// language (`# since v1.2.0`); a range or a minimum is one in code too.
func saidAsNowAt(lines []line, i int, at []int) bool {
	before, after := around(lines, i, at)
	if l := lines[i]; l.code && !inBlockComment(lines, i, at[0]) {
		return !versionRange.MatchString(before) && !versionLater.MatchString(after)
	}
	return saidAsNow(before, after)
}

// inBlockComment says whether the byte at col of lines[i], a code block's
// line, is in a comment, by the block's language: the block read from its
// first line, so a comment opened on a line before is one too.
func inBlockComment(lines []line, i, col int) bool {
	from := i
	for from > 0 && lines[from-1].code && !isFence(lines[from-1].text) {
		from--
	}
	var b strings.Builder
	for _, l := range lines[from:i] {
		b.WriteString(l.text + "\n")
	}
	offset := b.Len()
	b.WriteString(lines[i].text)
	return maskWith(fenceStyle(lines[i].lang), b.String())[offset+col]
}

// isFence says whether a line opens or closes a fenced code block.
func isFence(text string) bool {
	t := strings.TrimSpace(text)
	return strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~")
}

// around is the text before and after a version found at at in lines[i],
// a sentence wrapped across lines read whole: in prose, a version opening
// its line follows the previous one of its paragraph ("the first release
// after" / "v0.38.0"), one closing it is followed by the next ("v1.2.0" /
// "or later"). Markup is left out of it: the tags beside the version,
// in prose as in code, and in prose the markers opening a blockquote's
// lines (tagsBefore, quoteMarkers), on the lines joined too.
func around(lines []line, i int, at []int) (before, after string) {
	l := lines[i]
	// The markers are read on the whole line: `>1.2.0` opening it is a
	// range, the `>` cut from the version no marker.
	text := func(j int) string {
		if lines[j].code {
			return lines[j].text
		}
		return quoteMarkers.ReplaceAllString(lines[j].text, "")
	}
	q := min(len(l.text)-len(text(i)), at[0])
	before = tagsBefore.ReplaceAllString(l.text[q:at[0]], "${1}")
	after = tagsAfter.ReplaceAllString(l.text[at[1]:], "")
	if l.code {
		return before, after
	}
	prose := func(j int) bool {
		return j >= 0 && j < len(lines) && !lines[j].code && lines[j].heading == 0 && strings.TrimSpace(text(j)) != ""
	}
	// Nothing but markup and the version's `v` on its side of the line.
	if strings.Trim(before, " \t*_`\"'({v") == "" && prose(i-1) {
		before = tagsBefore.ReplaceAllString(text(i-1), "${1}") + " " + strings.TrimLeft(before, " \t")
	}
	if strings.Trim(after, " \t*_`\"')") == "" && prose(i+1) {
		after = " " + strings.TrimLeft(tagsAfter.ReplaceAllString(text(i+1), ""), " \t")
	}
	return before, after
}

// linesSaying lists the doc's body lines saying a version as the version
// now (saidAsNow), outside derived blocks.
func linesSaying(content, v string) []int {
	skip := derivedLines(content)
	var out []int
	lines := scan(content)
	for i, l := range lines {
		if skip[l.n] {
			continue
		}
		for _, at := range versionsIn(l.text) {
			if l.text[at[0]:at[1]] == v && saidAsNowAt(lines, i, at) {
				out = append(out, l.n)
				break
			}
		}
	}
	return out
}

// saysAsNow says whether a doc's body, outside derived blocks, says version
// v as the version now (saidAsNow): a pin the reader installs or copies —
// `WORKLINE_VERSION: v0.38.0`, `go install …@v0.38.0`, an image's tag —
// never the release a feature came in, nor a minimum (#346). v is as a
// release tool bumps it: three parts, a `v` before, maybe a pre-release
// after; a longer number holding it (10.38.0, 0.38.0.1) is not it.
func saysAsNow(content, v string) bool {
	v = strings.TrimPrefix(v, "v")
	digit := func(s string, i int) bool { return i >= 0 && i < len(s) && s[i] >= '0' && s[i] <= '9' }
	skip := derivedLines(content)
	lines := scan(content)
	for n, l := range lines {
		if skip[l.n] {
			continue
		}
		for from := 0; ; {
			i := strings.Index(l.text[from:], v)
			if i < 0 {
				break
			}
			at := []int{from + i, from + i + len(v)}
			from = at[1]
			t := l.text
			if digit(t, at[0]-1) || at[0] > 0 && t[at[0]-1] == '.' || digit(t, at[1]) || at[1] < len(t) && t[at[1]] == '.' && digit(t, at[1]+1) {
				continue
			}
			if saidAsNowAt(lines, n, at) {
				return true
			}
		}
	}
	return false
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
