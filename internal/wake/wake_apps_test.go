package wake

import (
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/JN0V/workline/internal/forge"
)

// appMergeForge lists one app's merge, closing no issue.
type appMergeForge struct {
	forge.Forge
	as string
}

func (s appMergeForge) MergedSince(time.Time) ([]forge.MergeRequest, error) {
	return []forge.MergeRequest{{ID: 12, MergedAs: s.as, Author: "release-app[bot]", Bot: true}}, nil
}
func (appMergeForge) Closes([]int) (map[int][]forge.Issue, error) {
	return map[int][]forge.Issue{}, nil
}
func (appMergeForge) Labelled(string) ([]forge.Target, error) { return nil, nil }

// An app's routine merge:
// Given a merge opened by an app and closing no issue,
// When the wake reads it,
// Then it is not listed under "closes no issue", and the Facts line still
// counts it among the apps' merges.
func TestAppMergeClosingNoIssueNotListed(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	repo := t.TempDir()
	for _, args := range [][]string{{"init", "-q", "-b", "main"}, {"commit", "-q", "--allow-empty", "-m", "a"}, {"commit", "-q", "--allow-empty", "-m", "b"}} {
		cmd := exec.Command("git", append([]string{"-C", repo, "-c", "user.name=t", "-c", "user.email=t@example.invalid", "-c", "core.hooksPath=/dev/null"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	head, _ := gitIn(repo).commit("HEAD")
	r, err := Read(Options{Repo: repo, After: "HEAD", Forge: appMergeForge{as: head}})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range r.Entries {
		if e.Kind == Need {
			t.Errorf("an app's merge listed: %+v", e)
		}
	}
	if !strings.Contains(r.Facts, "1 by apps") {
		t.Errorf("facts %q do not count the app's merge", r.Facts)
	}
}
