package backlog

import (
	"slices"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/work"
)

// newNames gives two meanings a new name in front, the old one still read,
// as ADR-0039 does: every read and write of a section must follow, and
// nothing else than the table changes.
func newNames(t *testing.T) {
	was := work.Headings
	t.Cleanup(func() { work.Headings = was })
	work.Headings = work.Headings.With(work.Need, "Problem").With(work.Validation, "Accepted when")
}

const (
	oldNames = "## Need\n\nA.\n\n## Validation\n\nB.\n\n---\n\n## Verification\n\nC.\n\n## Scope\n\nD."
	noAccept = "## Need\n\nA.\n\n---\n\n## Verification\n\nC.\n\n## Scope\n\nD."
)

func TestReadyFollowsTheTable(t *testing.T) {
	newNames(t)
	newer := strings.NewReplacer("## Need", "## Problem", "## Validation", "## Accepted when").Replace(oldNames)
	mixed := strings.Replace(oldNames, "## Need", "## Problem", 1)
	for _, body := range []string{oldNames, newer, mixed} {
		if missing := NotReady(body, nil, false); missing != nil {
			t.Errorf("%q: not ready: %v", body, missing)
		}
	}
	if missing := NotReady(noAccept, nil, false); !slices.Equal(missing, []string{"Accepted when is missing"}) {
		t.Errorf("a missing section is named as the table writes it: %v", missing)
	}
	// A draft recorded under the old heading is the role's under the new.
	draft := "\n\n" + DraftLine("product-owner") + "\n\nB."
	st := &State{Wrote: map[string]string{"Validation": BodyDigest(draft)}}
	for _, h := range []string{"## Validation", "## Accepted when"} {
		body := "## Need\n\nA.\n\n" + h + draft + "\n\n## Verification\n\nC.\n\n## Scope\n\nD."
		if !IsDraft(body, work.Validation, st) || NotReady(body, st, false) == nil || NotReady(body, st, true) != nil {
			t.Errorf("%s: a draft of the role's not accepted holds ready back, accepted it does not", h)
		}
		if own := Own(forge.Issue{Body: body}, st); !slices.Equal(own, []string{strings.TrimPrefix(h, "## ")}) {
			t.Errorf("%s: Own = %v", h, own)
		}
	}
}

func TestDraftingFollowsTheTable(t *testing.T) {
	newNames(t)
	// A missing section is written under the new name, where the layout
	// puts it among the old ones: after the need, above the line.
	body, added, _ := Refine(noAccept, Proposal{Validation: "B."}, "product-owner")
	want := "## Need\n\nA.\n\n## Accepted when\n\n" + DraftLine("product-owner") + "\n\nB.\n\n---\n\n## Verification\n\nC.\n\n## Scope\n\nD."
	if body != want || !slices.Equal(added, []string{"Accepted when"}) {
		t.Errorf("Refine = %q, %v\nwant %q", body, added, want)
	}
	// One there under an old name is that section: filled in place when
	// empty, kept when written, never added twice.
	empty := strings.Replace(oldNames, "B.", "_No response_", 1)
	body, added, _ = Refine(empty, Proposal{Validation: "B."}, "product-owner")
	if !slices.Equal(added, []string{"Validation"}) || strings.Contains(body, "Accepted when") || !strings.Contains(body, "## Validation\n\n"+DraftLine("product-owner")+"\n\nB.") {
		t.Errorf("an empty section under its old name: Refine = %q, %v", body, added)
	}
	body, added, kept := Refine(oldNames, Proposal{Need: "A2.", Validation: "B2."}, "product-owner")
	if body != oldNames || added != nil || !slices.Equal(kept, []string{"Need", "Validation"}) {
		t.Errorf("sections there under their old names: Refine = %q, %v, kept %v", body, added, kept)
	}
	// The role's own, under an old name, rewritten in place to answer.
	body, added, _ = Refine(oldNames, Proposal{Validation: "B2.", Revise: []string{"Accepted when"}}, "product-owner")
	if !slices.Equal(added, []string{"Validation"}) || !strings.Contains(body, "## Validation\n\n"+DraftLine("product-owner")+"\n\nB2.") {
		t.Errorf("revised under its old name: Refine = %q, %v", body, added)
	}
	// What a refine proposes is said under the names it writes.
	c := Proposal{Need: "A.", Validation: "B.", Added: []string{"Need", "Accepted when"}}
	if d := drafts(c); !strings.Contains(d, "**Need**\n\nA.") || !strings.Contains(d, "**Accepted when**\n\nB.") {
		t.Errorf("drafts = %q", d)
	}
	if !Drafted("Accepted when") || !Drafted("Validation") || Drafted("Verification") {
		t.Error("a section is drafted by its meaning, under any of its headings")
	}
}

func TestDeletedSectionFollowsTheTable(t *testing.T) {
	newNames(t)
	// Written under the old name, deleted by a person: never written again
	// under the new one.
	st := &State{Wrote: map[string]string{"Need": "aaaaaaaaaaaa", "Validation": "bbbbbbbbbbbb"}}
	deleted := DeletedSections(st, noAccept)
	if !slices.Equal(deleted, []string{"Validation"}) {
		t.Fatalf("DeletedSections = %v", deleted)
	}
	body, added, _ := Refine(noAccept, Proposal{Validation: "B.", Refused: deleted}, "product-owner")
	if body != noAccept || added != nil {
		t.Errorf("a section deleted under its old name written again: %q, %v", body, added)
	}
	st.Deleted = deleted
	if lacking(forge.Issue{Body: noAccept}, st) {
		t.Error("a section a person deleted is theirs, not lacking")
	}
	if again := DeletedSections(st, strings.Replace(noAccept, "## Need", "## Problem", 1)); !slices.Equal(again, deleted) {
		t.Errorf("a section deleted is recorded once: %v", again)
	}
	// Recorded again under the new name: the old record goes.
	st.SetWrote("Problem", "cccccccccccc")
	if _, ok := st.Wrote["Need"]; ok || st.Wrote["Problem"] != "cccccccccccc" {
		t.Errorf("SetWrote = %v", st.Wrote)
	}
}

func TestWatchedFollowsTheTable(t *testing.T) {
	newNames(t)
	st := &State{Sections: map[string]string{"Need": "Every row exported."}}
	if got := Rewritten(st, "## Problem\n\nEvery row  exported."); got != nil {
		t.Errorf("a heading renamed, its text as it was: Rewritten = %v", got)
	}
	body := "## Problem\n\nEvery row exported as JSON."
	if got := Rewritten(st, body); !slices.Equal(got, []string{"Problem"}) {
		t.Errorf("Rewritten = %v", got)
	}
	if st.KeptText("Problem") != "Every row exported." || BasisText(body, "Need") != "Every row exported as JSON." {
		t.Errorf("kept %q, now %q", st.KeptText("Problem"), BasisText(body, "Need"))
	}
	if got := Rewritten(st, "Its need, gone."); !slices.Equal(got, []string{"Need"}) {
		t.Errorf("a section gone is named as it was kept: %v", got)
	}
}
