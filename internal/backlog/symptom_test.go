package backlog

import (
	"errors"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// A bug is split only once the tester's Draft holds its tests (#1212); a
// feature, a person's yes, or a line with no tester known, as before.
func TestBugSplitWaitsForItsTests(t *testing.T) {
	into := []Child{
		{Title: "Multiply", Need: "n", Validation: "v", Scope: "s", Sources: []string{"lib.sh"}},
		{Title: "Divide", Need: "n", Validation: "v", Scope: "s", Sources: []string{"lib.sh"}},
	}
	for _, tc := range []struct {
		name   string
		labels []string
		tested func(int) (bool, error)
		ticked string
		rule   string
	}{
		{"bug, no Draft", []string{"bug"}, func(int) (bool, error) { return false, nil }, "", "bug-tests-first"},
		{"bug, Drafts unread", []string{"bug"}, func(int) (bool, error) { return false, errors.New("down") }, "", "bug-tests-unread"},
		{"bug, its Draft open", []string{"bug"}, func(int) (bool, error) { return true, nil }, "", ""},
		{"bug, a person's yes", []string{"bug"}, func(int) (bool, error) { return false, nil }, "ann", ""},
		{"bug, no tester known", []string{"bug"}, nil, "", ""},
		{"feature", []string{"enhancement"}, func(int) (bool, error) { return false, nil }, "", ""},
		{"bug, its tests crossed its size", []string{"bug", LabelBlocked}, func(int) (bool, error) { return false, nil }, "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := &Plan{issues: map[int]forge.Issue{3: {ID: 3, Labels: tc.labels}}, config: Config{Tested: tc.tested, Sized: 3, SizeAsked: true}}
			c := Proposal{Do: "split", Issue: 3, Into: into, Ticked: tc.ticked}
			rule, why := p.checkSplitRename(".", &c)
			if rule == "sources-unknown" {
				rule = "" // past the hold: the fixture's files are not this folder's
			}
			if rule != tc.rule {
				t.Fatalf("rule = %q (%s), want %q", rule, why, tc.rule)
			}
		})
	}
}

// A ready `too-big` holds on a bug in refine, sized in this run, its
// tests not yet written: they go first, then its split is asked (#1212).
func TestReadyTooBigWaitsForTests(t *testing.T) {
	untested := func(int) (bool, error) { return false, nil }
	for _, tc := range []struct {
		name    string
		labels  []string
		body    string
		sized   int
		tested  func(int) (bool, error)
		dropped string // in why, "" when it holds
	}{
		{"bug in refine", []string{"bug", LabelToRefine}, "", 3, untested, ""},
		{"feature", []string{"enhancement", LabelToRefine}, "", 3, untested, "only a bug"},
		{"no tester", []string{"bug", LabelToRefine}, "", 3, nil, "only a bug"},
		{"not sized here", []string{"bug", LabelToRefine}, "", 4, untested, "not asked"},
		{"a part", []string{"bug", LabelToRefine}, "Part of #1.", 3, untested, "never split again"},
		{"tests written", []string{"bug", LabelToRefine}, "", 3, func(int) (bool, error) { return true, nil }, "written already"},
		{"Drafts unread", []string{"bug", LabelToRefine}, "", 3, func(int) (bool, error) { return false, errors.New("down") }, "could not be read"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			is := forge.Issue{ID: 3, Labels: tc.labels, Body: tc.body}
			p := &Plan{issues: map[int]forge.Issue{3: is}, config: Config{Tested: tc.tested, Sized: tc.sized}}
			why := p.splitAfterTests(is, Proposal{Do: "ready", Issue: 3, TooBig: true})
			if tc.dropped == "" && why != "" || !strings.Contains(why, tc.dropped) {
				t.Errorf("why = %q, want %q", why, tc.dropped)
			}
		})
	}
}

// A bug's parent is closed alone only on its symptom green, run once all
// its parts were closed; the reason said otherwise.
func TestBugParentReadsItsSymptom(t *testing.T) {
	green := Symptom{Issue: 3, Branch: "main", Commit: "4369771f0a59", Parts: []int{6, 7}, Tests: []string{"tests.t1"}, Green: true}
	for _, tc := range []struct {
		name    string
		labels  []string
		symptom func(int) (Symptom, bool, error)
		said    string // in c.Symptom
		why     string // in c.Why
	}{
		{"green", []string{"bug"}, func(int) (Symptom, bool, error) { return green, true, nil }, "`tests.t1` green on main at 4369771f", ""},
		{"none", []string{"bug"}, func(int) (Symptom, bool, error) { return Symptom{}, false, nil }, "", "no reproduction test of its own"},
		{"red", []string{"bug"}, func(int) (Symptom, bool, error) {
			return Symptom{Issue: 3, Parts: []int{6, 7}, Why: "red on main"}, true, nil
		}, "", "its closing is a person's: red on main"},
		{"ran before a part closed", []string{"bug"}, func(int) (Symptom, bool, error) {
			s := green
			s.Parts = []int{6}
			return s, true, nil
		}, "", "ran before #7 closed"},
		{"unread", []string{"bug"}, func(int) (Symptom, bool, error) { return Symptom{}, false, errors.New("down") }, "", "could not be read (down)"},
		{"feature", []string{"enhancement"}, func(int) (Symptom, bool, error) { return green, true, nil }, "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := &Plan{issues: map[int]forge.Issue{3: {ID: 3, Labels: tc.labels, Children: []int{6, 7}}}, config: Config{Symptom: tc.symptom}}
			c := Proposal{Do: "close", Issue: 3, Parent: true, Why: "its 2 parts are closed", Symptom: "forged"}
			p.bugParent(&c, nil)
			if tc.said == "" && c.Symptom != "" || !strings.Contains(c.Symptom, tc.said) {
				t.Errorf("Symptom = %q, want %q", c.Symptom, tc.said)
			}
			if !strings.Contains(c.Why, tc.why) || tc.why == "" && c.Why != "its 2 parts are closed" {
				t.Errorf("Why = %q, want it to hold %q", c.Why, tc.why)
			}
		})
	}
}

// A proof older than the tester's run the line started proves nothing:
// that run failed, and left it in place.
func TestSymptomOlderThanItsRunIsNoProof(t *testing.T) {
	s := Symptom{Issue: 3, Green: true, Ran: "2026-11-10T08:00:00Z", Parts: []int{6}}
	if got := s.Since("2026-11-10T08:00:00Z"); !got.Green {
		t.Errorf("written at its run: %+v, want green", got)
	}
	if got := s.Since(""); !got.Green {
		t.Errorf("no run named: %+v, want green", got)
	}
	got := s.Since("2026-11-10T09:00:00Z")
	if got.Green || !strings.Contains(got.Proves([]int{6}), "older than its run") {
		t.Errorf("written before its run: %+v, want no proof", got)
	}
}
