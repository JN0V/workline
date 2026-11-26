package engine

import (
	"slices"
	"testing"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
)

// Given an issue the role labelled `task`, when a full read corrects it to
// `bug` and the same act is applied twice (two runs overlapping), then the
// label is `bug` both times, the state says it was corrected once, and a
// later swap is kept off as a person's.
func TestKindCorrectedOnceAppliedTwice(t *testing.T) {
	f, a := issueIn(t, "task")
	tg := forge.Target{Kind: "issue", ID: 1}
	if err := f.Sticky(tg, backlog.FormatState(backlog.State{Confirmed: "d8c4dff", Kind: "task"}), backlog.StateMarker(a.role), true); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := a.kind(backlog.Proposal{Do: "kind", Issue: 1, IssueKind: "bug", KindLabel: "bug"}); err != nil {
			t.Fatal(err)
		}
		is, _ := f.Issue(1)
		if !slices.Contains(is.Labels, "bug") || slices.Contains(is.Labels, "task") {
			t.Fatalf("labels %v, want bug alone", is.Labels)
		}
	}
	comments, _ := f.Comments(tg)
	if st, _, _ := backlog.ReadState(comments, a.role); st.Kind != "bug, corrected" {
		t.Errorf("state kind %q, want bug, corrected", st.Kind)
	}
	if err := a.kind(backlog.Proposal{Do: "kind", Issue: 1, IssueKind: "feature", KindLabel: "enhancement"}); err != nil {
		t.Fatal(err)
	}
	if is, _ := f.Issue(1); !slices.Equal(is.Labels, []string{"bug"}) {
		t.Errorf("labels %v after a second correction, want bug kept", is.Labels)
	}
}

// Given an issue with no kind, when the same kind is applied twice (two
// runs overlapping), then its label stays on.
func TestKindSetTwiceStays(t *testing.T) {
	f, a := issueIn(t)
	tg := forge.Target{Kind: "issue", ID: 1}
	if err := f.Sticky(tg, backlog.FormatState(backlog.State{Confirmed: "d8c4dff"}), backlog.StateMarker(a.role), true); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := a.kind(backlog.Proposal{Do: "kind", Issue: 1, IssueKind: "bug", KindLabel: "bug"}); err != nil {
			t.Fatal(err)
		}
	}
	if is, _ := f.Issue(1); !slices.Equal(is.Labels, []string{"bug"}) {
		t.Errorf("labels %v, want bug", is.Labels)
	}
}
