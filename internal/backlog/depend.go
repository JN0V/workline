package backlog

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/forge"
)

// What an issue waits on (ADR-0028): the forge's own relation where it has
// one, a line in the body elsewhere — `Blocked by #12, #13.` — the engine's
// ending with BlockedByMarker. Both are read back, a person's line too.

// BlockedByMarker ends the line the engine keeps in a body on a forge
// without the relation.
var BlockedByMarker = forge.Marker("blocked-by")

// maxBlockers bounds the blockers one act names.
const maxBlockers = 5

// blockedLine is a body's line saying what the issue waits on: "Blocked
// by" (case aside, a colon allowed) then issue references.
var blockedLine = regexp.MustCompile(`(?im)^[ \t>*_-]*blocked by:?[ \t]+((?:#\d+[^#\n]*)+)$`)

var issueRef = regexp.MustCompile(`#(\d+)\b`)

// BodyBlockers are the issues a body's "Blocked by" lines name.
func BodyBlockers(body string) []int {
	var out []int
	for _, m := range blockedLine.FindAllStringSubmatch(body, -1) {
		for _, r := range issueRef.FindAllStringSubmatch(m[1], -1) {
			if n, err := strconv.Atoi(r[1]); err == nil && !slices.Contains(out, n) {
				out = append(out, n)
			}
		}
	}
	return out
}

// Blockers are the issues one waits on, open or closed: the forge's
// relation and its body's lines, by number.
func Blockers(is forge.Issue) []int {
	out := slices.Clone(is.BlockedBy)
	for _, n := range BodyBlockers(is.Body) {
		if !slices.Contains(out, n) {
			out = append(out, n)
		}
	}
	out = slices.DeleteFunc(out, func(n int) bool { return n == is.ID })
	slices.Sort(out)
	return out
}

// WithBlockers is a body with the engine's line naming these blockers too:
// the line there rewritten with them added, or added after the text. A
// person's own "Blocked by" line is left as it is.
func WithBlockers(body string, add []int) string {
	lines := strings.Split(body, "\n")
	for i, l := range lines {
		if strings.Contains(l, BlockedByMarker) {
			ids := BodyBlockers(l)
			for _, n := range add {
				if !slices.Contains(ids, n) {
					ids = append(ids, n)
				}
			}
			lines[i] = blockedByLine(ids)
			return strings.Join(lines, "\n")
		}
	}
	text := strings.TrimRight(body, "\n")
	if text != "" {
		text += "\n\n"
	}
	return text + blockedByLine(add)
}

func blockedByLine(ids []int) string {
	ids = slices.Clone(ids)
	slices.Sort(ids)
	return "Blocked by " + issueList(ids) + ". " + BlockedByMarker
}

// Waiting are the open issues one waits on: a blocker closed, or not an
// open issue, holds nothing back.
func Waiting(is forge.Issue, open map[int]bool) []int {
	var out []int
	for _, n := range Blockers(is) {
		if open[n] {
			out = append(out, n)
		}
	}
	return out
}

// Order sorts issues in the backlog's order (docs/spec/backlog-acts.md,
// "Ordering"): Kahn's, the next the first by Less whose blockers among
// them are all placed. When none can be, those left hold a cycle: it is
// returned, the cycle's first by Less placed, and the order goes on — a
// cycle is reported, never followed (ADR-0028).
func Order(issues []forge.Issue) (cycles [][]int) {
	sort.SliceStable(issues, func(i, j int) bool { return Less(issues[i], issues[j]) })
	in := map[int]bool{}
	for _, is := range issues {
		in[is.ID] = true
	}
	placed := map[int]bool{}
	left := slices.Clone(issues)
	out := make([]forge.Issue, 0, len(issues))
	free := func(is forge.Issue) bool {
		for _, b := range Blockers(is) {
			if in[b] && !placed[b] {
				return false
			}
		}
		return true
	}
	for len(left) > 0 {
		i := slices.IndexFunc(left, free)
		if i < 0 {
			// The first left may only wait on a cycle: the cycle's first
			// member by Less is placed, never an issue outside it.
			c := cycleFrom(left[0], left, placed)
			if len(c) > 0 && !slices.ContainsFunc(cycles, func(o []int) bool { return sameCycle(o, c) }) {
				cycles = append(cycles, c)
			}
			i = max(0, slices.IndexFunc(left, func(is forge.Issue) bool { return slices.Contains(c, is.ID) }))
		}
		placed[left[i].ID] = true
		out = append(out, left[i])
		left = slices.Delete(left, i, i+1)
	}
	copy(issues, out)
	return cycles
}

