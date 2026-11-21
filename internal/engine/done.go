package engine

// What a merge leaves open (#999): once the developer's merge request
// closing an issue merges, the tester's Draft of that issue, its tests in
// the merge request built from it, has nothing left to merge, and the
// closed issue keeps none of the line's states. Follow tends both on each
// run — a push to the default branch, gardening's night —, so a Draft or
// an issue an earlier run missed is tended by the next.

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/kit/verdict"
)

// testerDraft is the branch prefix of the tester's Draft of an issue,
// workline/tester/issue-<n> (roles/tester/docs/hand-over.md).
const testerDraft = "workline/tester/issue-"

// settleDone closes the tester's Drafts whose issue is closed, and takes
// the states' labels off the closed issues bearing one; open, the
// branches of the roles' open merge requests. A forge that could not be
// read is said, a warning: the next run tends what this one could not.
func settleDone(f forge.Forge, repo, base string, open, states []string, res *Result) string {
	drafts, cleared := 0, 0
	warn := func(where string, err error) {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "done-not-tended", Level: "warn", Where: where,
			Message: "what a merge left open could not be tended here, the next run tries again: " + err.Error()})
	}
	for _, branch := range open {
		n, err := strconv.Atoi(strings.TrimPrefix(branch, testerDraft))
		if !strings.HasPrefix(branch, testerDraft) || err != nil || n <= 0 {
			continue
		}
		is, err := f.Issue(n)
		if err != nil {
			warn(branch, err)
			continue
		}
		if !is.Closed {
			continue
		}
		why := closedBy(f, *is, repo, base, branch)
		if err := f.CloseMergeRequest(branch, "Closed by workline: "+why+".", forge.Marker("closed")); err != nil {
			warn(branch, err)
			continue
		}
		drafts++
		res.Findings = append(res.Findings, verdict.Finding{Rule: "draft-closed", Where: branch, Message: why})
	}
	closed, err := closedBearing(f, states)
	if err != nil {
		warn("the closed issues", err)
	}
	for _, is := range closed {
		var off []string
		for _, l := range states {
			if slices.Contains(is.Labels, l) {
				off = append(off, l)
			}
		}
		if len(off) == 0 {
			continue
		}
		where := fmt.Sprintf("#%d", is.ID)
		if err := f.Label(forge.Target{Kind: "issue", ID: is.ID}, nil, off); err != nil {
			warn(where, err)
			continue
		}
		cleared++
		res.Findings = append(res.Findings, verdict.Finding{Rule: "state-cleared", Where: where,
			Message: "closed, it is off the line: " + strings.Join(off, ", ") + " taken off"})
	}
	return fmt.Sprintf("%d tester's Drafts of a closed issue closed, %d closed issues' states taken off", drafts, cleared)
}

// closedBy says why an issue's Draft is closed: the merge request that
// closed the issue, built from the Draft's tests, when the forge names
// one.
func closedBy(f forge.Forge, is forge.Issue, repo, base, branch string) string {
	why := fmt.Sprintf("#%d is closed", is.ID)
	if is.Reason == "not_planned" || is.Reason == "duplicate" {
		return why + " as " + strings.ReplaceAll(is.Reason, "_", " ") + ": its tests have nothing left to prove"
	}
	if b, ok := f.(forge.Backlog); ok {
		if closers, err := b.Closers(is.ID); err == nil {
			for _, c := range closers {
				if c.Kind == "pull-request" {
					return why + " by " + c.Ref + ", which brought its tests: nothing is left here to merge"
				}
			}
		}
	}
	if unmerged(repo, base, branch) {
		return why + ": its tests are not on " + base + " and stay on " + branch
	}
	return why + ": its tests have nothing left to merge here"
}

