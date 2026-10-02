package documentalist

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/verdict"
)

// fileDiff is one file of a unified diff.
type fileDiff struct {
	path    string
	deleted bool // +++ /dev/null: path is the file the diff deletes
	hunks   []hunk
}

// hunk is one hunk; each line keeps its ' ', '-' or '+' prefix.
type hunk struct {
	oldStart, oldCount int
	lines              []string
}

var hunkHeader = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

// parseDiff reads a unified diff as its lines read: a hunk runs to the next
// header, whatever counts its own header announces — agents often get them
// wrong, and the engine applies with git apply --recount, which reads it the
// same way.
func parseDiff(diff string) ([]fileDiff, error) {
	diff = intent.NormalizeDiff(diff) // read as git will: overlapping hunks merged
	lines := strings.Split(strings.TrimSuffix(diff, "\n"), "\n")
	var out []fileDiff
	for i := 0; i < len(lines); i++ {
		l := lines[i]
		switch {
		case strings.HasPrefix(l, "+++ "):
			p := strings.TrimSpace(strings.TrimPrefix(l, "+++ "))
			p, _, _ = strings.Cut(p, "\t")
			if p == "/dev/null" && i > 0 && strings.HasPrefix(lines[i-1], "--- ") {
				old, _, _ := strings.Cut(strings.TrimSpace(strings.TrimPrefix(lines[i-1], "--- ")), "\t")
				out = append(out, fileDiff{path: strings.TrimPrefix(old, "a/"), deleted: true})
				continue
			}
			out = append(out, fileDiff{path: strings.TrimPrefix(p, "b/")})
		case strings.HasPrefix(l, "@@"):
			m := hunkHeader.FindStringSubmatch(l)
			if m == nil || len(out) == 0 {
				return nil, fmt.Errorf("line %d: a hunk header must follow a file header, as `@@ -a,b +c,d @@`", i+1)
			}
			h := hunk{oldStart: atoi(m[1])}
			for ; i+1 < len(lines) && !diffHeader(lines, i+1); i++ {
				hl := lines[i+1]
				if strings.HasPrefix(hl, `\`) { // "\ No newline at end of file"
					continue
				}
				if hl == "" {
					hl = " " // a blank context line whose space was trimmed
				}
				switch hl[0] {
				case ' ', '-':
					h.oldCount++
				case '+':
				default:
					return nil, fmt.Errorf("the hunk at line %d holds a line starting with neither ' ', '-' nor '+': %q", h.oldStart, hl)
				}
				h.lines = append(h.lines, hl)
			}
			if h.oldStart == 0 && h.oldCount > 0 {
				h.oldStart = 1 // an insertion before the first line, quoting it: git reads it from line 1
			}
			if h.oldCount == 0 && h.oldStart > 0 && m[2] != "0" {
				return nil, fmt.Errorf("the hunk at line %d quotes no line of the doc; give context lines", h.oldStart)
			}
			out[len(out)-1].hunks = append(out[len(out)-1].hunks, h)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no file in the diff")
	}
	return out, nil
}

// diffHeader says whether lines[i] starts a hunk or a file.
func diffHeader(lines []string, i int) bool {
	l := lines[i]
	return strings.HasPrefix(l, "@@") || strings.HasPrefix(l, "diff ") || strings.HasPrefix(l, "index ") ||
		strings.HasPrefix(l, "+++ ") || strings.HasPrefix(l, "--- ") && i+1 < len(lines) && strings.HasPrefix(lines[i+1], "+++ ")
}

func atoi(s string) int { n, _ := strconv.Atoi(s); return n }

// misquoted checks that each hunk's context and removed lines are exactly the
// doc's lines at the numbers the hunk cites (Delfini's check on quotes). git
// apply alone would look for them elsewhere in the file and apply anyway.
func misquoted(old string, f fileDiff) string {
	lines := strings.Split(strings.TrimSuffix(old, "\n"), "\n")
	for _, h := range f.hunks {
		n := h.oldStart
		if h.oldCount == 0 {
			continue // a pure insertion quotes nothing
		}
		for _, l := range h.lines {
			if l[0] == '+' {
				continue
			}
			if n < 1 || n > len(lines) {
				return fmt.Sprintf("the hunk at line %d quotes line %d, but the doc has %d lines", h.oldStart, n, len(lines))
			}
			if lines[n-1] != l[1:] {
				return fmt.Sprintf("the hunk at line %d quotes %q as line %d, but line %d reads %q; cite the lines as numbered in the task", h.oldStart, l[1:], n, n, lines[n-1])
			}
			n++
		}
	}
	return ""
}

// notApplying says where a diff git cannot apply quotes a doc wrong: git
// says only the hunk's line, and agents skipping a blank line, or quoting
// one that is not there, were asked again with that alone (DomoticsCore,
// ADR-0014 step 4).
func notApplying(diff string, docs map[string]string) string {
	files, err := parseDiff(diff)
	if err != nil {
		return "; " + err.Error()
	}
	var why []string
	for _, f := range files {
		if old, ok := docs[f.path]; ok {
			if w := misquoted(old, placed(old, f)); w != "" {
				why = append(why, f.path+": "+w)
			}
		}
	}
	if len(why) == 0 {
		return ""
	}
	return "; " + strings.Join(why, "; ") + " (a blank line counts as a line: quote it as one, and none that is not there)"
}

// version is a version number as docs write them: 2.12.0, v1.4.
var version = regexp.MustCompile(`\d+(?:\.\d+)+`)

// replacedInPart says where a patch replaces a version on a line while the
// line still says the old one elsewhere: a badge's text bumped and its link
// left (DomoticsCore #113). A line removed is paired with the line added in
// its place, in order, when a hunk replaces as many as it removes.
func replacedInPart(f fileDiff) string {
	for _, h := range f.hunks {
		var removed, added []string
		check := func() string {
			if len(removed) != len(added) {
				return ""
			}
			for i, old := range removed {
				now := added[i]
				for _, v := range version.FindAllString(old, -1) {
					if n := strings.Count(now, v); n > 0 && n < strings.Count(old, v) {
						return fmt.Sprintf("a line changes %s, but still says %s elsewhere: %q; change every place on the line that says it — a link as well as its text — or say in a note why one stays", v, v, now)
					}
				}
			}
			return ""
		}
		for _, l := range append(h.lines, " ") {
			switch l[0] {
			case '-':
				if len(added) > 0 {
					if why := check(); why != "" {
						return why
					}
					removed, added = nil, nil
				}
				removed = append(removed, l[1:])
			case '+':
				added = append(added, l[1:])
			default:
				if why := check(); why != "" {
					return why
				}
				removed, added = nil, nil
			}
		}
	}
	return ""
}

// placeWithin is how many lines from where a hunk says it sits the judge
// looks for the lines it quotes: agents miscount by a line or two in long
// docs (DomoticsCore, 2026-10-01), never by their content.
const placeWithin = 3

// placed moves each hunk whose quoted lines are not where it says to where
// they are, when they are within placeWithin lines and found nowhere else
// in the doc: there is then no doubt where it goes, and git apply finds the
// same place. A hunk found nowhere so is mended where its place is still
// beyond doubt (mended). Any other hunk stays where it says, for misquoted
// to refuse. The hunks keep their order and do not overlap, or none is
// moved.
func placed(old string, f fileDiff) fileDiff {
	lines := strings.Split(strings.TrimSuffix(old, "\n"), "\n")
	out := f
	out.hunks = nil
	end := 0 // the last old line the hunks so far take
	for _, h := range f.hunks {
		quoted := oldSide(h)
		hs := []hunk{h}
		if h.oldCount > 0 && len(quoted) > 0 && !quotesAt(lines, h.oldStart, quoted) {
			found, near := 0, 0
			for s := 1; s <= len(lines); s++ {
				if quotesAt(lines, s, quoted) {
					found++
					if d := s - h.oldStart; d >= -placeWithin && d <= placeWithin {
						near = s
					}
				}
			}
			switch {
			case found == 1 && near > 0:
				hs[0].oldStart = near
			case found == 0:
				if m := mended(lines, h); m != nil {
					hs = m
				}
			}
		}
		for _, nh := range hs {
			if nh.oldStart <= end && nh.oldCount > 0 {
				return f
			}
			end = nh.oldStart + len(oldSide(nh)) - 1
			out.hunks = append(out.hunks, nh)
		}
	}
	return out
}

// oldSide are the lines a hunk quotes of the doc: its context and the
// lines it removes.
func oldSide(h hunk) []string {
	var quoted []string
	for _, l := range h.lines {
		if l[0] != '+' {
			quoted = append(quoted, l[1:])
		}
	}
	return quoted
}

// quotesAt says whether the doc's lines from start (from 1) are quoted.
func quotesAt(lines []string, start int, quoted []string) bool {
	if start < 1 || start+len(quoted)-1 > len(lines) {
		return false
	}
	for i, q := range quoted {
		if lines[start-1+i] != q {
			return false
		}
	}
	return true
}

// mended is a hunk quoting the doc wrong written anew, quoting the doc as
// it is, where its place is beyond doubt; nil otherwise. Sonnet skipped a
// blank line, and quoted a context line the doc has not, and the right
// fixes those hunks carried were lost (DomoticsCore, ADR-0014 step 4).
// First, a hunk whose context differs from the doc by blank lines alone
// is placed where its other lines are, found once in the doc: a blank line
// is let go in the context, never in the lines it changes. Else, each run
// of the lines it changes is placed by the lines it removes alone, when
// they are found once in the doc, and the context dropped: a run that only
// adds, or removes only blank lines, has nothing to place it by.
func mended(lines []string, h hunk) []hunk {
	if m, ok := blanksLetGo(lines, h); ok {
		return []hunk{m}
	}
	var runs [][]string
	var cur []string
	for _, l := range append(slices.Clone(h.lines), " ") {
		if l[0] != ' ' {
			cur = append(cur, l)
			continue
		}
		if len(cur) > 0 {
			runs = append(runs, cur)
		}
		cur = nil
	}
	var out []hunk
	for _, run := range runs {
		removed := oldSide(hunk{lines: run})
		if strings.TrimSpace(strings.Join(removed, "")) == "" {
			return nil
		}
		at := 0
		for s := 1; s <= len(lines); s++ {
			if quotesAt(lines, s, removed) {
				if at > 0 {
					return nil
				}
				at = s
			}
		}
		if at == 0 {
			return nil
		}
		out = append(out, hunk{oldStart: at, oldCount: len(removed), lines: run})
	}
	return out
}

// blanksLetGo places a hunk whose context differs from the doc by blank
// lines alone: written anew with the doc's lines as context, when it fits
// the doc at one place only. Between two lines it removes, nothing is let go.
func blanksLetGo(lines []string, h hunk) (hunk, bool) {
	blank := func(s string) bool { return strings.TrimSpace(s) == "" }
	solid := false // a line it quotes that is not blank, to place it by
	for _, l := range h.lines {
		if l[0] != '+' && !blank(l[1:]) {
			solid = true
		}
	}
	if !solid {
		return hunk{}, false
	}
	fit := func(s int) (hunk, int, bool) {
		p, first := s, -1
		var out []string
		lastRemoved := false
		for _, l := range h.lines {
			kind, text := l[0], l[1:]
			if kind == '+' {
				out = append(out, l)
				continue
			}
			if kind == ' ' && blank(text) {
				if p < len(lines) && blank(lines[p]) && first >= 0 {
					out = append(out, " "+lines[p])
					p++
				}
				continue // a blank line the doc has not here, let go
			}
			if first >= 0 && !(kind == '-' && lastRemoved) && !blank(text) {
				for p < len(lines) && blank(lines[p]) {
					out = append(out, " "+lines[p]) // a blank line the hunk skipped
					p++
				}
			}
			if p >= len(lines) || lines[p] != text {
				return hunk{}, 0, false
			}
			if first < 0 {
				first = p
			}
			if kind == ' ' {
				out = append(out, " "+lines[p])
			} else {
				out = append(out, l)
			}
			lastRemoved = kind == '-'
			p++
		}
		n := 0
		for _, l := range out {
			if l[0] != '+' {
				n++
			}
		}
		return hunk{oldStart: first + 1, oldCount: n, lines: out}, first, true
	}
	var got hunk
	found := 0
	for s := 0; s < len(lines); s++ {
		if m, first, ok := fit(s); ok && first == s {
			got = m
			found++
		}
	}
	return got, found == 1
}

// applyHunks returns old with the hunks applied. The hunks were checked by
// misquoted first, so they sit where they say.
func applyHunks(old string, f fileDiff) string {
	lines := strings.Split(strings.TrimSuffix(old, "\n"), "\n")
	var out []string
	next := 0 // index of the first old line not copied yet
	for _, h := range f.hunks {
		start := h.oldStart - 1
		if h.oldCount == 0 {
			start = h.oldStart // inserted after that line
		}
		out = append(out, lines[next:start]...)
		next = start
		for _, l := range h.lines {
			switch l[0] {
			case ' ':
				out = append(out, lines[next])
				next++
			case '-':
				next++
			case '+':
				out = append(out, l[1:])
			}
		}
	}
	out = append(out, lines[next:]...)
	return strings.Join(out, "\n") + "\n"
}

// touchesDerived says whether a hunk changes a line between workline:derive
// markers: those lines are regenerated from the code, never written.
func touchesDerived(old string, f fileDiff) bool {
	lines := strings.Split(old, "\n")
	inside := make([]bool, len(lines)+2)
	in := false
	for i, l := range lines {
		if strings.Contains(l, "<!-- workline:derive") {
			in = true
		}
		inside[i+1] = in
		if strings.Contains(l, "<!-- workline:end") {
			in = false
		}
	}
	for _, h := range f.hunks {
		n := h.oldStart
		for _, l := range h.lines {
			switch l[0] {
			case ' ':
				n++
			case '-':
				if n < len(inside) && inside[n] {
					return true
				}
				n++
			case '+':
				if n > 1 && n-1 < len(inside) && inside[n-1] && n < len(inside) && inside[n] {
					return true // inserted between two lines of a derived block
				}
			}
		}
	}
	return false
}

// checkedMatches says whether a doc's `checked`, as patched, names the commits
// it was judged against: each one given at least by its 7-character prefix.
func checkedMatches(content string, want map[string]string) bool {
	d, err := ParseDoc("", []byte(content))
	if err != nil || d == nil {
		return false
	}
	for repo, full := range want {
		got := d.Checked[repo]
		if len(got) < 7 || !strings.HasPrefix(full, got) {
			return false
		}
	}
	return true
}

// judgePatches checks each proposed patch: a diff the engine can apply, on the
// docs put before the agent only, quoting them at the lines it cites, leaving
// derived blocks alone, not growing the doc, recording the commits the doc was
// judged against, and bringing no new budget, link or duplicate problem. It
// returns the refusals, and the docs the patches handle.
// A doc judged in parts (inParts) is fixed with `checked` left where it is:
// nobody read it whole, so it stays suspect. So is a doc whose sources did
// not all go whole into the task (not in whole, ADR-0014): the agent cannot
// vouch for what it was not given. Nor may `checked` move while the doc
// states a line count off (ADR-0014, step 2). What the patches accepted do
// is returned too, for the checks on the docs as they will be. Every word a
// fix takes out of a body is cited, never by a comment alone (cite.go); a
// comment given alone is returned in reported, for a person.
//
// A place refused by the removal rule, a hunk quoting the doc wrong, or a
// move of `checked` nobody can vouch for is not asked for again: the rest
// of the doc's fix holds and is applied without it, the doc keeps
// `checked` and records `judged`, and the place is withheld — a finding
// for a person, with its reason. A re-ask sent the whole task again for
// one place, and rarely brought the right fix back (ADR-0014, step 4).
// Only a patch of which no hunk quotes the doc right is asked for again: a
// misquote is a slip the agent mends at the second answer.
func judgePatches(repo string, s Settings, judged map[string]map[string]string, inParts map[string]string, whole map[string]bool, intents []intent.Intention, fallback []intent.Intention, mayGrow bool) (j patchJudgement, err error) {
	refuse := func(rule, where, msg string) {
		j.refused = append(j.refused, verdict.Finding{Rule: rule, Where: where, Message: msg})
	}
	j.patched = map[string]bool{}
	j.partly = map[string]bool{}
	fixed := 0
	var tree Tree
	claims, unread := readCitations(intents)
	agentDocs := map[string]bool{}
	for _, in := range intents {
		if in.Kind == "patch" && !isFallback(in, fallback) {
			for _, p := range intent.PatchFiles(in.Value) {
				agentDocs[p] = true
			}
		}
	}
	var cc *citeContext
	after := map[string]string{}
	var fixes []fixedDoc
	accept := func(f fileDiff) {
		fixes = append(fixes, fixedDoc{path: f.path, blocks: changeBlocks(f)})
	}
	// judgeFile reads one doc's part of a patch: refused whole (rule), or
	// refused in places a narrower fix may leave out, or in its move of
	// `checked` alone (header); otherwise what the doc becomes.
	judgeFile := func(f fileDiff, old string, want map[string]string, diff string) fileVerdict {
		if why := misquoted(old, f); why != "" {
			return fileVerdict{rule: "misquoted", why: why}
		}
		if why := replacedInPart(f); why != "" {
			return fileVerdict{rule: "replaced-in-part", why: why}
		}
		if touchesDerived(old, f) {
			return fileVerdict{rule: "derived-block", why: "the patch changes lines between workline:derive markers; they are regenerated from the code, never written"}
		}
		now := applyHunks(old, f)
		if isAuthority(s, f.path) && body(now) != body(old) {
			return fileVerdict{rule: "truth-doc-changed", why: "this doc is an authority (truth: doc): the code follows it, never the reverse; set only `checked` and `verified`, and open an issue where the code disagrees"}
		}
		places, held, off, comments := cc.citationRefusals(f.path, old, f, claims, unread, len(agentDocs) == 1)
		j.reported = append(j.reported, comments...)
		if len(places) > 0 {
			rule, why := citationMessage(f.path, places, held, off)
			return fileVerdict{rule: rule, why: why, places: places}
		}
		if byGit, err := gitApplied(f.path, old, diff, false); err != nil || byGit != now {
			return fileVerdict{rule: "patch-ambiguous", why: "git would apply this diff differently from how it reads; send a plain unified diff"}
		}
		// The frontmatter records who checked the doc; only the body counts.
		// Truth before size: a fix may say what the code now does, by a
		// tenth of the doc at most; padding beyond that is refused.
		if n, o := len(scan(now)), len(scan(old)); n > o+o/10 && !mayGrow {
			return fileVerdict{rule: "patch-grows", why: fmt.Sprintf("the patch makes the doc %d lines longer, past a tenth of it (%d); fix what is wrong, and say only what the code now does", n-o, o/10)}
		}
		moved := checkedOf(now) != checkedOf(old)
		if _, ok := inParts[f.path]; ok {
			if moved {
				return fileVerdict{rule: "checked-moved-in-parts", why: "the patch moves `checked`, but the doc was judged in parts: nobody read it whole against its sources, so `checked` stays where it is; fix what the parts found wrong, and leave the header as it is"}
			}
			return fileVerdict{now: now, inParts: true}
		}
		full := want[""]
		if moved && !whole[f.path] {
			msg := "the patch moves `checked`, but not every source of the doc was given whole in the task: nobody can vouch for what was not read, so `checked` stays where it is"
			if full != "" {
				msg += fmt.Sprintf("; fix what you found wrong, and set `judged: %s` instead", full[:7])
			}
			return fileVerdict{rule: "checked-unread", why: msg, header: true,
				person: "`checked` not moved: not every source of the doc was given whole in the task, so nobody could vouch for it; `judged` set instead"}
		}
		if moved {
			d, _ := ParseDoc(f.path, []byte(now))
			if off := countsOff(repo, tree.Files, d, now); len(off) > 0 {
				var said []string
				for _, c := range off {
					said = append(said, c.message())
				}
				msg := "the patch moves `checked`, but the doc still states a line count off, which nobody can vouch for: " + strings.Join(said, "; ") + "; bring each count to the engine's number, with no claim: the engine's count is the evidence"
				if full != "" {
					msg += fmt.Sprintf(", or leave `checked` and set `judged: %s`", full[:7])
				}
				return fileVerdict{rule: "checked-over-count-off", why: msg, header: true,
					person: "`checked` not moved: the doc still states a line count off (" + strings.Join(said, "; ") + "); `judged` set instead"}
			}
		}
		// A fix the agent could not vouch for is kept: what it found wrong
		// is fixed, and the doc stays suspect (ADR-0012), recording in
		// `judged` when, so that it is not asked again before a source
		// changes.
		if !moved {
			if full != "" && !strings.HasPrefix(full, judgedOf(now)) {
				return fileVerdict{rule: "judged-not-set", why: fmt.Sprintf("the patch leaves `checked`, so it sets `judged: %s`: the commit the doc was judged at, without being vouched for; otherwise it is put before an agent again tomorrow, to the same end", full[:7])}
			}
			return fileVerdict{now: now}
		}
		if !checkedMatches(now, want) {
			return fileVerdict{rule: "still-suspect", why: fmt.Sprintf("the patch does not set `checked` to %s, the commit given in the task, so the doc would stay suspect", wanted(want))}
		}
		return fileVerdict{now: now, vouched: true}
	}
	narrowedAny := false
	for ii, in := range intents {
		if in.Kind != "patch" || isFallback(in, fallback) {
			continue // the role's own regenerated blocks are not the agent's to judge
		}
		if tree.Docs == nil {
			if tree, err = loadTree(repo, s.Docs); err != nil {
				return j, err
			}
			for p, c := range tree.Docs {
				after[p] = c
			}
			if cc, err = newCiteContext(repo, tree.Files); err != nil {
				return j, err
			}
			if len(agentDocs) > 1 {
				j.reported = append(j.reported, cc.attribute(claims, tree.Docs, agentFiles(intents, fallback, tree.Docs, judged))...)
			}
		}
		diff, ok := in.Value.(string)
		if !ok {
			refuse("patch-not-diff", "patch", "send a unified diff, not a whole file, so what it replaces can be checked against the doc")
			continue
		}
		narrowed := false
		var misquotes map[string][]verdict.Finding
		if _, err := gitIn(repo, intent.NormalizeDiff(diff), "apply", "--recount", "--unidiff-zero", "--check", "-"); err != nil {
			kept, dropped, ok := quotedRight(diff, after, judged)
			if ok {
				_, again := gitIn(repo, kept, "apply", "--recount", "--unidiff-zero", "--check", "-")
				ok = again == nil
			}
			if !ok {
				refuse("patch-does-not-apply", "patch", err.Error()+notApplying(diff, after))
				continue
			}
			diff, misquotes, narrowed = kept, dropped, true
			for _, p := range sortedKeys2(dropped) {
				j.withheld = append(j.withheld, dropped[p]...)
				j.partly[p] = true
			}
		}
		files, err := parseDiff(diff)
		if err != nil {
			refuse("patch-unreadable", "patch", err.Error())
			continue
		}
		start := map[string]string{}
		var order []string
		refusedBefore := len(j.refused)
		for _, f := range files {
			want, ok := judged[f.path]
			if !ok {
				refuse("patch-off-task", f.path, "only the docs put before you may be patched; anything else belongs in an issue or a note")
				continue
			}
			old := after[f.path]
			if _, seen := start[f.path]; !seen {
				start[f.path] = old
				order = append(order, f.path)
			}
			f = placed(old, f)
			fdiff := diff
			partly := len(misquotes[f.path]) > 0
			for tries := 0; ; tries++ {
				v := judgeFile(f, old, want, fdiff)
				if v.rule == "" && !(partly && v.vouched) {
					after[f.path] = v.now
					accept(f)
					fixed++
					switch {
					case v.inParts:
					case v.vouched:
						j.patched[f.path] = true
					case !j.patched[f.path]:
						j.patched[f.path] = false
					}
					break
				}
				// Narrowed: the places refused taken back, `checked` put
				// back and `judged` recorded; then judged again.
				if v.rule != "" && len(v.places) == 0 && !v.header || tries > len(f.hunks)+8 {
					refuse(v.rule, f.path, v.why)
					break
				}
				var drop []citedBlock
				for _, p := range v.places {
					drop = append(drop, p.block)
				}
				nf := withoutBlocks(f, drop)
				now, ok := asJudged(old, applyHunks(old, nf), want[""])
				if !ok {
					refuse(v.rule, f.path, v.why)
					break
				}
				for _, p := range v.places {
					j.withheld = append(j.withheld, verdict.Finding{Rule: p.rule, Where: f.path,
						Message: p.why + "; left as it was, the rest of the fix applied: a person checks this place against the sources"})
				}
				if v.header {
					j.withheld = append(j.withheld, verdict.Finding{Rule: v.rule, Where: f.path, Message: v.person})
				}
				partly, narrowed = true, true
				j.partly[f.path] = true
				if now == old {
					break // nothing of the fix left, not even its record
				}
				d, err := unifiedDiff(f.path, old, now, 3)
				if err != nil {
					return j, err
				}
				nfs, err := parseDiff(d)
				if err != nil {
					return j, err
				}
				f, fdiff = placed(old, nfs[0]), d
			}
		}
		if narrowed && len(j.refused) == refusedBefore {
			var b strings.Builder
			for _, p := range order {
				if after[p] == start[p] {
					continue
				}
				d, err := unifiedDiff(p, start[p], after[p], 3)
				if err != nil {
					return j, err
				}
				b.WriteString(d)
			}
			intents[ii].Value = b.String()
			narrowedAny = true
		}
	}
	if narrowedAny {
		j.intents = slices.DeleteFunc(slices.Clone(intents), func(in intent.Intention) bool { return in.Kind == "patch" && in.Value == "" })
	}
	if len(j.refused) > 0 || fixed == 0 {
		return j, nil
	}
	before := map[string]Problem{}
	for _, p := range Hygiene(tree, s.Budgets, s.Duplicates) {
		before[p.Key] = p
	}
	for _, p := range Hygiene(Tree{Docs: after, Files: tree.Files}, s.Budgets, s.Duplicates) {
		// A budget already exceeded may grow by what a fix allows: it stays
		// reported, for condensing. Any other problem, or a new one, refuses.
		if old, ok := before[p.Key]; !ok || p.Size > old.Size && !budgetRule[p.Rule] {
			refuse("patch-introduces", p.Where, p.Rule+": "+p.Message)
		}
	}
	j.fix = &judgedFixes{repo: repo, files: tree.Files, before: tree.Docs, after: after, fixes: fixes}
	return j, nil
}

// agentFiles are the docs' parts of the agent's patches, each placed on
// the doc as it is; a patch that cannot be read gives none.
func agentFiles(intents, fallback []intent.Intention, docs map[string]string, judged map[string]map[string]string) []fileDiff {
	var out []fileDiff
	for _, in := range intents {
		diff, ok := in.Value.(string)
		if in.Kind != "patch" || !ok || isFallback(in, fallback) {
			continue
		}
		files, err := parseDiff(diff)
		if err != nil {
			continue
		}
		for _, f := range files {
			old, isDoc := docs[f.path]
			if _, asked := judged[f.path]; isDoc && asked && !f.deleted {
				out = append(out, placed(old, f))
			}
		}
	}
	return out
}

// patchJudgement is what judgePatches finds of the agent's patches.
type patchJudgement struct {
	refused  []verdict.Finding // asked for again
	patched  map[string]bool   // the docs fixed: true when vouched for
	partly   map[string]bool   // the docs fixed with places withheld
	fix      *judgedFixes
	reported []verdict.Finding  // comments given as the only reason, claims read for no doc: for a person
	withheld []verdict.Finding  // places refused and left as they were, for a person
	intents  []intent.Intention // the proposals narrowed to what holds; nil when unchanged
}

// fileVerdict is what judgePatches makes of one doc's part of a patch.
type fileVerdict struct {
	rule, why string         // refused, and why, as the agent is told
	places    []placeRefusal // refused in these places alone
	header    bool           // refused in its move of `checked` alone
	person    string         // why, for a person, when header
	now       string         // the doc as fixed, when not refused
	vouched   bool           // `checked` moves
	inParts   bool           // judged in parts: neither vouched nor recorded here
}

// withoutBlocks is a doc's diff with the runs refused taken back: the lines
// they remove kept as context, the lines they add dropped.
func withoutBlocks(f fileDiff, drop []citedBlock) fileDiff {
	skip := map[[2]int]bool{}
	for _, b := range drop {
		for _, i := range b.at {
			skip[[2]int{b.hunk, i}] = true
		}
	}
	out := f
	out.hunks = nil
	for hi, h := range f.hunks {
		nh := hunk{oldStart: h.oldStart, oldCount: h.oldCount}
		for li, l := range h.lines {
			if skip[[2]int{hi, li}] {
				if l[0] == '-' {
					nh.lines = append(nh.lines, " "+l[1:])
				}
				continue
			}
			nh.lines = append(nh.lines, l)
		}
		out.hunks = append(out.hunks, nh)
	}
	return out
}

// asJudged is a doc as a fix leaves its body, under its header as it was
// with `judged` recorded: a fix applied in part vouches for nothing, so
// `checked` stays. ok is false when the doc has no header to record it in,
// or the commit is not known.
func asJudged(old, now, full string) (string, bool) {
	_, n := header(old)
	_, m := header(now)
	if n == 0 || m == 0 || len(full) < 7 {
		return "", false
	}
	head := slices.Clone(strings.Split(old, "\n")[:n])
	set := false
	for i := 1; i < n-1; i++ {
		if strings.HasPrefix(head[i], "judged:") {
			head[i], set = "judged: "+full[:7], true
		}
	}
	if !set {
		head = slices.Insert(head, n-1, "judged: "+full[:7])
	}
	return strings.Join(append(head, strings.Split(now, "\n")[m:]...), "\n"), true
}

// quotedRight keeps, of a diff git cannot apply, the hunks that quote the
// docs right once placed — a hunk mended where its place is beyond doubt
// among them — each doc's diff written anew from them; the hunks left out
// are returned by doc, each with what it misquotes, for a person. ok is
// false when no hunk quotes right, or the diff touches a file that is no
// doc put before the agent: then nothing holds.
func quotedRight(diff string, docs map[string]string, judged map[string]map[string]string) (kept string, dropped map[string][]verdict.Finding, ok bool) {
	files, err := parseDiff(diff)
	if err != nil {
		return "", nil, false
	}
	dropped = map[string][]verdict.Finding{}
	var b strings.Builder
	for _, f := range files {
		old, isDoc := docs[f.path]
		if _, asked := judged[f.path]; !isDoc || !asked || f.deleted {
			return "", nil, false
		}
		f = placed(old, f)
		var keep []hunk
		for _, h := range f.hunks {
			if why := misquoted(old, fileDiff{path: f.path, hunks: []hunk{h}}); why != "" {
				dropped[f.path] = append(dropped[f.path], verdict.Finding{Rule: "patch-does-not-apply", Where: f.path,
					Message: strings.TrimSuffix(why, "; cite the lines as numbered in the task") + "; that hunk left out, the rest of the fix applied: a person checks this place against the sources"})
				continue
			}
			keep = append(keep, h)
		}
		if len(keep) == 0 {
			continue
		}
		d, err := unifiedDiff(f.path, old, applyHunks(old, fileDiff{path: f.path, hunks: keep}), 3)
		if err != nil {
			return "", nil, false
		}
		b.WriteString(d)
	}
	if b.Len() == 0 {
		return "", nil, false
	}
	return b.String(), dropped, true
}

func sortedKeys2(m map[string][]verdict.Finding) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// judgedFixes is what the accepted patches do: every doc before and after
// them, and the lines each changed.
type judgedFixes struct {
	repo          string
	files         map[string]bool
	before, after map[string]string
	fixes         []fixedDoc
}

// placesFixed counts the places the accepted patches changed in a doc's
// body — each run of lines changed together — the header left out: a patch
// recording only `judged` or `checked` fixed nothing.
func (j *judgedFixes) placesFixed(path string) int {
	if j == nil {
		return 0
	}
	now, ok := j.after[path]
	if !ok || body(now) == body(j.before[path]) {
		return 0
	}
	d, err := zeroContextDiff(path, body(j.before[path]), body(now))
	if err != nil {
		return 1 // they differ; how much, the diff could not say
	}
	n := 0
	for _, l := range strings.Split(d, "\n") {
		if strings.HasPrefix(l, "@@ ") {
			n++
		}
	}
	return max(n, 1)
}

// nPlaces says n places, in words.
func nPlaces(n int) string {
	if n == 1 {
		return "1 place"
	}
	return fmt.Sprintf("%d places", n)
}

// countFixed says whether a count-off finding of a doc is one its fix
// brought right: the doc said it before, and says it no more.
func (j *judgedFixes) countFixed(f verdict.Finding) bool {
	if j == nil || f.Rule != "count-off" {
		return false
	}
	now, ok := j.after[f.Where]
	if !ok || now == j.before[f.Where] {
		return false
	}
	key := func(c countOff) string { return c.file + " " + c.stated }
	was := ""
	d, _ := ParseDoc(f.Where, []byte(j.before[f.Where]))
	for _, c := range countsOff(j.repo, j.files, d, j.before[f.Where]) {
		if c.message() == f.Message {
			was = key(c)
		}
	}
	if was == "" {
		return false
	}
	d, _ = ParseDoc(f.Where, []byte(now))
	for _, c := range countsOff(j.repo, j.files, d, now) {
		if key(c) == was {
			return false
		}
	}
	return true
}

// checkedOf is what a doc's header says in `checked`, as written.
func checkedOf(content string) string {
	d, err := ParseDoc("", []byte(content))
	if err != nil || d == nil {
		return ""
	}
	return fmt.Sprint(d.Checked)
}

// judgedOf is what a doc's header says in `judged`, as written; never empty
// for a doc whose header says nothing of it, so it is no prefix of a commit.
func judgedOf(content string) string {
	if d, err := ParseDoc("", []byte(content)); err == nil && d != nil && len(d.Judged) >= 7 {
		return d.Judged
	}
	return "-"
}

// budgetRule are the size rules a fix may make a little worse, within what
// patch-grows allows a doc.
var budgetRule = map[string]bool{"doc-too-long": true, "section-too-long": true, "folder-too-long": true, "card-too-long": true}

// gitApplied returns what git apply makes of one file of a diff, so the judge
// judges exactly what the engine will apply. A new file is created by the diff.
func gitApplied(path, old, diff string, isNew bool) (string, error) {
	dir, err := os.MkdirTemp("", "workline-judge-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)
	file := filepath.Join(dir, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return "", err
	}
	if !isNew {
		if err := os.WriteFile(file, []byte(old), 0o644); err != nil {
			return "", err
		}
	}
	if _, err := gitIn(dir, intent.NormalizeDiff(diff), "apply", "--recount", "--unidiff-zero", "--include="+path, "-"); err != nil {
		return "", err
	}
	data, err := os.ReadFile(file)
	return string(data), err
}

// wanted names the commits a doc's `checked` must hold, as the task gave them.
func wanted(want map[string]string) string {
	var parts []string
	for repo, full := range want {
		if repo == "" {
			parts = append(parts, full[:7])
		} else {
			parts = append(parts, repo+": "+full[:7])
		}
	}
	sort.Strings(parts)
	return strings.Join(parts, ", ")
}

// isFallback says whether a patch is one pre proposed itself, unchanged.
func isFallback(in intent.Intention, fallback []intent.Intention) bool {
	for _, f := range fallback {
		if f.Kind == in.Kind && fmt.Sprint(f.Value) == fmt.Sprint(in.Value) {
			return true
		}
	}
	return false
}