// cycleFrom follows, from an issue, the first blocker not placed, until an
// issue comes back: the cycle, from that issue.
func cycleFrom(start forge.Issue, left []forge.Issue, placed map[int]bool) []int {
	byID := map[int]forge.Issue{}
	for _, is := range left {
		byID[is.ID] = is
	}
	var path []int
	at := start
	for {
		if i := slices.Index(path, at.ID); i >= 0 {
			return append(slices.Clone(path[i:]), at.ID)
		}
		path = append(path, at.ID)
		next, found := forge.Issue{}, false
		for _, b := range Blockers(at) {
			if is, ok := byID[b]; ok && !placed[b] {
				next, found = is, true
				break
			}
		}
		if !found {
			return nil
		}
		at = next
	}
}

// sameCycle says whether two cycles hold the same issues.
func sameCycle(a, b []int) bool {
	x, y := slices.Clone(a[:len(a)-1]), slices.Clone(b[:len(b)-1])
	slices.Sort(x)
	slices.Sort(y)
	return slices.Equal(x, y)
}

// CycleText says a cycle: "#12 waits on #14, which waits on #12".
func CycleText(c []int) string {
	var parts []string
	for i := 0; i+1 < len(c); i++ {
		parts = append(parts, fmt.Sprintf("#%d waits on #%d", c[i], c[i+1]))
	}
	return strings.Join(parts, ", ")
}

// NextReady is the first issue of an ordered list bearing workline:ready
// that waits on no open issue: the one offered to whoever builds next
// (ADR-0028); nil when there is none. A parent is never offered: its
// parts are what is built, and a person accepts it (ADR-0029).
func NextReady(ordered []forge.Issue, open map[int]bool) *forge.Issue {
	for i, is := range ordered {
		if Offered(is, open) {
			return &ordered[i]
		}
	}
	return nil
}

// checkDepend checks a depend act against the backlog as it is (ADR-0028):
// the issue and each blocker open, not itself, 1 to 5; only those not
// there already kept; none that would close a cycle.
func (p *Plan) checkDepend(c *Proposal) (rule, why string) {
	if c.Do != "depend" {
		return "", ""
	}
	is, ok := p.issues[c.Issue]
	if !ok || c.Issue == p.Report {
		return "no-state", fmt.Sprintf("#%d is not an open issue", c.Issue)
	}
	if len(c.BlockedBy) == 0 || len(c.BlockedBy) > maxBlockers {
		return "depend-issue", fmt.Sprintf("an issue waits on 1 to %d issues (blocked-by)", maxBlockers)
	}
	have := append(Blockers(is), p.added[c.Issue]...)
	closing := p.closedInRun()
	var add []int
	for _, b := range c.BlockedBy {
		switch {
		case b == c.Issue:
			return "depend-issue", "an issue does not wait on itself"
		case !p.open[b] || b == p.Report || closing[b]:
			return "depend-issue", fmt.Sprintf("#%d is not an open issue: a closed one holds nothing back", b)
		case !slices.Contains(have, b) && !slices.Contains(add, b):
			add = append(add, b)
		}
	}
	if len(add) == 0 {
		return "depend-same", fmt.Sprintf("it waits on %s already", issueList(c.BlockedBy))
	}
	c.BlockedBy = add
	edges := map[int][]int{}
	for id, o := range p.issues {
		edges[id] = append(Blockers(o), p.added[id]...)
	}
	for _, b := range add {
		if reaches(edges, b, c.Issue) {
			return "depend-cycle", fmt.Sprintf("#%d waits on #%d already, directly or through others: a cycle, never written", b, c.Issue)
		}
	}
	return "", ""
}

// Backlog is what the order says of the open issues: the first ready one
// offered, those waiting, the cycles (ADR-0028). The report issue is left
// out.
type Backlog struct {
	Next    *forge.Issue
	Waiting map[int][]int // each issue waiting on an open one, and on which
	Blocked []int         // those issues, in the backlog's order
	Cycles  [][]int
}

