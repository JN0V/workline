// Package stall reads the line's pull requests that stand still: a
// developer's open pull request whose head has not moved for the age
// limit (routing.stalled-hours) and whose kept verdict is not a pass
// (ADR-0068 §8). It reads git and the forge, never an agent: the wake
// and the reconcile both say it.
package stall

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	wlclock "github.com/JN0V/workline/internal/clock"
	"github.com/JN0V/workline/internal/forge"
)

// DefaultHours is the age limit when the project sets none.
const DefaultHours = 12

// PR is one stalled pull request.
type PR struct {
	ID    int
	Age   time.Duration // since its head was made
	Step  string        // the step of the verdict kept on its head; "" when it names none
	Class string        // the verdict's class: red, held…
}

// Ref is the pull request as the forge names it.
func (p PR) Ref(f forge.Forge) string { return forge.MergeRef(f, p.ID) }

// AgeSaid is its age: hours under two days, days after.
func (p PR) AgeSaid() string {
	h := int(p.Age.Hours())
	if h >= 48 {
		return fmt.Sprintf("%d days", h/24)
	}
	return fmt.Sprintf("%d h", h)
}

// Said is "!2: 12 h, step reviewer, red".
func (p PR) Said(f forge.Forge) string {
	s := p.Ref(f) + ": head unmoved for " + p.AgeSaid()
	if p.Step != "" {
		s += ", verdict at the " + p.Step + "'s step"
	}
	return s + " (" + p.Class + ")"
}

var step = regexp.MustCompile(`\(([a-z][a-z0-9-]*) [a-z-]+\)`)

// Read lists the stalled pull requests, by number. branch is the prefix
// of the developer's branches; hours the age limit, 0 for none. A forge
// that keeps no verdict, or does not answer, lists none.
func Read(repo string, f forge.Forge, branch string, hours int) ([]PR, error) {
	m, ok := f.(forge.Merger)
	if !ok || hours <= 0 {
		return nil, nil
	}
	mrs, err := f.ListMergeRequests()
	if err != nil {
		return nil, err
	}
	var out []PR
	for _, mr := range mrs {
		if !strings.HasPrefix(mr.Branch, branch) || mr.Draft {
			continue
		}
		head, at := tip(repo, mr.Branch)
		if head == "" {
			continue
		}
		age := wlclock.Now().Sub(at)
		if age < time.Duration(hours)*time.Hour {
			continue
		}
		class, why, err := m.Verdict(head)
		if err != nil || class == "" || class == "pass" {
			continue
		}
		p := PR{ID: mr.ID, Age: age, Class: class}
		if g := step.FindStringSubmatch(why); g != nil {
			p.Step = g[1]
		}
		out = append(out, p)
	}
	return out, nil
}

func tip(repo, branch string) (string, time.Time) {
	// A checkout without local branches holds the remote-tracking ones.
	var out []byte
	var err error
	for _, ref := range []string{"refs/heads/", "refs/remotes/origin/"} {
		out, err = exec.Command("git", "-C", repo, "log", "-1", "--format=%H %cI", ref+branch).Output()
		if err == nil {
			break
		}
	}
	if err != nil {
		return "", time.Time{}
	}
	h, d, _ := strings.Cut(strings.TrimSpace(string(out)), " ")
	t, err := time.Parse(time.RFC3339, d)
	if err != nil {
		return "", time.Time{}
	}
	return h, t
}
