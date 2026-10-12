package forge

import (
	"fmt"
	"os/exec"
	"strings"
)

// Review keeps the comments as the merge request's threads, in one write,
// whole or not at all as GitHub's review; a line the merge request's diff
// does not show is left out. The diff is the repository's, from the merge
// request's base (main unless it names one) to head, as a forge computes
// it: renames found, three lines of context.
func (f *Fake) Review(id int, head, body string, comments []LineComment) (Reviewed, error) {
	return review(fakeLines{f, head}, id, head, body, comments)
}

// fakeLines is the simulated forge seen by a review at head.
type fakeLines struct {
	f    *Fake
	head string
}

func (l fakeLines) shownLines(id int) (shown, error) {
	s, err := l.f.load()
	if err != nil {
		return shown{}, err
	}
	it, err := s.item(Target{Kind: "merge-request", ID: id})
	if err != nil {
		return shown{}, err
	}
	base, head := it.Base, l.head
	if base == "" {
		base = "main"
	}
	if head == "" {
		head = "HEAD"
	}
	cmd := exec.Command("git", "diff", "--no-color", "--no-ext-diff", "-M", "-U3", base+"..."+head)
	cmd.Dir = l.f.Repo
	out, err := cmd.Output()
	if err != nil {
		return shown{}, fmt.Errorf("%w: the simulated forge reads no diff from %s to %s: %v", ErrUnreachable, base, head, err)
	}
	sh := newShown()
	sh.addGit(string(out))
	return sh, nil
}

func (l fakeLines) openKeys(id int) (map[string]bool, error) {
	s, err := l.f.load()
	if err != nil {
		return nil, err
	}
	it, err := s.item(Target{Kind: "merge-request", ID: id})
	if err != nil {
		return nil, err
	}
	open := map[string]bool{}
	for _, t := range it.Threads {
		if !t.Resolved {
			for _, k := range keysIn(t.Body) {
				open[k] = true
			}
		}
	}
	return open, nil
}

func (l fakeLines) postLines(id int, _, _ string, _ shown, comments []LineComment) (string, []string, error) {
	if s, err := l.f.load(); err != nil || s.ReviewRefused != "" {
		if err == nil {
			err = fmt.Errorf("%w: %s", errForbidden, s.ReviewRefused)
		}
		return "", nil, err
	}
	n := 0
	err := l.f.write(func(s *FakeState) error {
		it, err := s.item(Target{Kind: "merge-request", ID: id})
		if err != nil {
			return err
		}
		for _, c := range comments {
			it.Threads = append(it.Threads, FakeThread{Path: c.Path, Line: c.Line, Body: c.withKey()})
		}
		s.Reviews++
		n = s.Reviews
		return nil
	})
	if err != nil {
		return "", nil, err
	}
	return fmt.Sprintf("%s#review-%d", strings.TrimSuffix(l.changesPage(id), "/changes"), n), nil, nil
}

func (l fakeLines) changesPage(id int) string {
	return fmt.Sprintf("https://forge.example/merge-requests/%d/changes", id)
}
