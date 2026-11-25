package engine

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// A record names, never calls: Given a head pushed after JN0V's
// `workline:accepted`, When workline takes the label off and comments,
// Then the comment names JN0V and sends no mention notification.
func TestFollowRecordNamesWhoseYesItTookOffWithoutMention(t *testing.T) {
	for _, v := range []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI"} {
		t.Setenv(v, "")
	}
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_AUTHOR_NAME", "T")
	t.Setenv("GIT_AUTHOR_EMAIL", "t@example.invalid")
	t.Setenv("GIT_COMMITTER_NAME", "T")
	t.Setenv("GIT_COMMITTER_EMAIL", "t@example.invalid")
	repo, run, write := gitRepo(t)
	origin := filepath.Join(t.TempDir(), "origin.git")
	run("init", "-q", "--bare", "-b", "main", origin)
	run("remote", "add", "origin", origin)
	write("a.txt", "a\n")
	run("add", "a.txt")
	run("commit", "-q", "-m", "first")
	run("push", "-q", "origin", "main")
	run("checkout", "-q", "-b", "feat/sessions")
	write("b.txt", "b\n")
	run("add", "b.txt")
	run("commit", "-q", "-m", "feat: b")
	write("c.txt", "c\n")
	run("add", "c.txt")
	run("commit", "-q", "-m", "feat: c, pushed after the yes")
	run("push", "-q", "origin", "feat/sessions")
	run("checkout", "-q", "main")

	state := `{"issues": [], "merge-requests": [{"id": 7, "branch": "feat/sessions", "base": "main", "author": "zed",
		"labels": ["workline:accepted"],
		"label-events": [{"label": "workline:accepted", "added": true, "author": "JN0V", "insider": true, "writer": true, "created": "2026-01-04T10:00:00Z"}],
		"pushes": [{"head": "feat/sessions~1", "at": "2026-01-04T09:00:00Z", "by": "zed"}, {"head": "feat/sessions", "at": "2026-01-04T11:00:00Z", "by": "zed"}],
		"comments": []}]}`
	path := filepath.Join(t.TempDir(), "forge.json")
	if err := os.WriteFile(path, []byte(state), 0o644); err != nil {
		t.Fatal(err)
	}
	fake := &forge.Fake{Path: path, Repo: repo}

	res := &Result{}
	o := FollowOptions{Repo: repo, Base: "main", Merge: "cautious"}
	if _, err := mergePolicy(fake, o, false, res, map[string]kept{}); err != nil {
		t.Fatal(err)
	}

	notes, err := fake.Comments(forge.Target{Kind: "merge-request", ID: 7})
	if err != nil {
		t.Fatal(err)
	}
	mention := regexp.MustCompile(`@\w`)
	var took string
	for _, c := range notes {
		if strings.Contains(c, "took") && strings.Contains(c, "off") {
			took = c
		}
	}
	if took == "" {
		t.Fatalf("no comment saying the label was taken off: %q", notes)
	}
	if !strings.Contains(took, "JN0V") {
		t.Errorf("the comment does not name who gave the yes:\n%s", took)
	}
	if mention.MatchString(took) {
		t.Errorf("the comment mentions a person, it sends a notification:\n%s", took)
	}
}
