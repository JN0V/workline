package simulator

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/JN0V/workline/internal/forge"
)

// fixture is the C++ project the traces run on: the trial's Scope paths,
// small, each component's native suites in test/test_*/, built by g++ and
// run by tools/workline-tests.sh, as the trial copy's script ran
// PlatformIO's.
const fixture = "fixture"

// gitEnv is git with no machine's config, a fixed author and date.
var gitEnv = []string{"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1",
	"GIT_AUTHOR_NAME=Fixture", "GIT_AUTHOR_EMAIL=fixture@example.invalid",
	"GIT_COMMITTER_NAME=Fixture", "GIT_COMMITTER_EMAIL=fixture@example.invalid",
	"GIT_AUTHOR_DATE=2026-01-01T00:00:00Z", "GIT_COMMITTER_DATE=2026-01-01T00:00:00Z"}

func gitIn(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir, cmd.Env = dir, append(os.Environ(), gitEnv...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git %v: %v\n%s", args, err, out)
	}
	return nil
}

// fixtureRepo makes repo from the fixture and the line's config, one
// commit on main, and its bare remote beside it, origin.git; a copy of
// both names its own (newWorld).
func fixtureRepo(repo string) error {
	origin := filepath.Join(filepath.Dir(repo), "origin.git")
	src, err := filepath.Abs(fixture)
	if err != nil {
		return err
	}
	if out, err := exec.Command("cp", "-a", src, repo).CombinedOutput(); err != nil {
		return fmt.Errorf("the fixture: %v\n%s", err, out)
	}
	if err := os.MkdirAll(filepath.Join(repo, ".workline"), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(repo, ".workline", "config.yaml"), []byte(config), 0o644); err != nil {
		return err
	}
	for _, args := range [][]string{
		{"init", "-q", "-b", "main", "."},
		{"add", "."},
		{"commit", "-q", "--no-verify", "-m", "feat: the components"},
		{"init", "-q", "--bare", "-b", "main", origin},
		{"remote", "add", "origin", origin},
		{"push", "-q", "--no-verify", "origin", "main"},
		{"branch", "-q", "--set-upstream-to=origin/main"},
	} {
		if err := gitIn(repo, args...); err != nil {
			return err
		}
	}
	return nil
}

// trialBranches puts in origin each branch the trace's open pull
// requests name, from main: their code is not in the trace, their
// branches are, as the roles read them.
func (w *world) trialBranches(t *testing.T) {
	for _, mr := range w.state(t).MergeRequests {
		if mr.Closed || mr.Branch == "" {
			continue
		}
		if err := gitIn(w.repo, "push", "-q", "--no-verify", "origin", "main:refs/heads/"+mr.Branch); err != nil {
			t.Fatal(err)
		}
	}
}

var closes = regexp.MustCompile(`(?im)^(?:closes|fixes|resolves) #(\d+)`)

// merge is the forge and the person on it, between two ticks: each open
// pull request out of draft that closes an issue (`Closes #n`) is merged
// into main, its branch from origin, as the merge policy would once its
// checks are green, and the issue closed, as the forge closes it. One
// whose issue is blocked waits: its round is not over. It returns what
// it did, for the log.
func (w *world) merge(t *testing.T, at time.Time) []string {
	s := w.state(t)
	var did []string
	for i := range s.MergeRequests {
		mr := &s.MergeRequests[i]
		closing := closes.FindAllStringSubmatch(mr.Body, -1)
		if mr.Closed || mr.Draft || mr.Branch == "" || len(closing) == 0 || slices.ContainsFunc(closing, func(m []string) bool {
			id, _ := strconv.Atoi(m[1])
			is := issueOf(s, id)
			return is != nil && slices.Contains(is.Labels, "workline:blocked")
		}) {
			continue
		}
		for _, args := range [][]string{
			{"fetch", "-q", "origin", "refs/heads/" + mr.Branch},
			{"checkout", "-q", "main"},
			{"merge", "-q", "--no-ff", "--no-verify", "-m", fmt.Sprintf("Merge !%d", mr.ID), "FETCH_HEAD"},
			{"push", "-q", "--no-verify", "origin", "main"},
		} {
			if err := gitIn(w.repo, args...); err != nil {
				t.Fatalf("the forge's merge of !%d: %v", mr.ID, err)
			}
		}
		sha, err := exec.Command("git", "-C", w.repo, "rev-parse", "HEAD").Output()
		if err != nil {
			t.Fatal(err)
		}
		mr.Closed, mr.Merged, mr.MergedAt, mr.MergedAs = true, true, at.Format(time.RFC3339), strings.TrimSpace(string(sha))
		did = append(did, fmt.Sprintf("!%d merged", mr.ID))
		for _, m := range closing {
			id, _ := strconv.Atoi(m[1])
			if is := issueOf(s, id); is != nil && !is.Closed {
				is.Closed, is.Reason = true, "completed"
				is.ClosedBy = append(is.ClosedBy, forge.Closer{Kind: "pull-request", Ref: fmt.Sprintf("#%d", mr.ID), Text: mr.Title})
				did = append(did, fmt.Sprintf("#%d closed", id))
			}
		}
	}
	if len(did) > 0 {
		data, err := json.Marshal(s)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(w.forge, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return did
}
