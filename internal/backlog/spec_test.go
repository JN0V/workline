package backlog

import (
	"slices"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/work"
)

func TestSpecReviewReadsBack(t *testing.T) {
	r := SpecReview{Body: "3f9a1c0e2b7d", Round: 2, Open: 1, In: []string{"Verification", "Scope"}, Stopped: true}
	got, err := ReadSpecReview([]string{"state", "findings\n\n" + r.String() + "\n\n" + SpecMarker})
	if err != nil || got == nil || got.Body != r.Body || got.Round != 2 || got.Open != 1 || !slices.Equal(got.In, r.In) || !got.Stopped {
		t.Fatalf("read back %+v (%v) from %q", got, err, r.String())
	}
	if got, err := ReadSpecReview([]string{r.String()}); got != nil || err != nil {
		t.Fatalf("a record without the reviewer's marker read: %+v, %v", got, err)
	}
}

// A record the reviewer's comment holds that does not read is said (#302),
// never taken for none; the line is quoted as code, a summary showing it.
func TestSpecReviewUnreadSaid(t *testing.T) {
	for _, c := range []struct{ comment, says string }{
		{"<!-- workline:spec-review body=d424ff914f28 round=1 open=1 in=Steps to reproduce -->", "`<!-- workline:spec-review body=d424ff914f28 round=1 open=1 in=Steps to reproduce -->` does not read"},
		{"<!-- workline:spec-review body=d424ff914f28 round=1 open=1 in=Need%zz -->", "in=Need%zz -->` does not read"},
		{"Findings, the record taken out by hand.", "no `workline:spec-review` line in it"},
	} {
		got, err := ReadSpecReview([]string{c.comment + "\n\n" + SpecMarker})
		if got != nil || err == nil || !strings.Contains(err.Error(), c.says) {
			t.Errorf("%q: %+v, %v; want an error saying %q", c.comment, got, err, c.says)
		}
		is := forge.Issue{Body: "## Need\n\nA."}
		if rule, why := SpecHold(is, []string{c.comment + "\n\n" + SpecMarker}); rule != SpecRecordUnread || !strings.Contains(why, "held from ready") {
			t.Errorf("%q: held %q, %q", c.comment, rule, why)
		}
		if r, cleared := SpecCleared(is, []string{c.comment + "\n\n" + SpecMarker}); r != nil || cleared {
			t.Errorf("%q: cleared %+v", c.comment, r)
		}
	}
}

// The record holds the sections under any heading a body can have, and a
// record written before, its headings of letters only, reads the same and
// is written the same.
func TestSpecReviewHoldsAnyHeading(t *testing.T) {
	in := []string{work.Name(work.Steps), "Acceptance criteria", "What happened?", "Notes, ideas", "Critères", "a --> b", "C++ at 100%"}
	r := SpecReview{Body: "3f9a1c0e2b7d", Round: 3, Open: 2, In: in}
	got, err := ReadSpecReview([]string{r.String() + "\n\n" + SpecMarker})
	if err != nil || got == nil || got.Round != 3 || got.Open != 2 || !slices.Equal(got.In, in) {
		t.Fatalf("read back %+v (%v) from %q", got, err, r.String())
	}
	old := "<!-- workline:spec-review body=d424ff914f28 round=1 open=2 in=Verification,Scope -->"
	if got, err := ReadSpecReview([]string{old + "\n\n" + SpecMarker}); err != nil || got == nil || got.Round != 1 || !slices.Equal(got.In, []string{"Verification", "Scope"}) {
		t.Fatalf("a record written before read %+v (%v)", got, err)
	}
	if s := (SpecReview{Body: "d424ff914f28", Round: 1, Open: 2, In: []string{"Verification", "Scope"}}).String(); s != old {
		t.Fatalf("headings of letters only written %q, want as before: %q", s, old)
	}
	if got, err := ReadSpecReview([]string{"<!-- workline:spec-review body= round=1 open=0 in= -->\n\n" + SpecMarker}); err != nil || got == nil || got.In != nil {
		t.Fatalf("a record of no section read %+v (%v)", got, err)
	}
}

