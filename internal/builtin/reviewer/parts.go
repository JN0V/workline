package reviewer

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/role"
)

// A change too large for one call is read in parts (#446), as a doc too
// large for one task is judged (ADR-0009): each part holds what every part
// asks — the lenses, what the author says, the issue the change closes,
// every file it touches by name — then a share of its files, whole or by
// their hunks, packed in order to fill the role's context budget and
// diff-lines-max. A file larger than what is left of a part is cut to fill
// it, every line read once. Each part is asked apart; its findings are
// found again in the whole change and judged as one review's.

// askedAgainChars is the room kept in a part's prompt for an answer asked
// again: what was refused, about a thousand characters when it did not
// read (the product owner keeps the same, #427).
const askedAgainChars = 2000

// unitsRoomMin is the least room a part must leave for the change's
// files; less, and what every part holds is past the budget already.
const unitsRoomMin = 4000

// unit is what a lens reads of one file: whole, the change marked in it,
// or the change's hunks; or a piece of either, cut to fit a part.
type unit struct {
	path  string // the file
	whole bool   // the file whole, the change marked; else its diff
	label string // a whole file's heading: its path, and its lines when cut
	text  string // the marked file, or the diff's lines for it, header first
}

// size is what the unit takes in a task, in characters.
func (u unit) size() int {
	if u.whole {
		return len(wholeHeading(u.label)) + len(u.text) + len("```\n\n")
	}
	return len(u.text)
}

// lines is what the unit counts against diff-lines-max: a diff's lines; a
// file whole counts against code-lines-max, chosen before.
func (u unit) lines() int {
	if u.whole {
		return 0
	}
	return strings.Count(u.text, "\n")
}

func wholeHeading(label string) string { return "### " + label + "\n\n```diff\n" }

// lensMaterial is what a lens reads of the change: what the author says, what
// the change is for, the diff of the files too long to give whole, then
// the files given whole.
func lensMaterial(testimony, purpose string, units []unit) (text string, size int) {
	var diff, code strings.Builder
	for _, u := range units {
		if u.whole {
			code.WriteString(wholeHeading(u.label) + u.text + "```\n\n")
		} else {
			diff.WriteString(u.text)
		}
	}
	d, c := diff.String(), code.String()
	if d == "" {
		d = "(none: every file it changes is given whole below)\n"
	}
	if c == "" {
		c = "(none: each is too long to give whole; its change is above)\n"
	}
	size = len(d) + len(c) // what the finder floor is drawn from
	return fmt.Sprintf("%s\n%s## The change, in the files too long to give whole\n\n```diff\n%s```\n\n## The files it changes, whole, as they read now\n\nThe change is marked in them: `+` a line it added, `-` a line it removed, a space a line it kept. Read them whole: a defect anywhere in them is reported, the change's or not.\n\n%s",
		testimony, purpose, d, c), size
}

// partNote tells a part it holds a share of the change, and names every
// file the change touches, so a lens does not report missing what another
// part holds.
func partNote(k, n int, files []File) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## This part\n\nThe change is too large for one call: it is read in %d parts, each asked apart, each holding a share of its files. This is part %d of %d. Report what you find in this part's files, below. Do not report as missing what another part may hold — a test, a caller, the rest of the change: it is read there.\n\nEvery file the change touches, in every part:\n\n", n, k, n)
	for _, f := range files {
		b.WriteString("- " + f.Path + "\n")
	}
	return b.String() + "\n"
}

// fileDiffs splits a diff into each file's, its header first, in git's
// order: what a part takes whole while it fits.
func fileDiffs(diff string) []unit {
	var out []unit
	for _, line := range strings.SplitAfter(diff, "\n") {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "diff --git ") || len(out) == 0 {
			out = append(out, unit{})
		}
		out[len(out)-1].text += line
	}
	for i := range out {
		out[i].path = diffPath(out[i].text)
	}
	return out
}

// diffPath is the file a diff of one file is about: its path at the end,
// or where it was when removed.
func diffPath(text string) string {
	gone := ""
	for _, line := range strings.Split(text, "\n") {
		switch {
		case strings.HasPrefix(line, "@@"):
			return gone
		case strings.HasPrefix(line, "+++ b/"):
			return strings.TrimPrefix(line, "+++ b/")
		case strings.HasPrefix(line, "--- a/"):
			gone = strings.TrimPrefix(line, "--- a/")
		case strings.HasPrefix(line, "diff --git "):
			if i := strings.LastIndex(line, " b/"); i >= 0 {
				gone = strings.Trim(line[i+3:], `"`)
			}
		}
	}
	return gone
}

