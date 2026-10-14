package sample

import (
	"fmt"
	"strings"
	"testing"
)

// One in ten per role and kind, at least one when a kind has any: a rare
// kind is drawn beside a common one, and the same on a rerun.
func TestDrawEach(t *testing.T) {
	var acts []Act
	for i := range 23 {
		acts = append(acts, Act{Role: reviewRole, Kind: "dropped", Ref: fmt.Sprintf("#%d", i), key: fmt.Sprintf("d%d", i)})
	}
	acts = append(acts, Act{Role: reviewRole, Kind: "kept", Ref: "#1", key: "k1"}, Act{Role: productOwner, Kind: "ready", Ref: "#9", key: "r9"})
	drawn, counts := drawEach("2026-W41", acts)
	want := "[{product-owner ready 1 1} {reviewer dropped 23 3} {reviewer kept 1 1}]"
	if fmt.Sprint(counts) != want {
		t.Errorf("counts: %v, want %s", counts, want)
	}
	if len(drawn) != 5 || drawn[0].Role != productOwner || drawn[0].Verdict != "not-read" {
		t.Errorf("drawn: %+v", drawn)
	}
	again, _ := drawEach("2026-W41", append([]Act(nil), acts...))
	if fmt.Sprint(drawn) != fmt.Sprint(again) {
		t.Errorf("a rerun drew %v, then %v", drawn, again)
	}
}

// The reviewer's summary comment, as it reads: a finding kept and one
// dropped are its judge's acts; a nit, one left unverified and the rules'
// rows are not. A `|` in a finding is escaped in its cell; a block says
// the head it was found at.
func TestSummaryActs(t *testing.T) {
	comment := "**workline reviewer** — 2 findings on the change.\n\n" +
		"| | | Where | Finding |\n|---|---|---|---|\n" +
		"| **blocks** | correctness | `a.go:3` | Important: a \\| b is never true |\n" +
		"| warns | tests | `a_test.go:9` | Important: no test reads it — why (not verified: the judge was not asked) |\n" +
		"| warns | edge-cases | `a.go:1` | Nit: a name |\n" +
		"| warns | finding-judged-no |  | dropped, from the security lens: \"x at y\" at a.go:7 — the judge: the code shown handles it (independence: model (claude-sonnet-5 → claude-opus-5)) |\n" +
		"| warns | subject-length |  | the subject is long |\n\n" +
		"<!-- workline:record reviewed=abc runs=1 -->\n" +
		"<!-- workline:blocks [{\"lens\":\"correctness\",\"at\":\"f00d\",\"where\":\"a.go:3\",\"title\":\"x\",\"cause\":{\"path\":\"a.go\",\"quote\":\"a \\u002d b\"}}] -->\n"
	var got []string
	for _, a := range summaryActs("#7", comment) {
		got = append(got, fmt.Sprintf("%s %s %s %s %q", a.Kind, a.Where, a.Commit, a.By, a.Line))
	}
	want := []string{
		`kept a.go:3 f00d  "the correctness lens: Important: a | b is never true"`,
		`dropped a.go:7  claude-sonnet-5 → claude-opus-5 "dropped, from the security lens: \"x at y\" at a.go:7 — the judge: the code shown handles it (independence: model (claude-sonnet-5 → claude-opus-5))"`,
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("acts:\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}
