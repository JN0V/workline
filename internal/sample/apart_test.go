package sample

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// commentsDown is a forge whose comments do not answer: the reviewer's
// findings cannot be read, the rest can.
type commentsDown struct{ *forge.Fake }

func (commentsDown) Comments(forge.Target) ([]string, error) {
	return nil, errors.New("comments down")
}

// Each role's acts are read apart: the reviewer's findings failing to read
// is said for the reviewer, and the tester's removal is still drawn.
func TestActsReadApart(t *testing.T) {
	repo := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-c", "core.hooksPath=/dev/null"}, args...)...)
		cmd.Dir = repo
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q", "-b", "main")
	run("commit", "-q", "--allow-empty", "-m", "chore: start")
	run("checkout", "-q", "-b", "workline/tester/curate-tests")
	run("commit", "-q", "--allow-empty", "-m", "test: curate tests, 1 test fewer", "-m", "Workline-Role: tester")
	state, _ := json.Marshal(forge.FakeState{MergeRequests: []forge.FakeItem{{ID: 2, Branch: "workline/tester/curate-tests", Base: "main"}}})
	path := filepath.Join(t.TempDir(), "forge.json")
	if err := os.WriteFile(path, state, 0o600); err != nil {
		t.Fatal(err)
	}
	h := newHeads()
	h.merged = []forge.MergeRequest{{ID: 2, Branch: "workline/tester/curate-tests"}}
	res := &Result{}
	acts := actsOfTheWindow(Options{Repo: repo, Forge: commentsDown{&forge.Fake{Path: path, Repo: repo}}}, res, h, nil, "", "")
	var tester int
	for _, a := range acts {
		if a.Role == testRole && a.Kind == "removal" {
			tester++
		}
	}
	if tester != 1 {
		t.Errorf("the tester's removal drawn %d times, want 1: %+v", tester, acts)
	}
	said := strings.Join(res.NotDrawn, "\n")
	if !strings.Contains(said, "the reviewer's findings: the forge did not answer") || strings.Contains(said, "the tester's acts") {
		t.Errorf("not drawn: %s", said)
	}
}
