package backlog

import (
	"fmt"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/work"
)

// A changed need flags the open issues it touches (ADR-0032), as a
// requirements tool marks a link suspect when the item upstream changes:
// a person's change to an issue's Need or Scope, or to the lines of a file
// an issue was imported from, has the issues built on it read again or
// listed for a person; every act on them is proposed, never done alone.

// Watched are the sections whose change touches the issues built on an
// issue: what it needs, and the part of the code it holds.
var Watched = []string{"Need", "Scope"}

// Basis is what an issue's state keeps of its watched sections, to tell a
// person's change from the engine's: their text, the engine's own lines
// (a draft line, a blocked-by line) left out.
func Basis(body string) map[string]string {
	have := work.Sections(body)
	out := map[string]string{}
	for _, name := range Watched {
		var kept []string
		for _, l := range strings.Split(have[name], "\n") {
			if !strings.Contains(l, "<!-- workline:") {
				kept = append(kept, l)
			}
		}
		if t := strings.TrimSpace(strings.Join(kept, "\n")); t != "" {
			out[name] = t
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// Keep records in the state the body the engine read or left: its digest,
// and its watched sections.
func (s *State) Keep(body string) {
	s.Body = BodyDigest(body)
	s.Sections = Basis(body)
}

// Rewritten are the watched sections a person rewrote since the state was
// kept, spaces aside: a section written where there was none is not a
// change to what others were built on, nor is a state that kept none yet.
func Rewritten(st *State, body string) []string {
	if st == nil || st.Sections == nil {
		return nil
	}
	now := Basis(body)
	var out []string
	for _, name := range Watched {
		if old := st.Sections[name]; old != "" && squeeze(old) != squeeze(now[name]) {
			out = append(out, name)
		}
	}
	return out
}

// How an issue is touched by a change.
const (
	TouchPart    = "part"    // a part of the changed issue: read again
	TouchWaits   = "waits"   // waits on it: listed for a person
	TouchSources = "sources" // names the same code: listed for a person
	TouchImport  = "import"  // opened from the lines that changed: read again
)

// Change is a change to what open issues were built on, as the record
// keeps it until a person ticks it seen or its issues are closed.
type Change struct {
	Issue int      `yaml:"issue"`               // the issue whose sections changed, or the one opened from the lines
	What  []string `yaml:"what,flow,omitempty"` // the sections a person rewrote
	Path  string   `yaml:"path,omitempty"`      // an imported file whose lines changed
	Lines string   `yaml:"lines,omitempty"`     // those lines then, and now: "7-8 → 7-9"
	Since string   `yaml:"since,omitempty"`     // the day it was found, YYYY-MM-DD
	Touch []Touch  `yaml:"touched"`             // the open issues it touches
	Was   string   `yaml:"was,omitempty"`       // the text before, for the agent (not kept in the record)
	Now   string   `yaml:"now,omitempty"`       // the text after, for the agent (not kept in the record)
}

// Touch is an open issue a change touches, and how.
type Touch struct {
	Issue int    `yaml:"issue"`
	How   string `yaml:"how"`
	Files string `yaml:"files,omitempty"` // the code shared, for TouchSources
	Read  bool   `yaml:"read,omitempty"`  // read again by the run that found it, with the change
}

// Key is the change's box in the report.
func (c Change) Key() string { return fmt.Sprintf("changed/%d", c.Issue) }

// said names the change in a line.
func (c Change) said() string {
	if c.Path != "" {
		return fmt.Sprintf("the lines of %s it was opened from, changed", c.Path)
	}
	return fmt.Sprintf("#%d's %s, changed by a person", c.Issue, strings.Join(c.What, " and "))
}

// Touched lists the open issues a change to issue id's sections touches,
// none twice, a part first: its open parts (read again), the open issues
// that wait on it, and — its Scope changed — those whose sources share a
// file with its own: a Need rewritten moves what is built on it, not
// every issue on the same code.
func Touched(id int, open []forge.Issue, sources map[int][]string, scope bool) []Touch {
	var out []Touch
	seen := map[int]bool{id: true}
	byID := map[int]forge.Issue{}
	for _, is := range open {
		byID[is.ID] = is
	}
	for _, n := range Parts(byID[id]) {
		if _, ok := byID[n]; ok && !seen[n] {
			seen[n] = true
			out = append(out, Touch{Issue: n, How: TouchPart})
		}
	}
	for _, is := range open {
		if !seen[is.ID] && slices.Contains(Blockers(is), id) {
			seen[is.ID] = true
			out = append(out, Touch{Issue: is.ID, How: TouchWaits})
		}
	}
	mine := files(sources[id])
	if !scope {
		mine = nil
	}
	for _, is := range open {
		if seen[is.ID] || len(mine) == 0 {
			continue
		}
		var shared []string
		for _, f := range files(sources[is.ID]) {
			if slices.Contains(mine, f) {
				shared = append(shared, f)
			}
		}
		if len(shared) > 0 {
			seen[is.ID] = true
			out = append(out, Touch{Issue: is.ID, How: TouchSources, Files: strings.Join(shared, ", ")})
		}
	}
	return out
}

// files are the paths sources name, a symbol aside.
func files(sources []string) []string {
	var out []string
	for _, s := range sources {
		p, _, _ := strings.Cut(s, "#")
		if p != "" && !slices.Contains(out, p) {
			out = append(out, p)
		}
	}
	return out
}

// importedFrom reads where an issue was imported from, as the import wrote
// it: its file and the lines it was opened from.
var importedFrom = regexp.MustCompile("Opened from `([^`]+)`, lines (\\d+) to (\\d+) by the ")

// Imported is the file and lines an issue was opened from by an import,
// ok false when its body no longer says so or holds no import key.
func Imported(body string) (path string, from, to int, ok bool) {
	if !strings.Contains(body, "<!-- workline:import=") {
		return "", 0, 0, false
	}
	m := importedFrom.FindStringSubmatch(body)
	if m == nil || !strings.Contains(body, "<!-- workline:import="+m[1]+":") {
		return "", 0, 0, false
	}
	from, _ = strconv.Atoi(m[2])
	to, _ = strconv.Atoi(m[3])
	return m[1], from, to, from > 0 && to >= from
}

// hunk is one change of a diff with no context: old lines [Old, Old+OldN),
// new lines [New, New+NewN); with OldN 0, lines added after line Old.
type hunk struct{ Old, OldN, New, NewN int }

var hunkHead = regexp.MustCompile(`(?m)^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

func hunks(diff string) []hunk {
	var out []hunk
	for _, m := range hunkHead.FindAllStringSubmatch(diff, -1) {
		n := func(s string, def int) int {
			if s == "" {
				return def
			}
			v, _ := strconv.Atoi(s)
			return v
		}
		out = append(out, hunk{n(m[1], 0), n(m[2], 1), n(m[3], 0), n(m[4], 1)})
	}
	return out
}

// follow carries lines [from, to] of a file through a diff's hunks: where
// they are after it, and whether a hunk changed them — lines among them
// changed, removed, or added between two of them.
func follow(hs []hunk, from, to int) (nfrom, nto int, changed bool) {
	nfrom, nto = from, to
	shift := func(n int, last bool) int {
		d := 0
		for _, h := range hs {
			switch {
			case h.OldN == 0 && h.Old < n, h.OldN > 0 && h.Old+h.OldN-1 < n:
				d += h.NewN - h.OldN
			case h.OldN > 0 && h.Old <= n && n <= h.Old+h.OldN-1:
				// Inside a hunk: its first new line, or its last.
				if last {
					return h.New + h.NewN - 1 - n
				}
				if h.NewN == 0 {
					return h.New + 1 - n
				}
				return h.New - n
			}
		}
		return d
	}
	for _, h := range hs {
		if h.OldN == 0 && from <= h.Old && h.Old < to || h.OldN > 0 && h.Old <= to && from <= h.Old+h.OldN-1 {
			changed = true
		}
	}
	return from + shift(from, false), to + shift(to, true), changed
}

// LinesChange says whether a commit between base and head changed lines
// [from, to] of path as they were at opened — the commit the issue was
// opened at, its lines then — with the text at base and at head, and where
// the lines are now; an empty now when they are gone.
func LinesChange(repo, path string, from, to int, opened, base, head string) (was, now string, nfrom, nto int, changed bool) {
	diff := func(a, b string) []hunk {
		out, err := exec.Command("git", "-C", repo, "diff", "--no-ext-diff", "-U0", a, b, "--", path).Output()
		if err != nil {
			return nil
		}
		return hunks(string(out))
	}
	show := func(rev string, a, b int) string {
		out, err := exec.Command("git", "-C", repo, "show", rev+":"+path).Output()
		if err != nil || b < a {
			return ""
		}
		lines := strings.Split(string(out), "\n")
		if a < 1 || a > len(lines) {
			return ""
		}
		return strings.Join(lines[a-1:min(b, len(lines))], "\n")
	}
	if base != opened {
		from, to, _ = follow(diff(opened, base), from, to)
	}
	nfrom, nto, changed = follow(diff(base, head), from, to)
	if !changed {
		return "", "", nfrom, nto, false
	}
	return show(base, from, to), show(head, nfrom, nto), nfrom, nto, true
}

// readChanges settles the changes the record holds — ticked seen by a
// person of the project, or none of their issues open any more — then
// adds those this run found, a change found again kept with its day; an
// issue read again for one has every act on it proposed (ADR-0032).
func (p *Plan) readChanges(found []Change) {
	p.changedFor = map[int]Change{}
	var kept []Change
	for _, c := range p.Record.Changes {
		if t, ok := p.hand.Tick(c.Key()); ok && t.Person() {
			p.Changed = true
			p.said = append(p.said, fmt.Sprintf("- The change to #%d checked by %s: it leaves the report.", c.Issue, t.Who()))
			continue
		}
		if !p.open[c.Issue] && !slices.ContainsFunc(c.Touch, func(t Touch) bool { return p.open[t.Issue] }) {
			p.Changed = true
			continue
		}
		kept = append(kept, c)
	}
	today := time.Now().UTC().Format(dateLayout)
	for _, c := range found {
		c.Was, c.Now = "", ""
		for _, t := range c.Touch {
			if t.Read {
				p.changedFor[t.Issue] = c
			}
		}
		i := slices.IndexFunc(kept, func(k Change) bool { return k.Key() == c.Key() })
		if i < 0 {
			c.Since = today
			kept = append(kept, c)
			p.Changed = true
			continue
		}
		c.Since = kept[i].Since
		if !sameChange(kept[i], c) {
			kept[i], p.Changed = c, true
		}
	}
	p.Record.Changes = kept
}

// sameChange says whether two changes say the same in the report.
func sameChange(a, b Change) bool {
	return a.Issue == b.Issue && slices.Equal(a.What, b.What) && a.Path == b.Path && a.Lines == b.Lines &&
		a.Since == b.Since && slices.Equal(a.Touch, b.Touch)
}

// changes is the report's part on the changes it holds: each issue it
// touches, how, whether it was read again, and what is proposed for it.
func (p *Plan) changes() string {
	if len(p.Record.Changes) == 0 {
		return ""
	}
	proposed := map[int][]string{}
	for _, q := range p.Record.Proposed {
		if q.Issue > 0 && q.Key == "" && !slices.Contains(proposed[q.Issue], q.Act) {
			proposed[q.Issue] = append(proposed[q.Issue], q.Act)
		}
	}
	var b strings.Builder
	b.WriteString("\n## Changed needs\n\nWhat these issues were built on changed: the role read again those it could, and only proposes — it moves none of them alone. Check each against the change, then tick its box.\n\n")
	for _, c := range p.Record.Changes {
		title := ""
		if is, ok := p.issues[c.Issue]; ok {
			title = " " + is.Title
		}
		switch {
		case c.Path != "":
			fmt.Fprintf(&b, "- [ ] `%s`, lines %s, which #%d%s was opened from, changed (%s). %s\n", c.Path, c.Lines, c.Issue, title, c.Since, TickMarker(c.Key()))
		default:
			fmt.Fprintf(&b, "- [ ] #%d%s: a person changed its %s (%s). %s\n", c.Issue, title, strings.Join(c.What, " and "), c.Since, TickMarker(c.Key()))
		}
		for _, t := range c.Touch {
			if !p.open[t.Issue] {
				continue
			}
			how := map[string]string{TouchPart: "a part of it", TouchWaits: "waits on it", TouchSources: "names the same code, " + t.Files, TouchImport: "opened from those lines"}[t.How]
			what := "not read: check it against the change"
			if t.Read {
				what = "read again with the change: nothing proposed"
				if kinds := proposed[t.Issue]; len(kinds) > 0 {
					what = "read again with the change: proposed below — " + strings.Join(kinds, ", ")
				}
			}
			fmt.Fprintf(&b, "  - #%d %s, %s: %s.\n", t.Issue, p.issues[t.Issue].Title, how, what)
		}
	}
	return b.String()
}
