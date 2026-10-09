package report

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/line"
	"github.com/JN0V/workline/internal/verdict"
)

func TestLineSummaryPutsFindingsUnderTheirStep(t *testing.T) {
	doc := verdict.Finding{Rule: "suspect", Where: "docs/a.md", Message: "src/a.go changed\nmore\n(fix: run workline docs)"}
	gate := verdict.Finding{Rule: "secret", Where: "x.env", Message: "a token", Level: "warn"}
	route := verdict.Finding{Rule: "routing-error", Where: "documentalist", Message: "more than 3 handoffs"}
	r := &line.Result{Status: verdict.Block, Summary: "the line could not run",
		Steps: []line.Step{
			{Name: "gate:secrets", Status: verdict.Pass, Gate: &verdict.Verdict{Status: verdict.Pass, Findings: []verdict.Finding{gate}}},
			{Name: "documentalist", Status: verdict.Pass, Result: &engine.Result{Status: verdict.Pass, Summary: "1 doc suspect", Findings: []verdict.Finding{doc}}},
		},
		Findings: []verdict.Finding{gate, doc, route}}
	got := LineSummary("route merge-request", r).Markdown()
	want := "### workline route merge-request\n\n**block** — the line could not run\n\n" +
		"- **gate:secrets**: pass\n  - secret x.env (warn): a token\n" +
		"- **documentalist**: pass — 1 doc suspect\n  - suspect docs/a.md: src/a.go changed (fix: run workline docs)\n" +
		"- routing-error documentalist: more than 3 handoffs\n\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
	if strings.Count(got, "a token") != 1 {
		t.Error("a step's finding is listed twice")
	}
}

func TestHTMLSummaryEscapesAndNests(t *testing.T) {
	s := Summary{Title: "route merge-request", Status: verdict.Block,
		Steps: []Step{{Name: "documentalist", Status: verdict.Block,
			Findings: []verdict.Finding{{Rule: "suspect", Where: "docs/<b>.md", Message: "<script>alert(1)</script> `x`"}}}},
		Pending: 2}
	got := s.HTML()
	want := "<section>\n<h3>workline route merge-request</h3>\n<p><strong>block</strong></p>\n" +
		"<ul><li><strong>documentalist</strong>: block" +
		"<ul><li>suspect docs/&lt;b&gt;.md: &lt;script&gt;alert(1)&lt;/script&gt; <code>x</code></li></ul>\n" +
		"</li><li>to apply: 2 runs, by the job that holds the write token (<code>workline apply</code>)</li></ul>\n</section>\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestBacklogFindingsGroupedOneLineAnIssue(t *testing.T) {
	var fs []verdict.Finding
	for i := 1; i <= 12; i++ {
		fs = append(fs, verdict.Finding{Rule: "proposed", Level: "info", Where: fmt.Sprintf("#%d", i), Message: "Label it ready."})
	}
	fs = append(fs,
		verdict.Finding{Rule: "done", Level: "info", Where: "#3", Message: "Named the code #3 is about: a.go."},
		verdict.Finding{Rule: "done", Level: "info", Where: "#3", Message: "Refined #3: Scope. " + strings.Repeat("why ", 40)},
		verdict.Finding{Rule: "sources-unknown", Where: "#4", Message: "no file"},
		verdict.Finding{Rule: "waiting", Level: "info", Where: "#14", Message: "waits on #13"},
		verdict.Finding{Rule: "blocker-not-delivered", Level: "warn", Where: "#15", Message: "#13 closed as not planned"},
		verdict.Finding{Rule: "proposals-waiting", Level: "warn", Message: "10 issues wait on your answer"})
	s := Summary{Title: "apply", Status: verdict.Pass, Findings: fs, Issues: "https://f/issues/", Waiting: "https://f/issues?label=p"}
	got := s.Markdown()
	for _, want := range []string{
		"- **done alone** (1)\n  - [#3](https://f/issues/3): Named the code #3 is about: a.go.; Refined #3: Scope. why why",
		"- **proposed, waiting on a person** (12), [all that wait on a person](https://f/issues?label=p)\n  - [#1](https://f/issues/1): Label it ready.\n",
		"  - [#10](https://f/issues/10): Label it ready.\n  - and 2 more: [all that wait on a person](https://f/issues?label=p)\n",
		"- **read, left incomplete: yours to complete, or to leave** (1)\n  - [#4](https://f/issues/4): no file\n",
		"- [#14](https://f/issues/14): waits on #13\n- [#15](https://f/issues/15), warn: #13 closed as not planned\n- the backlog, warn: 10 issues wait on your answer\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the summary lacks %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "#11](") || strings.Count(got, "why") > 30 {
		t.Errorf("a group or a line not capped:\n%s", got)
	}
	if !strings.Contains(s.HTML(), `<ul><li><a href="https://f/issues/3">#3</a>: Named`) {
		t.Errorf("no third level nor link in the HTML:\n%s", s.HTML())
	}
	for _, l := range strings.Split(got, "\n") {
		if strings.HasPrefix(l, "  - [#3]") && len([]rune(l)) > 160 {
			t.Errorf("an issue's line not cut to its share: %d characters", len([]rune(l)))
		}
	}
	if got := short("says `"+strings.Repeat("code ", 40)+"` here", 50); strings.Count(got, "`")%2 != 0 || !strings.HasSuffix(got, "…`") || len([]rune(got)) > 50 {
		t.Errorf("a code span left open: %q", got)
	}
	long := strings.Repeat("a ", 100)
	for n, suffix := range map[int]string{1: "…", 3: "…", 4: "; and 1 more", 5: "; and 2 more"} {
		l := oneLineEach(slices.Repeat([]string{long}, n))
		if c := len([]rune(l)); c > lineMax || !strings.HasSuffix(l, suffix) || strings.Count(l, "…") != min(n, perLine) {
			t.Errorf("%d findings on one line: %d characters, %q", n, c, l)
		}
	}
}

