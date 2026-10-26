package report

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/gate"
	"github.com/JN0V/workline/internal/line"
)

// GatesShown is what the gates found a merge request adds, as GitHub shows
// it (ADR-0049, step 3): the annotations of the check forge.GatesCheck,
// each beside its line, and one summary comment, edited on each push.
// Both are made from a run of the merge request's own: its places and
// words are data, capped and defused, never instructions.
type GatesShown struct {
	Notes []forge.Annotation
	Left  int // past maxAnnotations: counted, not written
	New   int // the findings the change adds, every check's
	Head  string
	Base  string

	rows    []string // the table's, a check each
	entries []entry  // the findings, a rule each
	said    []string // a check that did not pass or did not work, and what to do
	worth   bool
}

// entry is the findings of one rule of a check: many make one line.
type entry struct {
	check, tool, rule string
	n                 int
	places            []string
	message           string
}

// placesShown are the places an entry names; the rest are beside their lines.
const placesShown = 3

// Gates reads what the gates of a line found the change adds: the result
// a line run with the gates alone printed (line.Gates.Only). A check that
// says how to fix what it finds, a formatter's, gets its command said
// rather than an annotation a line.
func Gates(r *line.Result, head string) *GatesShown {
	g := &GatesShown{Head: head}
	groups := map[string]*entry{}
	var order []*entry // as first found
	group := func(key string, e entry) *entry {
		if groups[key] == nil {
			groups[key] = &e
			order = append(order, groups[key])
		}
		return groups[key]
	}
	for _, s := range r.Steps {
		if s.New == nil {
			continue
		}
		name := strings.TrimPrefix(s.Name, "gate:")
		g.Base = s.New.Base
		fixes := map[string]string{}
		for _, c := range s.New.Checks {
			g.New += c.New
			fixes[c.ID] = c.Fix
			before := "–"
			if c.Before >= 0 {
				before = fmt.Sprint(c.Before)
			}
			found := fmt.Sprint(c.New)
			switch {
			case c.Error != "":
				found = "did not run"
				g.said = append(g.said, fmt.Sprintf("**%s** did not run: %s", span(c.ID), inert(c.Error, 300)))
				g.worth = true
			case c.Failed:
				found = "did not pass"
				g.said = append(g.said, fmt.Sprintf("**%s** did not pass%s", span(c.ID), fixSaid(c.Fix)))
				g.worth = true
			case c.Output == "exit":
				found = "passed"
			case c.Fix != "" && c.New > 0:
				g.said = append(g.said, fmt.Sprintf("**%s**: %d new %s in %s%s", span(c.ID), c.New, plural(c.New, "finding", "findings"),
					filesOf(s.New.Findings, c.ID), fixSaid(c.Fix)))
			}
			if c.New > 0 {
				g.worth = true
			}
			g.rows = append(g.rows, fmt.Sprintf("| %s | %s | %s | %s |", cell(name), cell(c.ID), found, before))
		}
		for _, f := range s.New.Findings {
			if fixes[f.Check] != "" {
				continue // the fix is said, not each line
			}
			e := group(f.Check+"\x00"+f.Tool+"\x00"+f.Rule, entry{check: f.Check, tool: f.Tool, rule: f.Rule, message: f.Message})
			e.n++
			at, ok := inRepo(f.Path)
			if !ok {
				continue
			}
			if len(e.places) < placesShown {
				e.places = append(e.places, fmt.Sprintf("%s:%d", at, max(f.Line, 1)))
			}
			g.note(forge.Annotation{Path: at, Line: max(f.Line, 1), Level: annotationLevel(f.Level), Title: title(f), Message: clip(f.Message)})
		}
		// A suppression the change adds is the gate's own finding, on its
		// line: one that names none is not placed.
		var own []Item
		if s.Gate != nil {
			for _, f := range s.Gate.Findings {
				if f.Rule == "suppression-added" {
					own = append(own, Item{Role: s.Name, Status: s.Status, Finding: f})
				}
			}
		}
		for _, p := range placeIn(func(string) ([]byte, error) { return nil, errNoFile }, own) {
			g.worth = true
			e := group("\x00suppression-added", entry{rule: "suppression-added", message: p.Message})
			e.n++
			if len(e.places) < placesShown {
				e.places = append(e.places, fmt.Sprintf("%s:%d", p.path, p.line))
			}
			level := "failure"
			if p.Level == "warn" {
				level = "warning"
			}
			g.note(forge.Annotation{Path: p.path, Line: p.line, Level: level, Title: "suppression-added", Message: clip(p.Message)})
		}
	}
	// The rules found most first; alike, as first found.
	sort.SliceStable(order, func(a, b int) bool { return order[a].n > order[b].n })
	for _, e := range order {
		g.entries = append(g.entries, *e)
	}
	return g
}

var errNoFile = fmt.Errorf("not read")

func (g *GatesShown) note(n forge.Annotation) {
	if len(g.Notes) == maxAnnotations {
		g.Left++
		return
	}
	g.Notes = append(g.Notes, n)
}

// Worth says whether there is anything to say: a finding the change adds,
// a check that did not pass or did not work. With none, the summary
// comment is only edited, when an earlier push left one.
func (g *GatesShown) Worth() bool { return g.worth }