// ReadBacklog orders the open issues and says what waits.
func ReadBacklog(open []forge.Issue, report int) Backlog {
	var list []forge.Issue
	isOpen := map[int]bool{}
	for _, is := range open {
		if is.ID != report {
			list = append(list, is)
			isOpen[is.ID] = true
		}
	}
	bl := Backlog{Waiting: map[int][]int{}}
	bl.Cycles = Order(list)
	for _, is := range list {
		if w := Waiting(is, isOpen); len(w) > 0 {
			bl.Waiting[is.ID] = w
			bl.Blocked = append(bl.Blocked, is.ID)
		}
	}
	bl.Next = NextReady(list, isOpen)
	return bl
}

// waiting is the report's part on what the order holds back: the issues
// waiting, the cycles; the first ready issues are under Next (ADR-0031).
func (p *Plan) waiting() string {
	closed := p.closedInRun()
	var open []forge.Issue
	for _, is := range p.issues {
		if closed[is.ID] {
			continue
		}
		open = append(open, p.asLeft(is))
	}
	bl := ReadBacklog(open, p.Report)
	if len(bl.Blocked) == 0 && len(bl.Cycles) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n## Waiting\n\nIssues that wait on an open issue: ordered after it, never offered first to whoever builds next.\n\n")
	for _, id := range bl.Blocked {
		fmt.Fprintf(&b, "- #%d %s waits on %s.\n", id, p.issues[id].Title, issueList(bl.Waiting[id]))
	}
	for _, c := range bl.Cycles {
		fmt.Fprintf(&b, "- **Cycle**: %s. None of them is offered first until a person takes a link off.\n", CycleText(c))
	}
	return b.String()
}

// closedInRun are the issues this run closes, as decided so far: no longer
// open.
func (p *Plan) closedInRun() map[int]bool {
	closed := map[int]bool{}
	for _, d := range p.Decisions {
		if d.Mode == Act && d.Act.Do == "close" && !d.Act.Announce {
			closed[d.Act.Issue] = true
		}
	}
	return closed
}

// joinIDs writes issue numbers as a record keeps them: "12,13".
func joinIDs(ids []int) string {
	var out []string
	for _, n := range ids {
		out = append(out, strconv.Itoa(n))
	}
	return strings.Join(out, ",")
}

// MarkedBlockers are the issues the engine's own line in a body names: the
// role's, whoever set the forge's relation.
func MarkedBlockers(body string) []int {
	for _, l := range strings.Split(body, "\n") {
		if strings.Contains(l, BlockedByMarker) {
			return BodyBlockers(l)
		}
	}
	return nil
}

// WithoutBlockers is a body with these blockers taken off the engine's
// line: the line rewritten with those left, or taken out, with the blank
// line before it, when none is. A person's own "Blocked by" line is left
// as it is.
func WithoutBlockers(body string, drop []int) string {
	lines := strings.Split(body, "\n")
	for i, l := range lines {
		if !strings.Contains(l, BlockedByMarker) {
			continue
		}
		ids := slices.DeleteFunc(BodyBlockers(l), func(n int) bool { return slices.Contains(drop, n) })
		if len(ids) > 0 {
			lines[i] = blockedByLine(ids)
			return strings.Join(lines, "\n")
		}
		before := strings.TrimRight(strings.Join(lines[:i], "\n"), "\n")
		after := strings.Join(lines[i+1:], "\n")
		switch {
		case strings.TrimSpace(after) == "":
			return before
		case before == "":
			return strings.TrimLeft(after, "\n")
		}
		return before + "\n" + after
	}
	return body
}

// OwnBlockers are the blockers of an issue the role set, still among its
// blockers: those its record keeps (a depend, a split's after) and those
// the engine's line in its body names. A person's link, or line, is never
// among them (ADR-0028).
func (h *Hand) OwnBlockers(is forge.Issue) []int {
	set := append(slices.Clone(h.Own[is.ID]), MarkedBlockers(is.Body)...)
	var out []int
	for _, b := range Blockers(is) {
		if slices.Contains(set, b) {
			out = append(out, b)
		}
	}
	return out
}

