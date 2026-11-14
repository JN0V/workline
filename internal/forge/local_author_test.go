package forge

import (
	"os"
	"os/exec"
	"testing"
)

// Person named: given a person's comment on a local merge request, when
// reconcile reads its notes for a review round, then the note names that
// person (the clone's git user.name), never an empty author.
func TestLocalNotesNameAuthor(t *testing.T) {
	for _, v := range []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI"} {
		t.Setenv(v, "")
	}
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	repo := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", repo, "-c", "core.hooksPath=/dev/null", "-c", "user.email=t@example.invalid"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
	}
	run("init", "-q", "-b", "main")
	run("config", "user.name", "Ann Person")
	run("commit", "-q", "--allow-empty", "-m", "first")
	run("checkout", "-q", "-b", "feature")
	run("commit", "-q", "--allow-empty", "-m", "feature")
	l := &Local{Repo: repo}
	id, err := l.OpenMergeRequest("feature", "main", "a change", "body", false)
	if err != nil {
		t.Fatal(err)
	}
	mr := Target{Kind: "merge-request", ID: id}
	if err := l.Comment(mr, "Please handle the empty case.", ""); err != nil {
		t.Fatal(err)
	}
	notes, err := l.Notes(mr)
	if err != nil || len(notes) != 1 {
		t.Fatalf("Notes = %+v, %v", notes, err)
	}
	if notes[0].Author != "Ann Person" {
		t.Errorf("author = %q, want the clone's user.name", notes[0].Author)
	}
}
