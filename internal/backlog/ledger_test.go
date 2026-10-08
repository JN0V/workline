package backlog

import (
	"slices"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// A line of the role's own text that GitLab would run as a quick action
// is escaped; one in a fenced block, or mid-line, is left (ADR-0038).
func TestInert(t *testing.T) {
	in := "/close\nText with /label inside.\n  /label ~bug\n```\n/assign @me\n```\n/unlabel ~x"
	want := "\\/close\nText with /label inside.\n  \\/label ~bug\n```\n/assign @me\n```\n\\/unlabel ~x"
	if got := Inert(in); got != want {
		t.Errorf("Inert:\n%q\nwant\n%q", got, want)
	}
}

// Only the reporter's and the project's people's comments count; a bot's,
// a stranger's, an unnamed author's, the engine's, a bare "+1" do not.
func TestCounted(t *testing.T) {
	is := forge.Issue{Author: "zed"}
	notes := []forge.Note{
		{Body: "state\n\n<!-- workline:sticky=product-owner/state -->", Author: "bot", Bot: true},
		{Body: "Mine.", Author: "zed"},
		{Body: "Ours.", Author: "ann", Insider: true},
		{Body: "Theirs.", Author: "kim"},
		{Body: "Deployed.", Author: "ci[bot]", Insider: true, Bot: true},
		{Body: "+1", Author: "ann", Insider: true},
		{Body: " 👍 ", Author: "zed"},
		{Body: "Who?"},
	}
	var got []string
	for _, n := range Counted(notes, is) {
		got = append(got, n.Body)
	}
	if !slices.Equal(got, []string{"Mine.", "Ours."}) {
		t.Errorf("counted %q", got)
	}
}

// A state that counted every comment (before `heard`) has read those it
// held, never more than count now.
func TestHeard(t *testing.T) {
	two := 2
	for _, c := range []struct {
		st      *State
		counted int
		want    int
	}{
		{nil, 3, 3},
		{&State{Comments: 5}, 3, 3},
		{&State{Comments: 1}, 3, 1},
		{&State{Heard: &two, Comments: 9}, 3, 2},
	} {
		if got := Heard(c.st, c.counted); got != c.want {
			t.Errorf("Heard(%+v, %d) = %d, want %d", c.st, c.counted, got, c.want)
		}
	}
}

// A kind undone max times across the issues is proposed everywhere; a
// closing reopened counts as one of its kind.
func TestDemoted(t *testing.T) {
	undone := []Undo{{Issue: 1, Act: "ready"}, {Issue: 2, Act: "ready"}, {Issue: 3, Act: "rename"}}
	wrong := []Closing{{Issue: 4, Act: "ready"}, {Issue: 5, Act: "close-duplicate"}}
	if got := demoted(undone, wrong, 3); !slices.Equal(got, []string{"ready"}) {
		t.Errorf("demoted at 3: %v", got)
	}
	if got := demoted(undone, wrong, 1); !slices.Equal(got, []string{"close-duplicate", "ready", "rename"}) {
		t.Errorf("demoted at 1: %v", got)
	}
}

// The role's comment says nothing until it did or proposes something;
// then it says it above its state, folded, and what it wants from a
// person while its label is on.
func TestStateSays(t *testing.T) {
	plain := FormatState(State{Confirmed: "abc"})
	if !strings.HasPrefix(plain, "What workline knows of this issue; edited by the engine") {
		t.Errorf("plain state: %q", plain)
	}
	s := State{Confirmed: "abc", Label: scopedProposed, Did: []Did{{Act: "refine", Day: "2026-10-08", Line: "Refined #9: Need (draft)."}},
		Proposed: []Pending{{Act: "rename", Line: "Rename #9.", Proposal: &Proposal{Do: "rename", Issue: 9, Title: "A title", Why: "```x```"}}}}
	got := FormatState(s)
	for _, want := range []string{"**The product owner**, on 2026-10-08: Refined #9: Need (draft).", "**Proposes**: Rename it to \"A title\".",
		"label `workline::accepted` to agree", "take `workline::proposed` off", "<details><summary>What workline knows of this issue, edited by the engine</summary>"} {
		if !strings.Contains(got, want) {
			t.Errorf("state says %q, want it to hold %q", got, want)
		}
	}
	if st, found, err := ReadState([]string{got + "\n\n" + StateMarker("product-owner")}, "product-owner"); !found || err != nil || st.Proposed[0].Proposal.Title != "A title" {
		t.Errorf("read back: %+v %v %v", st, found, err)
	}
}
