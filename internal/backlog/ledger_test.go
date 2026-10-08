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
	for _, c := range []struct{ in, want string }{
		// Inline code at a line's start opens no block.
		{"```x```\n/close", "```x```\n\\/close"},
		// A fence indented four spaces, or by a tab, is no fence.
		{"    ```\n/close", "    ```\n\\/close"},
		{"\t```\n/close", "\t```\n\\/close"},
		// Indented three spaces, it is one.
		{"   ```\n/close\n```", "   ```\n/close\n```"},
		// A fence with an info string opens a block; one whose info
		// string holds a backtick does not.
		{"```go\n/close\n```\n/close", "```go\n/close\n```\n\\/close"},
		{"```go`\n/close", "```go`\n\\/close"},
		// A shorter fence, or an indented one, inside a block closes
		// nothing.
		{"````\n```\n/close\n````\n/close", "````\n```\n/close\n````\n\\/close"},
		{"```\n    ```\n/close\n```\n/close", "```\n    ```\n/close\n```\n\\/close"},
	} {
		if got := Inert(c.in); got != c.want {
			t.Errorf("Inert(%q) = %q, want %q", c.in, got, c.want)
		}
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

// A text the agent wrote with a code fence is kept whole in the state: the
// block's fence is longer, so nothing in it ends the block, and a draft
// accepted later writes the text as proposed.
func TestStateKeepsAFence(t *testing.T) {
	v := "Run:\n\n```sh\ngo test ./...\n```\n\n````md\nfour\n````\n/close"
	s := State{Confirmed: "abc", Label: LabelProposed, Proposed: []Pending{{Act: "refine", Line: "Refine.", Proposal: &Proposal{Do: "refine", Issue: 9, Verification: v}}}}
	got := FormatState(s)
	st, found, err := ReadState([]string{got + "\n\n" + StateMarker("product-owner")}, "product-owner")
	if !found || err != nil || st.Proposed[0].Proposal.Verification != v {
		t.Fatalf("read back: %v %v %+v\n%s", found, err, st, got)
	}
	// Escaped again as a comment written by the engine: the block, fenced,
	// is left as it is.
	if st, _, err := ReadState([]string{Inert(got) + "\n\n" + StateMarker("product-owner")}, "product-owner"); err != nil || st.Proposed[0].Proposal.Verification != v {
		t.Errorf("a line inside the block escaped: %v\n%s", err, Inert(got))
	}
	// An older engine's block, three backticks, still reads.
	old := "What workline knows.\n\n```yaml\nsources: []\nconfirmed: abc\n```\n\n" + StateMarker("product-owner")
	if st, found, err := ReadState([]string{old}, "product-owner"); !found || err != nil || st.Confirmed != "abc" {
		t.Errorf("an older block: %v %v %+v", found, err, st)
	}
}

// A label's description fits GitHub's limit, 100 characters, in either
// spelling.
func TestProposedSaysFits(t *testing.T) {
	for _, l := range acceptedLabels {
		if n := len([]rune(proposedSays(l))); n > 100 {
			t.Errorf("%s: %d characters", l, n)
		}
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
	for _, want := range []string{"**The product owner**, on 2026-10-08:\n\n- Refined #9: Need (draft).", "**Proposes**: Rename it to \"A title\". Why: '''x'''",
		"label `workline::accepted` to agree", "take `workline::proposed` off", "<details><summary>What workline knows of this issue, edited by the engine</summary>"} {
		if !strings.Contains(got, want) {
			t.Errorf("state says %q, want it to hold %q", got, want)
		}
	}
	if st, found, err := ReadState([]string{got + "\n\n" + StateMarker("product-owner")}, "product-owner"); !found || err != nil || st.Proposed[0].Proposal.Title != "A title" {
		t.Errorf("read back: %+v %v %v", st, found, err)
	}
}