func TestEveryBacklogGroupHeaded(t *testing.T) {
	var fs []verdict.Finding
	for _, r := range []string{"done", "done-as-accepted", "proposed", "left-to-a-person", "asks-spent", "set-aside", "next-ready", "stuck"} {
		fs = append(fs, verdict.Finding{Rule: r, Level: "info", Where: "#7", Message: r + " said"})
	}
	fs = append(fs, verdict.Finding{Rule: "done", Where: "", Message: "on no issue"})
	for _, c := range []struct {
		ahead          bool
		issues, filter string
		want           []string
	}{
		{false, "https://f/issues/", "https://f/q", []string{
			"- **done alone** (2)\n  - [#7](https://f/issues/7): done said\n  - the backlog: on no issue\n",
			"- **done, as a person accepted** (1)\n",
			"- **proposed, waiting on a person** (1), [all that wait on a person](https://f/q)\n",
			"- **left to a person, its rounds spent** (1), [all that wait on a person](https://f/q)\n  - [#7](https://f/issues/7): left-to-a-person said; asks-spent said\n",
			"- **set aside by a person** (1)\n", "- **next to build** (1)\n", "- **stuck** (1)\n"}},
		{true, "", "", []string{
			"- **to do alone, once applied** (2)\n  - #7: done said\n",
			"- **to do, as a person accepted** (1)\n",
			"- **to propose, once applied** (1)\n  - #7: proposed said\n",
			"- **left to a person, its rounds spent** (1)\n"}},
	} {
		s := Summary{Title: "t", Status: verdict.Pass, Findings: fs, Issues: c.issues, Waiting: c.filter}
		if c.ahead {
			s.Pending = 1
		}
		got := s.Markdown()
		for _, w := range c.want {
			if !strings.Contains(got, w) {
				t.Errorf("ahead %v: the summary lacks %q:\n%s", c.ahead, w, got)
			}
		}
		if strings.Contains(got, " #7 (info)") {
			t.Errorf("a grouped finding also listed alone:\n%s", got)
		}
	}
}
