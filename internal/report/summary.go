package report

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/line"
	"github.com/JN0V/workline/internal/sample"
	"github.com/JN0V/workline/internal/verdict"
)

// Summary is what a CI job's page shows a person, in Markdown: what ran,
// each step's verdict and findings, what the agent was asked, and what waits
// to be applied, with no log nor JSON to read. Every finding is there, those
// that name no file too.
type Summary struct {
	Title          string // the command: "route merge-request", "apply"
	Status, Text   string
	Steps          []Step // a line's steps
	Findings       []verdict.Finding
	Calls          []stepCall
	Notes          [][2]string // the step, the agent's note
	Reads          [][2]string // the sample: each doc read, and its verdict
	Map            []string    // an import's map: each item to its issue or its reason
	Applied        []string
	Refused        []string
	Pending        int // runs judged and not applied yet
	NothingToApply bool
	// Issues and Waiting link the issues a backlog's findings name, and the
	// saved filter of what waits on a person (engine.Result's).
	Issues, Waiting string
}

// Step is one step of a line: its verdict, and the findings it made.
type Step struct {
	Name, Status, Text string
	Findings           []verdict.Finding
	ToApply            bool // judged, not applied: its acts are what will be done
}

type stepCall struct {
	step string
	agent.Call
}

// LineSummary is the summary of a line's run: each step with its findings,
// then the line's own (a routing that could not run).
func LineSummary(title string, r *line.Result) Summary {
	s := Summary{Title: title, Status: r.Status, Text: r.Summary, Pending: len(r.Pending)}
	n := 0
	for _, st := range r.Steps {
		step := Step{Name: st.Name, Status: st.Status}
		if st.Gate != nil {
			step.Text, step.Findings = st.Gate.Summary, st.Gate.Findings
		}
		if st.Result != nil {
			step.Text, step.Findings = st.Result.Summary, st.Result.Findings
			step.ToApply = st.Result.ToApply || len(st.Result.Pending) > 0
			if st.Result.Issues != "" {
				s.Issues, s.Waiting = st.Result.Issues, st.Result.Waiting
			}
			for _, c := range st.Result.Calls {
				s.Calls = append(s.Calls, stepCall{st.Name, c})
			}
			for _, note := range st.Result.Notes {
				s.Notes = append(s.Notes, [2]string{st.Name, note})
			}
		}
		n += len(step.Findings)
		s.Steps = append(s.Steps, step)
	}
	if n <= len(r.Findings) {
		s.Findings = r.Findings[n:]
	}
	return s
}

// RoleSummary is the summary of one role's run, or of runs applied.
func RoleSummary(title, role string, r *engine.Result) Summary {
	s := Summary{Title: title, Status: r.Status, Text: r.Summary, Findings: r.Findings, Applied: r.Applied, Refused: r.Refused,
		Issues: r.Issues, Waiting: r.Waiting}
	for _, c := range r.Calls {
		s.Calls = append(s.Calls, stepCall{role, c})
	}
	for _, n := range r.Notes {
		s.Notes = append(s.Notes, [2]string{role, n})
	}
	switch {
	case len(r.Pending) > 0:
		s.Pending = len(r.Pending)
	case r.ToApply:
		s.Pending = 1
	}
	return s
}

// SampleSummary is the summary of the weekly sample's read or write.
func SampleSummary(title string, r *sample.Result) Summary {
	s := Summary{Title: title, Status: r.Status, Text: r.Summary, Findings: r.Findings, Applied: r.Applied, Refused: r.Refused}
	for _, c := range r.Calls {
		s.Calls = append(s.Calls, stepCall{"judge", c})
	}
	for _, rd := range r.Reads {
		s.Reads = append(s.Reads, [2]string{rd.Doc, rd.Verdict})
	}
	return s
}

