package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// closesAfterListing is a simulated forge on which a person closes the
// pull request between the run's one read of the open pull requests and
// the weighing of each: once the list is read, the state file is
// replaced by the closed one.
type closesAfterListing struct {
	*forge.Fake
	closed string
}

func (c closesAfterListing) ListMergeRequests() ([]forge.MergeRequest, error) {
	open, err := c.Fake.ListMergeRequests()
	if err != nil {
		return nil, err
	}
	return open, os.WriteFile(c.Path, []byte(c.closed), 0o644)
}

// Closed mid-run: given a pull request open when `workline follow`
// starts, accepted by dana, with a comment of lee's no role answered,
// when lee comments and closes it before the run weighs it, then no
// Held comment is posted on it and no persons-comment-open finding is
// said.
func TestFollowPostsNoHeldCommentOnAPullRequestClosedMidRun(t *testing.T) {
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
	run("push", "-q", "origin", "feat/sessions")
	run("checkout", "-q", "main")

	state := func(closed string) string {
		return `{"issues": [], "merge-requests": [{"id": 7, "branch": "feat/sessions", "base": "main", "author": "zed"` + closed + `,
			"labels": ["workline:accepted"],
			"label-events": [{"label": "workline:accepted", "added": true, "author": "dana", "insider": true, "writer": true, "created": "2026-01-04T10:00:00Z"}],
			"pushes": [{"head": "feat/sessions", "at": "2026-01-04T09:00:00Z", "by": "zed"}],
			"comments": [{"body": "Closed without merging, it is not needed.", "author": "lee", "insider": true, "created": "2026-01-04T09:30:00Z"}]}]}`
	}
	path := filepath.Join(t.TempDir(), "forge.json")
	if err := os.WriteFile(path, []byte(state("")), 0o644); err != nil {
		t.Fatal(err)
	}
	fake := &forge.Fake{Path: path, Repo: repo}
	f := closesAfterListing{Fake: fake, closed: state(`, "closed": true`)}

	res := &Result{}
	o := FollowOptions{Repo: repo, Base: "main", Merge: "cautious"}
	if _, err := mergePolicy(f, o, false, res, map[string]kept{}); err != nil {
		t.Fatal(err)
	}

	notes, err := fake.Comments(forge.Target{Kind: "merge-request", ID: 7})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range notes {
		if strings.Contains(c, forge.Marker(commentOpenMarker)) || strings.Contains(c, "**Held**") {
			t.Errorf("a Held comment on a pull request closed mid-run: %q", c)
		}
	}
	for _, x := range res.Findings {
		if x.Rule == "persons-comment-open" {
			t.Errorf("a pull request closed mid-run held for the person's comment: %+v", x)
		}
	}
}
