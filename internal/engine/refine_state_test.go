package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
)

// stateLabelsOf are the labels each naming a state of the line.
var stateLabelsOf = []string{backlog.LabelToRefine, backlog.LabelReady, backlog.LabelNeedsTests, "workline:developing", backlog.LabelBlocked}

// issueIn is a fake forge holding issue 1, complete, carrying labels, and
// an applier of the product owner on it.
func issueIn(t *testing.T, labels ...string) (*forge.Fake, *applier) {
	t.Helper()
	body := "## Need\n\nThe export keeps its last row.\n\n## Validation\n\n- **Last row:** Given three rows, When Ann exports them, Then she reads three.\n\n---\n\n## Scope\n\nEdit `export.go`.\n"
	data, err := json.Marshal(forge.FakeState{Issues: []forge.FakeItem{{ID: 1, Title: "Keep the last row", Body: body, Labels: labels, Author: "ann", Insider: true}}})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	f := &forge.Fake{Path: path}
	return f, &applier{forge: f, role: "product-owner"}
}

// carried is the issue's labels and the number of them that are states.
func carried(t *testing.T, f *forge.Fake) ([]string, int) {
	t.Helper()
	is, err := f.Issue(1)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, l := range is.Labels {
		if slices.Contains(stateLabelsOf, l) {
			n++
		}
	}
	return is.Labels, n
}

// Scope refined on a ready issue
// Given an issue in workline:ready,
// When the product owner's refine rewrites its Scope,
// Then the issue keeps workline:ready and does not get workline:to-refine.
func TestScopeRefinedOnAReadyIssue(t *testing.T) {
	f, a := issueIn(t, backlog.LabelReady)
	err := a.refining(backlog.Proposal{Do: "refine", Issue: 1, Scope: "Edit `export.go` and `csv.go`.", Revise: []string{"Scope"}})
	if err != nil {
		t.Fatal(err)
	}
	is, err := f.Issue(1)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(is.Body, "csv.go") {
		t.Fatalf("the refine did not rewrite the Scope: %q", is.Body)
	}
	if !slices.Contains(is.Labels, backlog.LabelReady) {
		t.Errorf("labels %v: workline:ready was lost, want it kept", is.Labels)
	}
	if slices.Contains(is.Labels, backlog.LabelToRefine) {
		t.Errorf("labels %v: workline:to-refine was added to a ready issue, want it not", is.Labels)
	}
}

// Never two states
// Given an issue in workline:ready, workline:needs-tests or workline:developing,
// When any product owner act is applied,
// Then the issue carries one state label at most.
func TestNeverTwoStates(t *testing.T) {
	for _, from := range []string{backlog.LabelReady, backlog.LabelNeedsTests, "workline:developing"} {
		acts := map[string]func(a *applier) error{
			"refine": func(a *applier) error {
				return a.refining(backlog.Proposal{Do: "refine", Issue: 1, Scope: "Edit `export.go` and `csv.go`.", Revise: []string{"Scope"}})
			},
			"ask": func(a *applier) error {
				return a.refining(backlog.Proposal{Do: "ask", Issue: 1, Questions: "Which export: CSV or JSON?", Round: 1})
			},
			"unready": func(a *applier) error {
				a.plan = &backlog.Plan{Decisions: []backlog.Decision{{Index: 0, Mode: backlog.Act, Act: backlog.Proposal{Do: "unready", Issue: 1}}}}
				a.index = 0
				return a.apply(intent.Intention{Kind: "unready", Value: map[string]any{"issue": 1, "why": "the need changed"}})
			},
		}
		for name, act := range acts {
			f, a := issueIn(t, from)
			if err := act(a); err != nil {
				t.Fatalf("%s on %s: %v", name, from, err)
			}
			if labels, n := carried(t, f); n > 1 {
				t.Errorf("%s on an issue in %s: labels %v carry %d states, want one at most", name, from, labels, n)
			}
		}
	}
}
