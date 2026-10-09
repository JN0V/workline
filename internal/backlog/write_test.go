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

func TestIssueStatesSaysHowEachClosed(t *testing.T) {
	got := IssueStates([]int{81, 206, 300, 5}, map[int]bool{81: true}, map[int]string{206: "not_planned", 5: "completed"})
	if want := "#81 (open), #206 (closed as not planned), #5 (closed)"; got != want {
		t.Errorf("IssueStates = %q, want %q", got, want)
	}
}
