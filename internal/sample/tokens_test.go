package sample

import (
	"errors"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// worksForge answers Issue and MergeRequest only.
type worksForge struct {
	forge.Forge
	issues map[int][]string
	mrs    map[int]forge.MergeRequest
}

func (f worksForge) Issue(id int) (*forge.Issue, error) {
	labels, ok := f.issues[id]
	if !ok {
		return nil, errors.New("no such issue")
	}
	return &forge.Issue{ID: id, Labels: labels}, nil
}

func (f worksForge) MergeRequest(id int) (forge.MergeRequest, error) {
	mr, ok := f.mrs[id]
	if !ok {
		return forge.MergeRequest{}, errors.New("no such merge request")
	}
	return mr, nil
}

// A merge request's work: its closed issue's kind first, else its
// title's type; ci and build are neither product nor upkeep.
func TestWorkOfAMergeRequest(t *testing.T) {
	f := worksForge{
		issues: map[int][]string{1: {"bug"}, 2: {}, 3: {"chore"}},
		mrs: map[int]forge.MergeRequest{
			10: {Title: "docs: a fix", Body: "Fixes #1"},
			11: {Title: "refactor: split it", Body: "Closes #2"},
			12: {Title: "ci: cache the build"},
			13: {Title: "Tidy up", Body: "Closes #2"},
			14: {Title: "chore(main): release 1.0.0"},
			15: {Title: "test: more", Body: "Closes #3"},
		},
	}
	w := works{f: f, issues: map[int]workKind{}, mrs: map[int]workKind{}}
	for id, want := range map[int]string{10: product, 11: product, 12: "", 13: "", 14: release, 15: upkeep, 99: ""} {
		if got := w.of(&forge.Target{Kind: "merge-request", ID: id}); got.kind != want || (want == "" && got.why == "") {
			t.Errorf("!%d: %+v, want %q", id, got, want)
		}
	}
}

// Runs whose calls say no tokens are counted, no share divided by zero.
func TestTokensWordsWithNoTokens(t *testing.T) {
	tk := &Tokens{Runs: 2}
	if got, want := tk.Words(), "0 fresh tokens over 2 runs: upkeep 0, product 0"; got != want {
		t.Errorf("%q, want %q", got, want)
	}
}
