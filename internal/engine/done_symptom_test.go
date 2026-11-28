package engine

import (
	"strconv"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// No false message:
// Given a bug's parent closed as fixed, its tester's Draft open with tests
// that never reached main,
// When the line follows the roles' merge requests and settles what is done,
// Then no message says the Draft's tests have nothing left to merge.
func TestFollowNeverSaysUnmergedTestsHaveNothingLeftToMerge(t *testing.T) {
	repo, branches, _, _ := manyDrafts(t, 1)
	id, err := strconv.Atoi(strings.TrimPrefix(branches[0], testerDraft))
	if err != nil {
		t.Fatal(err)
	}
	l := &forge.Local{Repo: repo}
	if err := l.Close(id, 0); err != nil {
		t.Fatal(err)
	}
	res := Follow(FollowOptions{Repo: repo, Forge: "local", Base: "main"})
	if _, err := git(repo, nil, "cat-file", "-e", "main:test_"+strconv.Itoa(id)+".txt"); err == nil {
		t.Fatal("the Draft's test reached main on its own: the scenario's Given does not hold")
	}
	const false = "nothing left to merge"
	for _, f := range res.Findings {
		if strings.Contains(f.Message, false) {
			t.Errorf("finding %s on %s says %q while the tests never reached main: %s", f.Rule, f.Where, false, f.Message)
		}
	}
	items, err := l.Items("merge-request")
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range items {
		for _, c := range it.Comments {
			if strings.Contains(c, false) {
				t.Errorf("the comment on the Draft says %q while its tests never reached main:\n%s", false, c)
			}
		}
	}
}
