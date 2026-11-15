package stall

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/JN0V/workline/internal/forge"
)

// stub is a forge that lists mrs and keeps verdicts by head; what else a
// forge does is never called.
type stub struct {
	forge.Forge
	forge.Merger
	mrs      []forge.MergeRequest
	verdicts map[string]string
}

func (s stub) ListMergeRequests() ([]forge.MergeRequest, error) { return s.mrs, nil }
func (s stub) Verdict(head string) (string, string, error) {
	return s.verdicts[head], "the line's verdict on its head is block (reviewer block): review-finding", nil
}

func git(t *testing.T, dir string, env []string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@example.invalid", "-c", "core.hooksPath=/dev/null"}, args...)...)
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// clone makes a repository whose branch workline/developer/1 is kept
// only under refs/remotes/origin, its head made 3 days ago, as a CI
// checkout of the forge keeps it; it returns the clone and the head.
func clone(t *testing.T) (string, string) {
	t.Helper()
	old := []string{"GIT_AUTHOR_DATE=" + time.Now().Add(-72*time.Hour).Format(time.RFC3339),
		"GIT_COMMITTER_DATE=" + time.Now().Add(-72*time.Hour).Format(time.RFC3339)}
	origin := t.TempDir()
	git(t, origin, old, "init", "-q", "-b", "main")
	git(t, origin, old, "commit", "-q", "--allow-empty", "-m", "first")
	git(t, origin, old, "checkout", "-q", "-b", "workline/developer/1")
	git(t, origin, old, "commit", "-q", "--allow-empty", "-m", "fix")
	head := git(t, origin, nil, "rev-parse", "HEAD")
	git(t, origin, nil, "checkout", "-q", "main")
	dir := t.TempDir()
	git(t, dir, nil, "clone", "-q", origin, ".")
	if out, err := exec.Command("git", "-C", dir, "rev-parse", "-q", "--verify", "refs/heads/workline/developer/1").Output(); err == nil {
		t.Fatalf("the branch is a local one too: %s", out)
	}
	return dir, head
}

// On GitHub, the wake's and the reconcile's checkouts keep the
// developer's branches under refs/remotes/origin: a stall is read there.
func TestReadsTheBranchFetchedFromOrigin(t *testing.T) {
	dir, head := clone(t)
	f := stub{mrs: []forge.MergeRequest{{ID: 2, Branch: "workline/developer/1"}}, verdicts: map[string]string{head: "red"}}
	prs, err := Read(dir, f, "workline/developer/", DefaultHours)
	if err != nil {
		t.Fatal(err)
	}
	if len(prs) != 1 || prs[0].ID != 2 || prs[0].Class != "red" || prs[0].Step != "reviewer" || prs[0].AgeSaid() != "3 days" {
		t.Fatalf("got %+v", prs)
	}
}

// The developer turns a red or capped pull request into a Draft: it is
// listed, and said a Draft.
func TestADraftIsListed(t *testing.T) {
	dir, head := clone(t)
	f := stub{mrs: []forge.MergeRequest{{ID: 2, Branch: "workline/developer/1", Draft: true}}, verdicts: map[string]string{head: "waits"}}
	prs, err := Read(dir, f, "workline/developer/", DefaultHours)
	if err != nil {
		t.Fatal(err)
	}
	if len(prs) != 1 || !prs[0].Draft {
		t.Fatalf("got %+v", prs)
	}
	if said := prs[0].Said(f); !strings.Contains(said, "a Draft") || !strings.Contains(said, "(waits)") {
		t.Errorf("said %q", said)
	}
}
