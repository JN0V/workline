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
	Applied        []string
	Refused        []string
	Pending        int // runs judged and not applied yet
	NothingToApply bool
}

// Step is one step of a line: its verdict, and the findings it made.
type Step struct {
	Name, Status, Text string
	Findings           []verdict.Finding
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
	s := Summary{Title: title, Status: r.Status, Text: r.Summary, Findings: r.Findings, Applied: r.Applied, Refused: r.Refused}
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
		for _, f := range st.Findings {
			b.WriteString("  " + finding(f))
		}
	}
	for _, f := range s.Findings {
		b.WriteString(finding(f))
	}
	for _, r := range s.Reads {
		fmt.Fprintf(&b, "- read, %s: %s\n", r[0], r[1])
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
)

// HTML renders the summary's Markdown as a page a browser shows, for a CI
// that shows an HTML file and no Markdown one: GitLab previews a job's HTML
// artifact (with Pages), not its Markdown. Every text is escaped first.
func (s Summary) HTML() string {
	var b strings.Builder
	inline := func(t string) string {
		t = html.EscapeString(t)
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
		if x, ok := strings.CutPrefix(l, "  - "); ok {
			d, item = 2, x
		} else if x, ok := strings.CutPrefix(l, "- "); ok {
			d, item = 1, x
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
