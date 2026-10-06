package report

import (
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
