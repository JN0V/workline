package documentalist

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
// same place. Any other hunk stays where it says, for misquoted to refuse.
// The hunks keep their order and do not overlap, or none is moved.
func placed(old string, f fileDiff) fileDiff {
	lines := strings.Split(strings.TrimSuffix(old, "\n"), "\n")
	at := func(start int, quoted []string) bool {
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
	out := f
	out.hunks = append([]hunk(nil), f.hunks...)
	end := 0 // the last old line the hunks so far take
	for i, h := range out.hunks {
		var quoted []string
		for _, l := range h.lines {
			if l[0] != '+' {
				quoted = append(quoted, l[1:])
			}
		}
		if h.oldCount > 0 && len(quoted) > 0 && !at(h.oldStart, quoted) {
			found := 0
			for s := 1; s <= len(lines); s++ {
				if at(s, quoted) {
					found++
					if d := s - h.oldStart; d >= -placeWithin && d <= placeWithin {
						out.hunks[i].oldStart = s
					}
				}
			}
			if found != 1 {
				out.hunks[i].oldStart = h.oldStart
			}
		}
		if out.hunks[i].oldStart <= end && h.oldCount > 0 {
			return f
		}
		end = out.hunks[i].oldStart + len(quoted) - 1
	}
	return out
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
// nobody read it whole, so it stays suspect.
func judgePatches(repo string, s Settings, judged map[string]map[string]string, inParts map[string]string, intents []intent.Intention, fallback []intent.Intention, mayGrow bool) ([]verdict.Finding, map[string]bool, error) {
	var refused []verdict.Finding
	refuse := func(rule, where, msg string) {
		refused = append(refused, verdict.Finding{Rule: rule, Where: where, Message: msg})
	}
	patched := map[string]bool{}
	fixed := 0
	var tree Tree
	after := map[string]string{}
	for _, in := range intents {
		if in.Kind != "patch" || isFallback(in, fallback) {
			continue // the role's own regenerated blocks are not the agent's to judge
		}
		if tree.Docs == nil {
			var err error
			if tree, err = loadTree(repo, s.Docs); err != nil {
				return nil, nil, err
			}
			for p, c := range tree.Docs {
				after[p] = c
			}
		}
		diff, ok := in.Value.(string)
		if !ok {
			refuse("patch-not-diff", "patch", "send a unified diff, not a whole file, so what it replaces can be checked against the doc")
			continue
		}
		if _, err := gitIn(repo, intent.NormalizeDiff(diff), "apply", "--recount", "--unidiff-zero", "--check", "-"); err != nil {
			refuse("patch-does-not-apply", "patch", err.Error())
			continue
		}
		files, err := parseDiff(diff)
		if err != nil {
			refuse("patch-unreadable", "patch", err.Error())
			continue
		}
		for _, f := range files {
			want, ok := judged[f.path]
			if !ok {
				refuse("patch-off-task", f.path, "only the docs put before you may be patched; anything else belongs in an issue or a note")
				continue
			}
			old := after[f.path]
			f = placed(old, f)
			if why := misquoted(old, f); why != "" {
				refuse("misquoted", f.path, why)
				continue
			}
			if why := replacedInPart(f); why != "" {
				refuse("replaced-in-part", f.path, why)
				continue
			}
			if touchesDerived(old, f) {
				refuse("derived-block", f.path, "the patch changes lines between workline:derive markers; they are regenerated from the code, never written")
				continue
			}
			now := applyHunks(old, f)
			if isAuthority(s, f.path) && body(now) != body(old) {
				refuse("truth-doc-changed", f.path, "this doc is an authority (truth: doc): the code follows it, never the reverse; set only `checked` and `verified`, and open an issue where the code disagrees")
				continue
			}
			if byGit, err := gitApplied(f.path, old, diff, false); err != nil || byGit != now {
				refuse("patch-ambiguous", f.path, "git would apply this diff differently from how it reads; send a plain unified diff")
				continue
			}
			// The frontmatter records who checked the doc; only the body counts.
			// Truth before size: a fix may say what the code now does, by a
			// tenth of the doc at most; padding beyond that is refused.
			if n, o := len(scan(now)), len(scan(old)); n > o+o/10 && !mayGrow {
				refuse("patch-grows", f.path, fmt.Sprintf("the patch makes the doc %d lines longer, past a tenth of it (%d); fix what is wrong, and say only what the code now does", n-o, o/10))
				continue
			}
			if _, ok := inParts[f.path]; ok {
				if checkedOf(now) != checkedOf(old) {
					refuse("checked-moved-in-parts", f.path, "the patch moves `checked`, but the doc was judged in parts: nobody read it whole against its sources, so `checked` stays where it is; fix what the parts found wrong, and leave the header as it is")
					continue
				}
				after[f.path] = now
				fixed++
				continue
			}
			// A fix the agent could not vouch for is kept: what it found wrong
			// is fixed, and the doc stays suspect (ADR-0012), recording in
			// `judged` when, so that it is not asked again before a source
			// changes. It is in patched as false: fixed, not vouched for.
			if checkedOf(now) == checkedOf(old) {
				if full := want[""]; full != "" && !strings.HasPrefix(full, judgedOf(now)) {
					refuse("judged-not-set", f.path, fmt.Sprintf("the patch leaves `checked`, so it sets `judged: %s`: the commit the doc was judged at, without being vouched for; otherwise it is put before an agent again tomorrow, to the same end", full[:7]))
					continue
				}
				after[f.path] = now
				if !patched[f.path] {
					patched[f.path] = false
				}
				fixed++
				continue
			}
			if !checkedMatches(now, want) {
				refuse("still-suspect", f.path, fmt.Sprintf("the patch does not set `checked` to %s, the commit given in the task, so the doc would stay suspect", wanted(want)))
				continue
			}
			after[f.path] = now
			patched[f.path] = true
			fixed++
		}
	}
	if len(refused) > 0 || fixed == 0 {
		return refused, patched, nil
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
	return refused, patched, nil
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