// pack shares the units into parts, in order, each within chars
// characters and lines diff lines (0: no limit). A unit that fits a part
// of its own is not cut to fill one; a larger one is cut to fill what is
// left, and a line longer than a part is read alone.
func pack(units []unit, chars, lines int) [][]unit {
	if chars <= 0 {
		chars = math.MaxInt
	}
	if lines <= 0 {
		lines = math.MaxInt
	}
	var parts [][]unit
	var cur []unit
	usedC, usedL := 0, 0
	flush := func() {
		if len(cur) > 0 {
			parts = append(parts, cur)
		}
		cur, usedC, usedL = nil, 0, 0
	}
	add := func(u unit) {
		cur = append(cur, u)
		usedC, usedL = usedC+u.size(), usedL+u.lines()
	}
	queue := append([]unit(nil), units...)
	for len(queue) > 0 {
		u := queue[0]
		if u.size() <= chars-usedC && u.lines() <= lines-usedL {
			add(u)
			queue = queue[1:]
			continue
		}
		if len(cur) > 0 && u.size() <= chars && u.lines() <= lines {
			flush()
			continue
		}
		head, rest, ok := u.cut(chars-usedC, lines-usedL, false)
		if !ok {
			if len(cur) > 0 {
				flush()
				continue
			}
			// A line longer than a part, alone; nothing to cut it by, the
			// unit whole, alone.
			if head, rest, ok = u.cut(chars, lines, true); !ok {
				head, rest = u, unit{path: u.path, whole: u.whole}
			}
		}
		add(head)
		if rest.empty() {
			queue = queue[1:]
		} else {
			queue[0] = rest
		}
		flush()
	}
	flush()
	return parts
}

// cut is the longest head of the unit that fits chars characters and
// lines diff lines, holding at least one line of the file, and the rest;
// false when not even one fits, unless forced: then one line. A whole file
// is cut by its lines, each piece headed with the lines it holds; a diff
// between its hunks, or within one, each piece with the file's header and
// a hunk header of its own.
func (u unit) cut(chars, lines int, force bool) (head, rest unit, ok bool) {
	if u.whole {
		return u.cutWhole(chars, force)
	}
	return u.cutDiff(chars, lines, force)
}

// empty says whether the unit holds no line of its file: what is left of
// one cut at its end.
func (u unit) empty() bool {
	if u.whole {
		return u.text == ""
	}
	return !strings.HasPrefix(u.text, "@@") && !strings.Contains(u.text, "\n@@")
}

func (u unit) cutWhole(chars int, force bool) (head, rest unit, ok bool) {
	all := strings.SplitAfter(u.text, "\n")
	if all[len(all)-1] == "" {
		all = all[:len(all)-1]
	}
	from, total := 1, len(all)
	if f, t, ok := wholeSpan(u.label); ok {
		from, total = f, t
	}
	piece := func(a, b int) unit { // lines a to b of this unit, 0-based, b excluded
		return unit{path: u.path, whole: true, label: fmt.Sprintf("%s, lines %d-%d of %d", u.path, from+a, from+b-1, total),
			text: strings.Join(all[a:b], "")}
	}
	n := 0
	for n < len(all) && piece(0, n+1).size() <= chars {
		n++
	}
	if n == 0 {
		if !force {
			return u, u, false
		}
		n = 1
	}
	if n == len(all) {
		return u, unit{path: u.path, whole: true}, true
	}
	return piece(0, n), piece(n, len(all)), true
}

// wholeSpan reads back the lines a piece of a whole file holds, from its
// label: where it starts, and the file's lines.
func wholeSpan(label string) (from, total int, ok bool) {
	m := spanLabel.FindStringSubmatch(label)
	if m == nil {
		return 0, 0, false
	}
	from, _ = strconv.Atoi(m[1])
	total, _ = strconv.Atoi(m[2])
	return from, total, true
}

var spanLabel = regexp.MustCompile(`, lines (\d+)-\d+ of (\d+)$`)

