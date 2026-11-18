package productowner

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// Parked issue in refine.
// Given an open issue labelled workline:to-refine and workline:parked,
// whose state woke the product owner, when the product owner runs, then
// it reads no issue: the parked one is not moved toward ready.
func TestRefineLeavesAParkedIssue(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull) // the machine's hooks stay out
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, c := range [][]string{{"init", "-q"}, {"add", "."}, {"-c", "user.name=T", "-c", "user.email=t@x.invalid", "commit", "-qm", "x"}} {
		if out, err := exec.Command("git", append([]string{"-C", repo}, c...)...).CombinedOutput(); err != nil {
			t.Fatal(string(out))
		}
	}
	state := forge.FakeState{Issues: []forge.FakeItem{{ID: 5, Title: "Refine me", Author: "reporter",
		Body: "A need that is not yet written out.", Labels: []string{"workline:to-refine", "workline:parked"}}}}
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	fakePath := filepath.Join(t.TempDir(), "forge.json")
	if err := os.WriteFile(fakePath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	runDir := t.TempDir()
	for _, d := range []string{"in", "out"} {
		if err := os.MkdirAll(filepath.Join(runDir, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(runDir, "in", "settings.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WORKLINE_FORGE", "fake:"+fakePath)
	t.Setenv("WORKLINE_ROLE", "product-owner")
	t.Setenv("WORKLINE_EVENT", "issue")
	t.Setenv("WORKLINE_TARGET", "issue:5")
	t.Setenv("WORKLINE_AI", "")
	t.Setenv("WORKLINE_JUDGED", "")
	t.Setenv("WORKLINE_ROLES_DIR", "")

	code := Pre(runDir, repo)
	read, err := os.ReadFile(filepath.Join(runDir, "in", "issues-read"))
	if err != nil {
		t.Fatalf("pre exited %d and listed no issues read: %v", code, err)
	}
	if string(read) != "" {
		t.Errorf("the product owner reads %q: a parked issue in refine is to be left alone", read)
	}
	if _, err := os.Stat(filepath.Join(runDir, "in", "task.md")); err == nil {
		t.Error("a task was written for the agent, though the only open issue is parked")
	}
}