// unmerged tells whether the Draft's branch holds a file that base does
// not hold at all: its tests never reached base. A branch
// that cannot be read is said unmerged: nothing proves they did (#1321).
func unmerged(repo, base, branch string) bool {
	if repo == "" {
		return true
	}
	ref := "refs/heads/" + branch
	if _, err := git(repo, nil, "rev-parse", "--verify", "-q", ref); err != nil {
		ref = "refs/remotes/origin/" + branch
		if _, err := git(repo, nil, "rev-parse", "--verify", "-q", ref); err != nil {
			return true
		}
	}
	if base == "" {
		base = "main"
	}
	out, err := git(repo, nil, "diff", "-z", "--name-only", base+"..."+ref)
	if err != nil {
		return true
	}
	mb, _ := git(repo, nil, "merge-base", base, ref)
	mb = strings.TrimSpace(mb)
	for _, file := range strings.Split(out, "\x00") {
		if file == "" {
			continue
		}
		a, errA := git(repo, nil, "rev-parse", "--verify", "-q", ref+":"+file)
		if errA != nil {
			continue
		}
		b, errB := git(repo, nil, "rev-parse", "--verify", "-q", base+":"+file)
		if errB != nil {
			return true
		}
		if strings.TrimSpace(a) == strings.TrimSpace(b) {
			continue
		}
		// a file the Draft added, edited on base since, is reached; a file
		// that already existed is reached only by the Draft's own content
		if mb != "" {
			if _, err := git(repo, nil, "rev-parse", "--verify", "-q", mb+":"+file); err != nil {
				continue
			}
		}
		return true
	}
	return false
}

// changeOnBase tells whether the Draft's change to one file, from the
// merge-base, applies in reverse on base: its content is there.
func changeOnBase(repo, base, mb, ref, file string) bool {
	run := func(env []string, stdin []byte, args ...string) ([]byte, error) {
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		cmd.Env = append(os.Environ(), env...)
		if stdin != nil {
			cmd.Stdin = bytes.NewReader(stdin)
		}
		return cmd.Output()
	}
	patch, err := run(nil, nil, "diff", "--binary", mb, ref, "--", file)
	if err != nil || len(patch) == 0 {
		return false
	}
	dir, err := os.MkdirTemp("", "workline-unmerged-")
	if err != nil {
		return false
	}
	defer os.RemoveAll(dir)
	env := []string{"GIT_INDEX_FILE=" + filepath.Join(dir, "index")}
	if _, err := run(env, nil, "read-tree", base); err != nil {
		return false
	}
	_, err = run(env, patch, "apply", "-R", "--check", "--cached")
	return err == nil
}

// closedBearing is the closed issues bearing one of labels: a call a
// label where the forge lists them so, else every issue read once.
func closedBearing(f forge.Forge, labels []string) ([]forge.Issue, error) {
	if len(labels) == 0 {
		return nil, nil
	}
	if l, ok := f.(forge.Labelled); ok {
		var out []forge.Issue
		seen := map[int]bool{}
		for _, label := range labels {
			found, err := l.ClosedLabelled(label)
			if err != nil {
				return out, err
			}
			for _, is := range found {
				if !seen[is.ID] {
					seen[is.ID] = true
					out = append(out, is)
				}
			}
		}
		return out, nil
	}
	b, ok := f.(forge.Backlog)
	if !ok {
		return nil, errors.New("this forge lists no issues: the closed ones keep their states")
	}
	all, err := b.AllIssues()
	return slices.DeleteFunc(all, func(is forge.Issue) bool {
		return !is.Closed || !slices.ContainsFunc(labels, func(l string) bool { return slices.Contains(is.Labels, l) })
	}), err
}

// settleMerges closes the issues the merge requests merged outside the
// forge say they close, as a forge does on a merge (#1282): the local
// forge's, merged with git. Before settleDone, which takes their states
// off.
func settleMerges(s forge.Settler, res *Result) string {
	settled, err := s.SettleMerges()
	if err != nil {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "done-not-tended", Level: "warn", Where: "the merged merge requests",
			Message: "what a merge left open could not be tended here, the next run tries again: " + err.Error()})
	}
	closed := 0
	for _, m := range settled {
		for _, n := range m.Issues {
			closed++
			res.Findings = append(res.Findings, verdict.Finding{Rule: "issue-closed", Where: fmt.Sprintf("#%d", n),
				Message: fmt.Sprintf("!%d, which says it closes it, is merged into %s", m.MergeRequest, m.Base)})
		}
	}
	return fmt.Sprintf("%d merge requests merged with git, %d of the issues they close closed", len(settled), closed)
}
