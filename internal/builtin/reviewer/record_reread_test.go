package reviewer

import (
	"os"
	"slices"
	"testing"
)

// A standing block is read again whichever file the commit that fixes it
// changes: Given a standing block on a.go, When a commit fixes it in b.go
// only, Then its lens is asked and the block is read again, so it is
// answered or stands by the lens's own reading, never held unread.
func TestStandingBlockReadAgainWhenAnotherFileFixesIt(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	repo := gitRepo(t, map[string]string{"a.go": "package a\n\nfunc Mul() int { return 2 + 3 }\n", "b.go": "package a\n"})
	run, commit := gitSteps(t, repo)
	base := run("rev-parse", "HEAD")
	head := commit("b.go", "package a\n\nfunc mul(x, y int) int { return x * y }\n")

	block := Block{Lens: "correctness", Where: "a.go:3", Title: "Important: Mul adds",
		Cause: Quote{Path: "a.go", Quote: "func Mul() int { return 2 + 3 }"}}
	files, err := changedFiles(repo, base, head)
	if err != nil {
		t.Fatal(err)
	}
	if got := answering([]Block{block}, files, nil); !slices.Equal(got, []string{"correctness"}) {
		t.Errorf("answering: %v, want the correctness lens asked on a fix in b.go", got)
	}

	st := state{Head: head, From: base, Lenses: []string{"correctness"}, Standing: []Block{block}}
	left, err := reread(repo, Settings{}, st, candidates{}, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 0 {
		t.Errorf("reread: %v still stand, want the block read again and left out", left)
	}
}
