package forge

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// fakeBot is who the simulated forge says turned auto-merge on for
// workline: its token's user.
const fakeBot = "workline-bot"

// mergeToken: as the file says, a job's own token or not.
func (f *Fake) mergeToken() string {
	if s, err := f.load(); err == nil && s.JobToken {
		return "the job's own token: a merge it makes starts no pipeline on the base"
	}
	return ""
}

// MergeState reads a merge request's state as the file holds it, its head
// and its pushes' heads resolved in the repository: the branch's tip on
// origin, a fork's under refs/pull/<id>/head.
func (f *Fake) MergeState(mr MergeRequest) (MergeState, error) {
	s, err := f.load()
	if err != nil {
		return MergeState{}, err
	}
	it, err := s.item(Target{Kind: "merge-request", ID: mr.ID})
	if err != nil {
		return MergeState{}, err
	}
	ref := "refs/heads/" + it.Branch
	if it.Fork {
		ref = fmt.Sprintf("refs/pull/%d/head", it.ID)
	}
	out, _ := exec.Command("git", "-C", f.Repo, "ls-remote", "origin", ref).Output()
	head, _, _ := strings.Cut(string(out), "\t")
	st := MergeState{Head: strings.TrimSpace(head), Draft: it.Draft, AutoMerge: it.AutoMerge, AutoMergeAt: it.AutoMergeAt,
		Checks: it.Checks, Labels: it.LabelEvents}
	if it.AutoMerge {
		st.AutoMergeBy = it.AutoMergeBy // an app's is named <app>[bot], as on GitHub
		st.AutoMergeBot = it.AutoMergeBy == fakeBot || strings.HasSuffix(it.AutoMergeBy, "[bot]")
	}
	for _, p := range it.Pushes {
		h, err := exec.Command("git", "-C", f.Repo, "rev-parse", "--verify", "-q", p.Head+"^{commit}").Output()
		if err != nil {
			return MergeState{}, fmt.Errorf("the simulated forge's push %q is no revision of the repository", p.Head)
		}
		st.Pushes = append(st.Pushes, Push{Head: strings.TrimSpace(string(h)), At: p.At})
		st.Pushers = append(st.Pushers, p.By)
	}
	return st, nil
}

// Guarded: as the file says (`unguarded`).
func (f *Fake) Guarded(base string) (string, error) {
	s, err := f.load()
	if err != nil {
		return "", err
	}
	return s.Unguarded, nil
}

// resolve is a revision of the repository as a commit, "" when it is none.
func (f *Fake) resolve(rev string) string {
	out, err := exec.Command("git", "-C", f.Repo, "rev-parse", "--verify", "-q", rev+"^{commit}").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// ownStatus is the last status of that name on head set by workline's
// token's user; one another set is never read.
func (f *Fake) ownStatus(head, name string) (FakeStatus, bool, error) {
	s, err := f.load()
	if err != nil {
		return FakeStatus{}, false, err
	}
	for i := len(s.Statuses) - 1; i >= 0; i-- {
		if st := s.Statuses[i]; st.Name == name && st.By == fakeBot && f.resolve(st.Commit) == head {
			return st, true, nil
		}
	}
	return FakeStatus{}, false, nil
}

func (f *Fake) setStatus(head, name, state, description string) error {
	return f.write(func(s *FakeState) error {
		s.Statuses = append(s.Statuses, FakeStatus{Commit: head, Name: name, State: state, Description: statusText(description), By: fakeBot})
		return nil
	})
}

// KeepVerdict keeps the verdict as GitHub does, a status by the token.
func (f *Fake) KeepVerdict(head, class, why string) error {
	return f.setStatus(head, VerdictStatus, map[bool]string{true: "success", false: "failure"}[class == "pass"], verdictText(class, why))
}

// Verdict reads back the verdict the token kept on head.
func (f *Fake) Verdict(head string) (string, string, error) {
	st, ok, err := f.ownStatus(head, VerdictStatus)
	if err != nil || !ok {
		return "", "", err
	}
	class, why := readVerdict(st.Description)
	return class, why, nil
}

// Accept sets workline/accepted on head, as GitHub does.
func (f *Fake) Accept(head string, ok bool, why string) error {
	return f.setStatus(head, Accepted, map[bool]string{true: "success", false: "failure"}[ok], why)
}

// AutoMerge turns auto-merge on, now, as workline's token's user, or off;
// one Clean is merged at once instead. A draft is refused, as GitHub
// refuses it.
func (f *Fake) AutoMerge(mr MergeRequest, head string, on bool) (bool, error) {
	merged := false
	err := f.write(func(s *FakeState) error {
		it, err := s.item(Target{Kind: "merge-request", ID: mr.ID})
		if err != nil {
			return err
		}
		switch {
		case !on:
			it.AutoMerge, it.AutoMergeBy = false, ""
		case it.Draft:
			return fmt.Errorf("%w: the simulated forge turns no auto-merge on for a draft", errRefused)
		case it.Clean:
			it.Closed, it.Merged, merged = true, true, true
		default:
			it.AutoMerge, it.AutoMergeBy, it.AutoMergeAt = true, fakeBot, time.Now().UTC().Format(time.RFC3339)
		}
		return nil
	})
	return merged, err
}
