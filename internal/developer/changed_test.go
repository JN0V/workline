package developer

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// A held merge request's files are its diff from the merge base with its
// base, its head fetched from origin (#1256): main moving on after the
// branch was cut adds nothing; a head that cannot be fetched is unread.
func TestChangedReadsTheDiffFromOrigin(t *testing.T) {
	dir := t.TempDir()
	repo, origin := filepath.Join(dir, "repo"), filepath.Join(dir, "origin.git")
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = repo
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1",
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.invalid", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.invalid")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out)
		}
	}
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(name, s string) {
		if err := os.WriteFile(filepath.Join(repo, name), []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run("git", "init", "-q", "-b", "main")
	write("a.txt", "a\n")
	write("b.txt", "b\n")
	run("git", "add", ".")
	run("git", "commit", "-q", "-m", "first")
	run("git", "init", "-q", "--bare", origin)
	run("git", "remote", "add", "origin", origin)
	run("git", "checkout", "-q", "-b", "workline/developer/1")
	write("a.txt", "a2\n")
	write("c.txt", "c\n")
	run("git", "add", ".")
	run("git", "commit", "-q", "-m", "change")
	run("git", "checkout", "-q", "main")
	write("b.txt", "b2\n")
	run("git", "commit", "-qam", "main moves on")
	run("git", "push", "-q", "origin", "main", "workline/developer/1")
	run("git", "branch", "-q", "-D", "workline/developer/1")

	changed := Changed(repo, false)
	files, ok := changed(forge.MergeRequest{ID: 4, Branch: "workline/developer/1", Base: "main", Here: true})
	if !ok || !slices.Equal(files, []string{"a.txt", "c.txt"}) {
		t.Errorf("changed = %v, %v: want a.txt and c.txt, read", files, ok)
	}
	if files, ok := changed(forge.MergeRequest{ID: 5, Branch: "workline/developer/9", Base: "main", Here: true}); ok {
		t.Errorf("a branch origin lacks read as %v", files)
	}
	if Changed("", false) != nil {
		t.Error("no repository reads something")
	}
}

// A held issue whose pull request's diff was read holds that diff alone,
// not its Scope nor the files listed beyond it; one unread holds its
// prediction (#1256).
func TestHeldHoldsTheDiffRead(t *testing.T) {
	issues := []forge.Issue{
		{ID: 1, Labels: []string{"workline:developing"}, Body: "## Scope\n\n- `src/lib.sh`.\n"},
		{ID: 2, Labels: []string{"workline:developing"}, Body: "## Scope\n\n- `src/other.sh`.\n"},
	}
	mrs := []forge.MergeRequest{
		{ID: 4, Branch: "b1", Body: "Closes #1\n\n## " + BeyondHeading + "\n\n- `go.mod`: a newer module\n"},
		{ID: 5, Branch: "b2", Body: "Closes #2"},
	}
	diffs := func(m forge.MergeRequest) ([]string, bool) {
		if m.Branch == "b1" {
			return []string{"build.gradle.kts"}, true
		}
		return nil, false
	}
	held := Held(issues, mrs, "workline:developing", nil, nil, diffs)
	if !slices.Equal(held[1], []string{"build.gradle.kts"}) {
		t.Errorf("held[1] = %v, want its diff alone", held[1])
	}
	if !slices.Equal(held[2], []string{"src/other.sh"}) {
		t.Errorf("held[2] = %v, want its Scope, its diff unread", held[2])
	}
}
