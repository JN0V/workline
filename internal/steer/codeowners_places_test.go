package steer

import (
	"os"
	"testing"
)

// Two lists: given .github/CODEOWNERS and a root CODEOWNERS with different
// paths, when a pull request changes a path only .github/CODEOWNERS lists,
// then the merge policy holds it, as GitHub reads .github/ first.
func TestFilesReadsGithubCodeownersBeforeRoot(t *testing.T) {
	repo, tip := commit(t, map[string]string{
		".github/CODEOWNERS": "/only-github.go @owner\n",
		"CODEOWNERS":         "/only-root.go @owner\n",
	})
	held, err := Files(repo, tip)
	if err != nil {
		t.Fatal(err)
	}
	if got := held("only-github.go"); got != ListedWhy {
		t.Errorf("only-github.go: %q, want %q", got, ListedWhy)
	}
	if got := held("only-root.go"); got != "" {
		t.Errorf("only-root.go: %q, want none", got)
	}
}

// The wake's own test guarded: given workline's CODEOWNERS, when a pull
// request changes the wake's forge test, then it waits for the person's
// label, as the wake's code does.
func TestWorklineCodeownersGuardsWakeTest(t *testing.T) {
	data, err := os.ReadFile("../../CODEOWNERS")
	if err != nil {
		t.Fatal(err)
	}
	co := ParseCodeowners("CODEOWNERS", data)
	if !co.Holds("internal/forge/wake.go") {
		t.Fatal("the wake's code is not held")
	}
	if !co.Holds("internal/forge/wake_test.go") {
		t.Error("internal/forge/wake_test.go is not held by CODEOWNERS")
	}
}
