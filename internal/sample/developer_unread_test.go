package sample

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/developer"
	"github.com/JN0V/workline/internal/forge"
)

// Given a developer's record only another account wrote, When the weekly
// sample runs, Then its row says who wrote it and who is read, not just
// "cannot be read".
func TestDeveloperRecordByAnotherAccount(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
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
	branch := developer.Branch + "issue-7"
	run("init", "-q", "-b", "main")
	run("commit", "-q", "--allow-empty", "-m", "chore: start")
	run("checkout", "-q", "-b", branch)
	run("commit", "-q", "--allow-empty", "-m", "feat: the change", "-m", "Workline-Role: developer")

	marker := forge.Marker("sticky=developer/" + developer.RecordKey)
	state, _ := json.Marshal(forge.FakeState{MergeRequests: []forge.FakeItem{{
		ID: 3, Branch: branch, Base: "main",
		Comments: []forge.FakeComment{{Author: "someone-else", Body: "the record of its code\n\n" + marker}},
	}}})
	path := filepath.Join(t.TempDir(), "forge.json")
	if err := os.WriteFile(path, state, 0o600); err != nil {
		t.Fatal(err)
	}
	f := &forge.Fake{Path: path, Repo: repo}

	acts, _, err := developerActs(repo, f, []forge.MergeRequest{{ID: 3, Branch: branch}})
	if err != nil {
		t.Fatal(err)
	}
	if len(acts) != 1 {
		t.Fatalf("developer acts drawn: %d, want 1: %+v", len(acts), acts)
	}
	rd := &reader{o: Options{Repo: repo, Forge: f}}
	said := rd.developerMaterial(&acts[0], &material{})
	if !strings.Contains(said, "@someone-else") || !strings.Contains(said, "only workline's own account's is read") || !strings.Contains(said, "workline-bot") {
		t.Errorf("developerMaterial = %q, want it to say @someone-else wrote the record and that only workline's own account (workline-bot) is read", said)
	}
}
