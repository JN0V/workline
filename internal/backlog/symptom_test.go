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
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := &Plan{issues: map[int]forge.Issue{3: {ID: 3, Labels: tc.labels}}, config: Config{Tested: tc.tested}}
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