// Title is the check's title: every new finding, those not written there
// too.
func (g *GatesShown) Title() string {
	return fmt.Sprintf("%d new %s", g.New, plural(g.New, "finding", "findings"))
}

// Summary is what the check says above its annotations.
func (g *GatesShown) Summary() string {
	s := fmt.Sprintf("The findings this change adds to the gates' tools since %s, each beside its line: what was there before is counted, not shown. "+
		"None is a conversation to resolve; the gate's verdict is the line's. The summary comment on the merge request counts them by rule.", span(short7(g.Base)))
	if g.Left > 0 {
		s += fmt.Sprintf(" %d more are not written here.", g.Left)
	}
	return s
}

// Comment is the summary comment's body; url, the check's page, when one
// was written: with none, the comment names no check.
func (g *GatesShown) Comment(url string, entries int) string {
	var b strings.Builder
	switch {
	case g.New == 0 && !g.worth:
		b.WriteString("### Gates: no new finding\n\n")
	default:
		fmt.Fprintf(&b, "### Gates: %d new %s\n\n", g.New, plural(g.New, "finding", "findings"))
	}
	fmt.Fprintf(&b, "On %s, only what this change adds since %s counts; what was there before is counted, not shown.\n\n", span(short7(g.Head)), span(short7(g.Base)))
	if len(g.rows) > 0 {
		b.WriteString("| Gate | Check | New | Before it |\n|---|---|---|---|\n")
		for _, r := range g.rows {
			b.WriteString(r + "\n")
		}
		b.WriteString("\n")
	}
	for _, s := range g.said {
		b.WriteString("- " + s + "\n")
	}
	shown, rest, restN := g.entries, 0, 0
	if entries > 0 && len(shown) > entries {
		for _, e := range shown[entries:] {
			rest++
			restN += e.n
		}
		shown = shown[:entries]
	}
	for _, e := range shown {
		b.WriteString("- " + e.line() + "\n")
	}
	if rest > 0 {
		fmt.Fprintf(&b, "- %d more %s, %d %s, not listed here: the reviewer's `findings-max` (%d)\n", rest, plural(rest, "rule", "rules"),
			restN, plural(restN, "finding", "findings"), entries)
	}
	if len(g.Notes) > 0 && url != "" {
		where := fmt.Sprintf("the check [%s](%s)", forge.GatesCheck, url)
		fmt.Fprintf(&b, "\nEach is beside its line in %s, on the changed files; none is a conversation to resolve.", where)
		if g.Left > 0 {
			fmt.Fprintf(&b, " %d more are not written there.", g.Left)
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// line is an entry as the comment lists it.
func (e entry) line() string {
	name := span(e.rule)
	if e.rule == "" {
		name = span(e.check)
	}
	s := "**" + name + "**"
	if e.tool != "" && e.tool != e.check {
		s += " (" + span(e.tool) + ")"
	}
	s += fmt.Sprintf(", %d new", e.n)
	if len(e.places) > 0 {
		var at []string
		for _, p := range e.places {
			at = append(at, span(p))
		}
		s += ": " + strings.Join(at, ", ")
		if e.n > len(e.places) {
			s += ", …"
		}
	}
	if m := inert(e.message, 200); m != "" {
		s += ". " + m
	}
	return s
}

// inRepo is a finding's file as an annotation takes it: from the
// repository's root, inside it.
func inRepo(p string) (string, bool) {
	if p == "" {
		return "", false
	}
	c := path.Clean(p)
	if path.IsAbs(c) || c == ".." || strings.HasPrefix(c, "../") || strings.ContainsAny(c, "\x00\n") {
		return "", false
	}
	return c, true
}

// filesOf says in how many files a check's findings are.
func filesOf(fs []gate.Finding, check string) string {
	files := map[string]bool{}
	for _, f := range fs {
		if f.Check == check {
			files[f.Path] = true
		}
	}
	return fmt.Sprintf("%d %s", len(files), plural(len(files), "file", "files"))
}

func fixSaid(fix string) string {
	if fix == "" {
		return ""
	}
	return ": run " + span(fix) + ", then push"
}

func title(f gate.Finding) string {
	t := f.Rule
	if t == "" {
		t = f.Check
	}
	if f.Tool != "" {
		t += " (" + f.Tool + ")"
	}
	if len(t) > maxTitleLen {
		t = strings.ToValidUTF8(t[:maxTitleLen-len("…")], "") + "…"
	}
	return t
}

// maxTitleLen is the most of an annotation's title GitHub takes.
const maxTitleLen = 255

func annotationLevel(sarif string) string {
	switch sarif {
	case "error":
		return "failure"
	case "note", "none":
		return "notice"
	}
	return "warning"
}

// cell is text in a code span in a table's cell.
func cell(s string) string { return strings.ReplaceAll(span(s), "|", `\|`) }

// span is text in a code span, its backticks taken out.
func span(s string) string {
	return "`" + strings.ReplaceAll(oneLine(s), "`", "'") + "`"
}

// inert is a run's text made one line, cut, its mentions and HTML defused.
func inert(s string, n int) string {
	s = oneLine(s)
	if len(s) > n {
		s = strings.ToValidUTF8(s[:n], "") + "…"
	}
	s = strings.ReplaceAll(s, "@", "@​")
	return strings.NewReplacer("<", "&lt;", ">", "&gt;").Replace(s)
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func short7(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