// Stale are the links the role set whose reason is gone: each open issue
// with its own blockers that are no longer open — closed, or gone from the
// forge. A closed blocker holds nothing back; the engine takes the link
// off, with no agent (ADR-0028).
func (h *Hand) Stale(open []forge.Issue) map[int][]int {
	isOpen := map[int]bool{}
	for _, is := range open {
		isOpen[is.ID] = true
	}
	out := map[int][]int{}
	for _, is := range open {
		for _, b := range h.OwnBlockers(is) {
			if !isOpen[b] {
				out[is.ID] = append(out[is.ID], b)
			}
		}
	}
	return out
}

// checkUndepend checks an undepend act (ADR-0028): the issue open, 1 to 5
// blockers, each still one of its blockers — one gone already is left out
// — and each set by the role: a person's link is never taken off. Native
// says which are in the forge's own relation, the rest in the engine's
// line.
func (p *Plan) checkUndepend(c *Proposal) (rule, why string) {
	if c.Do != "undepend" {
		return "", ""
	}
	is, ok := p.issues[c.Issue]
	if !ok || c.Issue == p.Report {
		return "no-state", fmt.Sprintf("#%d is not an open issue", c.Issue)
	}
	if len(c.BlockedBy) == 0 || len(c.BlockedBy) > maxBlockers {
		return "undepend-issue", fmt.Sprintf("a link is taken off 1 to %d blockers at a time (blocked-by)", maxBlockers)
	}
	have, own := Blockers(is), p.hand.OwnBlockers(is)
	var left []int
	for _, b := range c.BlockedBy {
		switch {
		case !slices.Contains(have, b):
		case !slices.Contains(own, b):
			return "undepend-theirs", fmt.Sprintf("#%d's link to #%d was not set by the role: a person's link is never taken off", c.Issue, b)
		case !slices.Contains(left, b):
			left = append(left, b)
		}
	}
	if len(left) == 0 {
		return "undepend-gone", fmt.Sprintf("#%d no longer waits on %s", c.Issue, issueList(c.BlockedBy))
	}
	c.BlockedBy, c.Native = left, nil
	for _, b := range left {
		if slices.Contains(is.BlockedBy, b) {
			c.Native = append(c.Native, b)
		}
	}
	return "", ""
}

// allClosed says whether every blocker an act names is closed, or closes
// in this run: the link's reason is gone, and taking it off is the
// engine's, not a person's.
func (p *Plan) allClosed(ids []int) bool {
	closing := p.closedInRun()
	for _, b := range ids {
		if p.open[b] && !closing[b] {
			return false
		}
	}
	return true
}

// recordUndepend takes the blockers an undepend took off out of the depend
// the record keeps on that issue: a link the role took off is not one a
// person did, should its blocker open again.
func (p *Plan) recordUndepend(c Proposal) {
	for i := range p.Record.Done {
		d := &p.Record.Done[i]
		if d.Issue != c.Issue || d.Act != "depend" {
			continue
		}
		var keep []string
		for _, f := range strings.Split(d.Set, ",") {
			if n, err := strconv.Atoi(f); err != nil || !slices.Contains(c.BlockedBy, n) {
				keep = append(keep, f)
			}
		}
		d.Set = strings.Join(keep, ",")
	}
	p.Record.Done = slices.DeleteFunc(p.Record.Done, func(d Done) bool { return d.Act == "depend" && d.Set == "" })
	p.dropped[c.Issue] = append(p.dropped[c.Issue], c.BlockedBy...)
	p.Changed = true
}

// asLeft is an issue as this run leaves what it waits on: the links its
// depend acts add, those its undepend acts take off.
func (p *Plan) asLeft(is forge.Issue) forge.Issue {
	is.BlockedBy = append(slices.Clone(is.BlockedBy), p.added[is.ID]...)
	if drop := p.dropped[is.ID]; len(drop) > 0 {
		is.BlockedBy = slices.DeleteFunc(is.BlockedBy, func(n int) bool { return slices.Contains(drop, n) })
		is.Body = WithoutBlockers(is.Body, drop)
	}
	return is
}

// reaches says whether from waits, through the relations given, on to.
func reaches(edges map[int][]int, from, to int) bool {
	seen := map[int]bool{}
	stack := []int{from}
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if n == to {
			return true
		}
		if seen[n] {
			continue
		}
		seen[n] = true
		stack = append(stack, edges[n]...)
	}
	return false
}