// Markdown renders the summary as GitHub's job summary and GitLab's
// Markdown both show it.
func (s Summary) Markdown() string {
	var b strings.Builder
	title := "### workline " + s.Title
	var agents []string
	for _, c := range s.Calls {
		if c.Agent != "" && !contains(agents, c.Agent) {
			agents = append(agents, c.Agent)
		}
	}
	if len(agents) > 0 {
		title += " (" + strings.Join(agents, ", ") + ")"
	}
	b.WriteString(title + "\n\n**" + s.Status + "**")
	if s.Text != "" {
		b.WriteString(" — " + oneLine(s.Text))
	}
	b.WriteString("\n\n")
	for _, st := range s.Steps {
		fmt.Fprintf(&b, "- **%s**: %s", st.Name, st.Status)
		if st.Text != "" {
			b.WriteString(" — " + oneLine(st.Text))
		}
		b.WriteString("\n")
		s.findings(&b, "  ", st.Findings, st.ToApply)
	}
	s.findings(&b, "", s.Findings, s.Pending > 0)
	for _, r := range s.Reads {
		fmt.Fprintf(&b, "- read, %s: %s\n", r[0], r[1])
	}
	for _, m := range s.Map {
		fmt.Fprintf(&b, "- map, %s\n", oneLine(m))
	}
	for _, c := range s.Calls {
		model := c.Model
		if model == "" {
			model = c.Asked
		}
		if model == "" {
			model = "?"
		}
		fmt.Fprintf(&b, "- agent, %s: %s %d tokens in, %d out\n", c.step, model, c.TokensIn, c.TokensOut)
	}
	for _, n := range s.Notes {
		fmt.Fprintf(&b, "- note from the agent, %s: %s\n", n[0], firstLine(n[1]))
	}
	if len(s.Applied) > 0 {
		fmt.Fprintf(&b, "- applied: %s\n", strings.Join(s.Applied, ", "))
	}
	if len(s.Refused) > 0 {
		fmt.Fprintf(&b, "- refused: %s\n", strings.Join(s.Refused, ", "))
	}
	switch {
	case s.Pending == 1:
		b.WriteString("- to apply: 1 run, by the job that holds the write token (`workline apply`)\n")
	case s.Pending > 1:
		fmt.Fprintf(&b, "- to apply: %d runs, by the job that holds the write token (`workline apply`)\n", s.Pending)
	case s.NothingToApply:
		b.WriteString("- nothing to apply: the line proposed nothing\n")
	}
	b.WriteString("\n")
	return b.String()
}

// A group of a backlog's findings, one line an issue under its heading
// (ADR-0038): what was done or will be, what waits on a person.
type group struct {
	rules       []string
	done, ahead string // the heading once applied, and when judged only
	waiting     bool   // linked to the saved filter of what waits on a person
}

var groups = []group{
	{[]string{"done"}, "done alone", "to do alone, once applied", false},
	{[]string{"done-as-accepted"}, "done, as a person accepted", "to do, as a person accepted", false},
	{[]string{"proposed"}, "proposed, waiting on a person", "to propose, once applied", true},
	{[]string{"left-to-a-person", "asks-spent"}, "left to a person, its rounds spent", "left to a person, its rounds spent", true},
	{[]string{"set-aside"}, "set aside by a person", "set aside by a person", false},
	{[]string{"next-ready"}, "next to build", "next to build", false},
	{[]string{"stuck"}, "stuck", "stuck", false},
}

// groupMax is how many issues a group lists; "and N more" after.
const groupMax = 10

// findings writes a list of findings: a backlog's grouped first, one line
// an issue, linked, the rest a bullet each, in their order.
func (s Summary) findings(b *strings.Builder, indent string, fs []verdict.Finding, ahead bool) {
	grouped := map[int]bool{}
	for _, g := range groups {
		var order []string
		said := map[string][]string{}
		for i, f := range fs {
			if !contains(g.rules, f.Rule) {
				continue
			}
			grouped[i] = true
			if _, ok := said[f.Where]; !ok {
				order = append(order, f.Where)
			}
			said[f.Where] = append(said[f.Where], message(f))
		}
		if len(order) == 0 {
			continue
		}
		head := g.done
		if ahead {
			head = g.ahead
		}
		filter := ""
		if g.waiting && s.Waiting != "" {
			filter = "[all that wait on a person](" + s.Waiting + ")"
		}
		fmt.Fprintf(b, "%s- **%s** (%d)", indent, head, len(order))
		if filter != "" {
			b.WriteString(", " + filter)
		}
		b.WriteString("\n")
		for i, where := range order {
			if i == groupMax {
				more := fmt.Sprintf("and %d more", len(order)-groupMax)
				if filter != "" {
					more += ": " + filter
				}
				fmt.Fprintf(b, "%s  - %s\n", indent, more)
				break
			}
			fmt.Fprintf(b, "%s  - %s: %s\n", indent, s.issue(where), oneLineEach(said[where]))
		}
	}
	for i, f := range fs {
		if !grouped[i] {
			b.WriteString(indent + finding(f))
		}
	}
}

var issueRef = regexp.MustCompile(`^#(\d+)$`)

// issue is where a finding is, linked to its page when it is an issue.
func (s Summary) issue(where string) string {
	if m := issueRef.FindStringSubmatch(where); m != nil && s.Issues != "" {
		return "[" + where + "](" + s.Issues + m[1] + ")"
	}
	if where == "" {
		return "the backlog"
	}
	return where
}

// oneLineEach is an issue's findings in a group on one line of about
// lineMax characters — a cut's "…" and a code span closed aside: the first
// perLine, each cut to its share, then how many more.
func oneLineEach(said []string) string {
	shown := said[:min(len(said), perLine)]
	budget := lineMax - 2*(len(shown)-1) // "; " between
	more := ""
	if n := len(said) - len(shown); n > 0 {
		more = fmt.Sprintf("and %d more", n)
		budget -= len(more) + 2
	}
	var out []string
	for _, t := range shown {
		out = append(out, short(t, budget/len(shown)))
	}
	if more != "" {
		out = append(out, more)
	}
	return strings.Join(out, "; ")
}