func TestLeftOpenHoldsTheRolesOwnOpenWork(t *testing.T) {
	is := forge.Issue{ID: 9, Author: "ann", Body: "## Need\n\nA sale counted once."}
	digest := BodyDigest(is.Body)
	ask := forge.Note{Body: "@ann, to refine this issue: Split it?\n\n<!-- workline:" + Keeper + "/ask -->"}
	answer := forge.Note{Body: "One issue.", Author: "ann"}
	for _, c := range []struct {
		name  string
		st    *State
		notes []forge.Note
		held  bool
	}{
		{"nothing pending", &State{}, nil, false},
		{"question open", &State{}, []forge.Note{ask}, true},
		{"question open, no state", nil, []forge.Note{ask}, true},
		{"question assumed", &State{Assumed: 1}, []forge.Note{ask}, false},
		{"question answered", &State{}, []forge.Note{ask, answer}, false},
		{"moved back on this body", &State{Did: []Did{{Act: "unready", Body: digest}}}, nil, true},
		{"moved back on an older body", &State{Did: []Did{{Act: "unready", Body: "111111111111"}}}, nil, false},
		{"moved back by an engine keeping no body", &State{Did: []Did{{Act: "unready"}}}, nil, false},
		{"readied after the move", &State{Did: []Did{{Act: "unready", Body: digest}, {Act: "ready", Body: digest}}}, nil, false},
		{"moved back after a ready", &State{Did: []Did{{Act: "ready", Body: digest}, {Act: "unready", Body: digest}, {Act: "refine"}}}, nil, true},
	} {
		if got := LeftOpen(c.st, c.notes, is, Keeper); (got != "") != c.held {
			t.Errorf("%s: LeftOpen = %q, held %v", c.name, got, c.held)
		}
	}
}

func TestSectionAt(t *testing.T) {
	body := "Intro.\n\n## Need\n\nA.\n\n### Verification\n\nB."
	for line, want := range map[int]string{1: "", 3: "Need", 5: "Need", 7: "Verification", 9: "Verification"} {
		if got := SectionAt(body, line); got != want {
			t.Errorf("line %d: %q, want %q", line, got, want)
		}
	}
}

// The one rule (ADR-0060): the builder's part — the Scope — always the
// role's, even a person's text; the person's part only while still the
// role's own, a draft or a text as it wrote it; the Verification never,
// the tester's (ADR-0066).
func TestRevisableBuildersPartAlwaysPersonsPartWhenOwn(t *testing.T) {
	is := forge.Issue{Body: "## Need\n\n" + DraftLine("product-owner") + "\n\nA.\n\n## Verification\n\nB.\n\n## Validation\n\nC.\n\n## Scope\n\nD."}
	st := &State{Wrote: map[string]string{"Verification": BodyDigest("B."), "Scope": BodyDigest("not D.")}}
	r := &SpecReview{In: []string{"Need", "Verification", "Validation", "Scope"}}
	if got := Revisable(is, st, r); !slices.Equal(got, []string{"Need", "Scope"}) {
		t.Fatalf("revisable %v, want the draft and the Scope", got)
	}
	// A person wrote it whole: their Need and Validation theirs, the
	// Scope the role's, the Verification the tester's.
	person := forge.Issue{Body: "## Need\n\nA.\n\n## Example\n\nE.\n\n## Validation\n\nC.\n\n---\n\n## Verification\n\nB.\n\n## Scope\n\nD."}
	for _, st := range []*State{nil, {}} {
		if got := Revisable(person, st, r); !slices.Equal(got, []string{"Scope"}) {
			t.Errorf("a person's issue, state %v: revisable %v, want Scope", st, got)
		}
		if got := Own(person, st); !slices.Equal(got, []string{"Scope"}) {
			t.Errorf("a person's issue, state %v: own %v, want Scope", st, got)
		}
	}
	for m, want := range map[work.Meaning]bool{work.Need: false, work.Example: false, work.Validation: false, work.Verification: false, work.Scope: true} {
		if got := Rewritable(person, nil, m); got != want {
			t.Errorf("a person's %s rewritable: %v, want %v", m, got, want)
		}
	}
	// Accepted, a draft is a person's; the Scope stays the role's.
	accepted := is
	accepted.Labels = []string{LabelAccepted}
	if got := Revisable(accepted, &State{}, r); !slices.Equal(got, []string{"Scope"}) {
		t.Errorf("accepted: revisable %v, want Scope", got)
	}
}

