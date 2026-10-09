package backlog

import (
	"slices"
	"testing"
)

func TestLinkDecisionsLinksABareOneOnly(t *testing.T) {
	decisions := map[int]string{38: "docs/adr/0038-proposals.md"}
	got := LinkDecisions("As ADR-0038 says; [ADR-0038](x) and `ADR-0038` stay; ADR-0099 is not held.", "https://f/blob/HEAD/", decisions)
	want := "As [ADR-0038](https://f/blob/HEAD/docs/adr/0038-proposals.md) says; [ADR-0038](x) and `ADR-0038` stay; ADR-0099 is not held."
	if got != want {
		t.Errorf("LinkDecisions =\n%s\nwant\n%s", got, want)
	}
	fenced := "```\nADR-0038\n```\nthen ADR-0038, `a` [ADR-0038](x) `b`"
	if got := LinkDecisions(fenced, "p/", decisions); got != "```\nADR-0038\n```\nthen [ADR-0038](p/docs/adr/0038-proposals.md), `a` [ADR-0038](x) `b`" {
		t.Errorf("a fenced block: %q", got)
	}
	if got := LinkDecisions("ADR-0038", "", decisions); got != "ADR-0038" {
		t.Errorf("no forge pages: %q", got)
	}
}

func TestMarkClosedSaysItOnce(t *testing.T) {
	pages := "https://f/issues/"
	text := "Waits on #206 and #81; see [#206](https://f/issues/206), [the tester](https://f/issues/206#c1), #206 (closed) already, `#206` in code, other/repo#206, &#206;."
	got := MarkClosed(text, pages, map[int]bool{206: true})
	want := "Waits on #206 (closed) and #81; see [#206](https://f/issues/206) (closed), [the tester](https://f/issues/206#c1) (closed), #206 (closed) already, `#206` in code, other/repo#206, &#206;."
	if got != want {
		t.Errorf("MarkClosed =\n%s\nwant\n%s", got, want)
	}
	if ids := CitedIssues(text, pages); !slices.Equal(ids, []int{81, 206}) {
		t.Errorf("CitedIssues = %v", ids)
	}
	if ids := CitedIssues("see https://other/issues/7 and #3", ""); !slices.Equal(ids, []int{3}) {
		t.Errorf("CitedIssues, no pages = %v", ids)
	}
}

func TestMarkClosedSkipsCodeOnly(t *testing.T) {
	// A stray backtick, a fenced block: what follows them is still marked.
	text := "A lone ` here, then #206.\n\n```\nsee #206\n```\n\nAfter the block, #206 and `#206`."
	got := MarkClosed(text, "", map[int]bool{206: true})
	want := "A lone ` here, then #206 (closed).\n\n```\nsee #206\n```\n\nAfter the block, #206 (closed) and `#206`."
	if got != want {
		t.Errorf("MarkClosed =\n%s\nwant\n%s", got, want)
	}
	// A one-line ```span``` is no fence; a fence line with an info string
	// does not close a block; a cite right after a code span is read.
	edge := "```#206``` done; then #206.\n\n```go\nx #206\n```go\ny #206\n```\n\n`x`#206"
	wantEdge := "```#206``` done; then #206 (closed).\n\n```go\nx #206\n```go\ny #206\n```\n\n`x`#206 (closed)"
	if got := MarkClosed(edge, "", map[int]bool{206: true}); got != wantEdge {
		t.Errorf("MarkClosed, edges =\n%s\nwant\n%s", got, wantEdge)
	}
	// No page known: a bare cite is still read, and marked.
	if got := MarkClosed("[#5](x) and #5", "", map[int]bool{5: true}); got != "[#5](x) and #5 (closed)" {
		t.Errorf("no pages: %q", got)
	}
}

func TestIssueStatesSaysHowEachClosed(t *testing.T) {
	got := IssueStates([]int{81, 206, 300, 5}, map[int]bool{81: true}, map[int]string{206: "not_planned", 5: "completed"})
	if want := "#81 (open), #206 (closed as not planned), #5 (closed)"; got != want {
		t.Errorf("IssueStates = %q, want %q", got, want)
	}
}
