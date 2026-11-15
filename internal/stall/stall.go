// Package stall reads the line's pull requests that stand still: a
// developer's open pull request whose head has not moved for the age
// limit (routing.stalled-hours) and whose kept verdict is not a pass
// (ADR-0068 §8). A Draft counts: the developer turns a red or capped
// pull request into one. It reads git and the forge, never an agent: the
// wake and the reconcile both say it.
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
	Class string        // the verdict's class: red, held, waits…
	Draft bool          // a Draft: red, capped or its agent failed
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

// Said is "!2: head unmoved for 12 h, verdict at the reviewer's step (red)".
func (p PR) Said(f forge.Forge) string {
	s := p.Ref(f)
	if p.Draft {
		s += ", a Draft"
	}
	s += ": head unmoved for " + p.AgeSaid()
	if p.Step != "" {
		s += ", verdict at the " + p.Step + "'s step"
	}
	return s + " (" + p.Class + ")"
}

var step = regexp.MustCompile(`\(([a-z][a-z0-9-]*) [a-z-]+\)`)

// Read lists the stalled pull requests, by number. branch is the prefix
// of the developer's branches; hours the age limit, 0 for none. A forge
// that keeps no verdict lists none. A verdict the forge does not give is
// said in notRead, one line a reason: "the verdict on #3, #5: …", never
// skipped in silence.
func Read(repo string, f forge.Forge, branch string, hours int) (prs []PR, notRead []string, err error) {
	m, ok := f.(forge.Merger)
	if !ok || hours <= 0 {
		return nil, nil, nil
	}
	mrs, err := f.ListMergeRequests()
	if err != nil {
		return nil, nil, err
	}
	var out []PR
	var reasons []string
	missed := map[string][]string{}
	for _, mr := range mrs {
		if !strings.HasPrefix(mr.Branch, branch) {
			continue
		}
		head, at := tip(repo, f, mr.Branch)
		if head == "" {
			continue
		}
		age := wlclock.Now().Sub(at)
		if age < time.Duration(hours)*time.Hour {
			continue
		}
		class, why, err := m.Verdict(head)
		if err != nil {
			r := forge.Said(err)
			if _, seen := missed[r]; !seen {
				reasons = append(reasons, r)
			}
			missed[r] = append(missed[r], forge.MergeRef(f, mr.ID))
			continue
		}
		if class == "" || class == "pass" {
			continue
		}
		p := PR{ID: mr.ID, Age: age, Class: class, Draft: mr.Draft}
		if g := step.FindStringSubmatch(why); g != nil {
			p.Step = g[1]
		}
		out = append(out, p)
	}
	for _, r := range reasons {
		what := "the verdict on "
		if len(missed[r]) > 1 {
			what = "the verdicts on "
		}
		notRead = append(notRead, what+strings.Join(missed[r], ", ")+": "+r)
	}
	return out, notRead, nil
}

// tip is the branch's head and its commit date: as fetched from origin
// (a CI checkout keeps the developer's branches there), else this clone's
// own branch; only the latter on a forge that keeps branches here (the
// local one).
func tip(repo string, f forge.Forge, branch string) (string, time.Time) {
	refs := []string{"refs/remotes/origin/" + branch, "refs/heads/" + branch}
	if forge.KeepsBranches(f) {
		refs = refs[1:]
	}
	for _, ref := range refs {
		out, err := exec.Command("git", "-C", repo, "log", "-1", "--format=%H %cI", ref, "--").Output()
		if err != nil {
			continue
		}
		h, d, _ := strings.Cut(strings.TrimSpace(string(out)), " ")
		t, err := time.Parse(time.RFC3339, d)
		if err != nil || h == "" {
			continue
		}
		return h, t
	}
	return "", time.Time{}
}