// hunkHead is a hunk's header, its numbers read to cut it.
var hunkHead = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@(.*)`)

func (u unit) cutDiff(chars, lines int, force bool) (head, rest unit, ok bool) {
	all := strings.SplitAfter(u.text, "\n")
	if all[len(all)-1] == "" {
		all = all[:len(all)-1]
	}
	h := 0 // the file's header: up to its first hunk
	for h < len(all) && !strings.HasPrefix(all[h], "@@") {
		h++
	}
	header := strings.Join(all[:h], "")
	body := all[h:]
	fits := func(text string) bool {
		return len(text) <= chars && strings.Count(text, "\n") <= lines
	}
	// Whole hunks first, while they fit.
	taken, end := header, 0
	for i := 0; i < len(body); {
		j := i + 1
		for j < len(body) && !strings.HasPrefix(body[j], "@@") {
			j++
		}
		if !fits(taken + strings.Join(body[i:j], "")) {
			break
		}
		taken, end, i = taken+strings.Join(body[i:j], ""), j, j
	}
	if end > 0 {
		return unit{path: u.path, text: taken}, unit{path: u.path, text: header + strings.Join(body[end:], "")}, true
	}
	// The first hunk alone does not fit: cut it by its lines.
	if len(body) == 0 {
		return u, u, false
	}
	m := hunkHead.FindStringSubmatch(strings.TrimRight(body[0], "\n"))
	if m == nil {
		return u, u, false
	}
	j := 1
	for j < len(body) && !strings.HasPrefix(body[j], "@@") {
		j++
	}
	hunk := body[1:j]
	oldFrom, _ := strconv.Atoi(m[1])
	newFrom, _ := strconv.Atoi(m[3])
	section := m[5]
	piece := func(lines []string, oldAt, newAt int) string {
		o, n := 0, 0
		for _, l := range lines {
			switch {
			case strings.HasPrefix(l, "-"):
				o++
			case strings.HasPrefix(l, "+"):
				n++
			case strings.HasPrefix(l, `\`): // no newline at end of file
			default:
				o, n = o+1, n+1
			}
		}
		return fmt.Sprintf("@@ -%d,%d +%d,%d @@%s\n", oldAt, o, newAt, n, section) + strings.Join(lines, "")
	}
	k := 0
	for k < len(hunk) && fits(header+piece(hunk[:k+1], oldFrom, newFrom)) {
		k++
	}
	if k == 0 {
		if !force {
			return u, u, false
		}
		k = 1
	}
	if k == len(hunk) { // the hunk whole: its last line, or a header shorter than git's
		return unit{path: u.path, text: header + piece(hunk, oldFrom, newFrom)}, unit{path: u.path, text: header + strings.Join(body[j:], "")}, true
	}
	o, n := 0, 0 // where the rest starts
	for _, l := range hunk[:k] {
		switch {
		case strings.HasPrefix(l, "-"):
			o++
		case strings.HasPrefix(l, "+"):
			n++
		case strings.HasPrefix(l, `\`):
		default:
			o, n = o+1, n+1
		}
	}
	restOld, restNew := oldFrom+o, newFrom+n
	if oldFrom == 0 { // a new file: no old line to start from
		restOld = 0
	}
	if newFrom == 0 { // a file removed
		restNew = 0
	}
	head = unit{path: u.path, text: header + piece(hunk[:k], oldFrom, newFrom)}
	rest = unit{path: u.path, text: header + piece(hunk[k:], restOld, restNew) + strings.Join(body[j:], "")}
	return head, rest, true
}

// taskRoom is the characters a lens's task may hold so the call stays
// within the role's context budget, measured as the engine measures it —
// the same facets, agent.Prompt, agent.Tokens —, room kept for an answer
// asked again; 0 when the role or its budget is not known: no limit but
// diff-lines-max, the engine refusing a prompt too large, said.
func taskRoom(runDir, repo string) (int, error) {
	r, err := role.Load(os.Getenv("WORKLINE_ROLES_DIR"), roleName())
	if err != nil || r.Context.Budget <= 0 {
		return 0, nil
	}
	budget := r.Context.Budget
	asked := *r // a part, as the engine asks it
	asked.Intentions = r.PartAnswers()
	asked.Context.Budget = 0
	dir, err := os.MkdirTemp(filepath.Join(runDir, "in"), "measure-")
	if err != nil {
		return 0, err
	}
	defer os.RemoveAll(dir)
	if err := os.MkdirAll(filepath.Join(dir, "in"), 0o755); err != nil {
		return 0, err
	}
	if err := os.WriteFile(filepath.Join(dir, "in", "task.md"), nil, 0o644); err != nil {
		return 0, err
	}
	system, user, err := agent.Prompt(agent.Request{RunDir: dir, Repo: repo, Role: &asked})
	if err != nil {
		return 0, nil // the engine says why when it asks
	}
	return max(charsWithin(budget)-len(system)-len(user)-askedAgainChars, 1), nil
}

// charsWithin is the most characters a prompt holds within so many
// tokens, by agent.Tokens' own estimate, whatever its ratio.
func charsWithin(budget int) int {
	lo, hi := 0, max(budget, 0)*16
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if agent.Tokens(mid) <= budget {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return lo
}