const (
	lineMax = 120
	perLine = 3 // findings on an issue's line, each about 35 characters at least
)

// short is a text on one line, cut at a word past n characters; a code
// span the cut leaves open is closed.
func short(t string, n int) string {
	t = oneLine(t)
	r := []rune(t)
	if len(r) <= n {
		return t
	}
	cut := string(r[:n])
	if i := strings.LastIndex(cut, " "); i > len(cut)/2 {
		cut = cut[:i]
	}
	cut = strings.TrimRight(cut, " ,;:.") + "…"
	if strings.Count(cut, "`")%2 == 1 {
		cut += "`"
	}
	return cut
}

// finding is a finding's bullet: its rule, where, level and message.
func finding(f verdict.Finding) string {
	where, level := "", ""
	if f.Where != "" {
		where = " " + f.Where
	}
	if f.Level != "" {
		level = " (" + f.Level + ")"
	}
	return fmt.Sprintf("- %s%s%s: %s\n", f.Rule, where, level, message(f))
}

// message is a finding's first line, and its last when that one is a
// parenthesis, which says how to act on it.
func message(f verdict.Finding) string {
	if f.Message == "" {
		return f.Rule
	}
	l := strings.Split(strings.TrimRight(f.Message, "\n"), "\n")
	if len(l) > 1 && strings.HasPrefix(l[len(l)-1], "(") {
		return l[0] + " " + l[len(l)-1]
	}
	return l[0]
}

func firstLine(s string) string { l, _, _ := strings.Cut(s, "\n"); return l }

func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }

func contains(l []string, s string) bool {
	for _, x := range l {
		if x == s {
			return true
		}
	}
	return false
}

var (
	bold = regexp.MustCompile(`\*\*(.+?)\*\*`)
	code = regexp.MustCompile("`([^`]+)`")
	link = regexp.MustCompile(`\[([^\]]+)\]\((https?://[^)\s]+)\)`)
)

// HTML renders the summary's Markdown as a page a browser shows, for a CI
// that shows an HTML file and no Markdown one: GitLab previews a job's HTML
// artifact (with Pages), not its Markdown. Every text is escaped first.
func (s Summary) HTML() string {
	var b strings.Builder
	inline := func(t string) string {
		t = link.ReplaceAllString(html.EscapeString(t), `<a href="$2">$1</a>`)
		return code.ReplaceAllString(bold.ReplaceAllString(t, "<strong>$1</strong>"), "<code>$1</code>")
	}
	depth := 0
	closeTo := func(d int) {
		for ; depth > d; depth-- {
			b.WriteString("</li></ul>\n")
		}
	}
	b.WriteString("<section>\n")
	for _, l := range strings.Split(strings.TrimRight(s.Markdown(), "\n"), "\n") {
		d, item := 0, ""
		trimmed := strings.TrimLeft(l, " ")
		if x, ok := strings.CutPrefix(trimmed, "- "); ok {
			d, item = (len(l)-len(trimmed))/2+1, x
		}
		switch {
		case d > depth:
			for ; depth < d; depth++ {
				b.WriteString("<ul><li>")
			}
			b.WriteString(inline(item))
		case d > 0:
			closeTo(d)
			b.WriteString("</li><li>" + inline(item))
		case l == "":
		default:
			closeTo(0)
			if h, ok := strings.CutPrefix(l, "### "); ok {
				b.WriteString("<h3>" + inline(h) + "</h3>\n")
			} else {
				b.WriteString("<p>" + inline(l) + "</p>\n")
			}
		}
	}
	closeTo(0)
	b.WriteString("</section>\n")
	return b.String()
}

const htmlHead = `<!doctype html>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>workline summary</title>
<style>body{font-family:system-ui,sans-serif;max-width:60rem;margin:2rem auto;padding:0 1rem;line-height:1.5}code{font-size:.9em}</style>
`

// AppendSummary adds the summary to each file, as GitHub's
// $GITHUB_STEP_SUMMARY is added to: a job that judges, then one that
// applies, write one page. A file named .html gets HTML, any other
// Markdown.
func AppendSummary(files []string, s Summary) error {
	for _, file := range files {
		text := s.Markdown()
		if strings.EqualFold(filepath.Ext(file), ".html") {
			text = s.HTML()
			if st, err := os.Stat(file); err != nil || st.Size() == 0 {
				text = htmlHead + text
			}
		}
		f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		if _, err := f.WriteString(text); err != nil {
			f.Close()
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
	}
	return nil
}