// Findings in the Verification alone are the tester's (ADR-0066): they
// hold nothing from ready, and the product owner is not given them.
func TestVerificationFindingsAreTheTesters(t *testing.T) {
	body := "## Need\n\nA.\n\n## Validation\n\nC.\n\n---\n\n## Verification\n\nB.\n\n## Scope\n\nD."
	is := forge.Issue{Body: body}
	only := []string{"<!-- workline:spec-review body=" + BodyDigest(body) + " round=1 open=1 in=Verification -->\n\n" + SpecMarker}
	if rule, _ := SpecHold(is, only); rule != "" {
		t.Errorf("a finding in the Verification alone holds it: %s", rule)
	}
	if SpecOpen(is, only) != nil {
		t.Error("a finding in the Verification alone is given to the product owner")
	}
	if c, in := SpecForTester(only); c == "" || !slices.Equal(in, []string{"Verification"}) {
		t.Errorf("the tester is not given the finding: %q %v", c, in)
	}
	both := []string{"<!-- workline:spec-review body=" + BodyDigest(body) + " round=1 open=2 in=Scope,Verification -->\n\n" + SpecMarker}
	if rule, _ := SpecHold(is, both); rule != "spec-findings-open" {
		t.Errorf("a finding in the Scope too holds nothing: %q", rule)
	}
}

// The tester writes the Verification where there is none, rewrites its
// own, and keeps another's word for word (ADR-0066).
func TestWithVerificationKeepsAPersons(t *testing.T) {
	bare := "## Need\n\nA.\n\n## Validation\n\nC.\n\n---\n\n## Scope\n\nD."
	got, changed := WithVerification(bare, "- **C:** `TestC`.", nil)
	if !changed || !strings.Contains(got, "---\n\n## Verification\n\n"+TesterMarker+"\n\n- **C:** `TestC`.\n\n## Scope") {
		t.Fatalf("not written above the Scope: %v\n%s", changed, got)
	}
	again, changed := WithVerification(got, "- **C:** `TestC2`.", nil)
	if !changed || !strings.Contains(again, "`TestC2`") || strings.Contains(again, "`TestC`.") {
		t.Errorf("its own not rewritten: %v\n%s", changed, again)
	}
	person := strings.Replace(bare, "## Scope", "## Verification\n\nRun the suite.\n\n## Scope", 1)
	if kept, changed := WithVerification(person, "- **C:** `TestC`.", nil); changed || kept != person {
		t.Errorf("a person's rewritten: %v\n%s", changed, kept)
	}
	po := &State{}
	po.SetWrote("Verification", BodyDigest("Run the suite."))
	if out, changed := WithVerification(person, "- **C:** `TestC`.", po); !changed || strings.Contains(out, "Run the suite.") {
		t.Errorf("the product owner's own, from before ADR-0066, kept: %v\n%s", changed, out)
	}
	edited := strings.Replace(person, "Run the suite.", "Run the whole suite.", 1)
	if kept, changed := WithVerification(edited, "- **C:** `TestC`.", po); changed || kept != edited {
		t.Errorf("the product owner's, a person edited, rewritten: %v\n%s", changed, kept)
	}
}

// A review answered by asking a person for their Scope, under the rule
// before ADR-0060, is read again once: then never again for it.
func TestUnrevisedOnceAReview(t *testing.T) {
	body := "## Need\n\nA.\n\n## Validation\n\nC.\n\n---\n\n## Verification\n\nB.\n\n## Scope\n\nD."
	is := forge.Issue{Body: body}
	comments := []string{"<!-- workline:spec-review body=" + BodyDigest(body) + " round=1 open=1 in=Scope -->\n\n" + SpecMarker}
	if got := Unrevised(is, &State{Answered: BodyDigest(body)}, comments); !slices.Equal(got, []string{"Scope"}) {
		t.Fatalf("answered before the rule: unrevised %v, want Scope", got)
	}
	if got := Unrevised(is, &State{Answered: BodyDigest(body), Revised: []string{"Scope"}}, comments); len(got) != 0 {
		t.Errorf("answered with it to rewrite: unrevised %v, want none", got)
	}
	if got := Unrevised(is, &State{}, comments); len(got) != 0 {
		t.Errorf("not answered yet: unrevised %v, want none (read for the review itself)", got)
	}
	if got := Unrevised(is, nil, comments); len(got) != 0 {
		t.Errorf("no state: unrevised %v, want none", got)
	}
}
