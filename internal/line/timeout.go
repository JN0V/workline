package line

// A question times out (ADR-0060, decision 1): an issue in refine whose
// question to its reporter got no answer — or whose drafts no person
// accepted, all it lacks — for the product owner's delay (acts.ask.wait)
// wakes the product owner once, with no agent, to take its assumption.
// The delay is counted from the question's comment as the forge dates it
// (created_at), never from the issue's last update; the hourly reconcile
// tells it passed. A person's answer before it cancels it: the question
// is no longer unanswered.

import (
	"fmt"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/role"
)

// newTallyFor is a run's tally, the roles and the project it reads the
// product owner's delay from.
func newTallyFor(o engine.Options) *tally {
	t := newTally()
	t.rolesDir, t.repo = o.RolesDir, o.Repo
	return t
}

// askWait is the product owner's delay, as the project sets it over its
// level, read once a run; its default when the role does not load, or
// when the run says no roles folder (a test of the line alone).
func (t *tally) askWait() backlog.Delay {
	if t.wait != nil {
		return *t.wait
	}
	w := backlog.DefaultWait
	if t.rolesDir != "" {
		if r, err := role.Load(t.rolesDir, backlog.Keeper); err == nil {
			if pc, err := role.LoadProjectConfig(t.repo); err == nil {
				w = backlog.AskWait(r.MergedSettings(pc))
			}
		}
	}
	t.wait = &w
	return w
}

// timedOut says what of an issue in refine waits on a person past the
// product owner's delay: the key the record keeps, so it wakes the role
// once for it, and what the line says; "" when nothing does, or the
// record holds that key already.
func timedOut(b forge.Backlog, t *tally, is forge.Issue, notes []forge.Note, rec Record) (key, said string, err error) {
	w := t.askWait()
	if w.Never {
		return "", "", nil
	}
	now := clock().UTC()
	if s, due := backlog.SilenceDue(notes, backlog.Keeper, w, now); due {
		key = fmt.Sprintf("ask=%d", s.Round)
		said = fmt.Sprintf("its question of %s got no answer in %s (acts.ask.wait): the %s takes its assumption",
			s.Since.UTC().Format("2006-01-02 15:04 UTC"), w.Say(), backlog.Keeper)
	} else if st, _, _ := backlog.ReadState(forge.Bodies(notes), backlog.Keeper); !backlog.Readied(is) && backlog.OnlyDrafts(is, st) {
		passed := w.D == 0
		if !passed {
			trail, err := b.Trail(is.ID, backlog.LabelDraft)
			if err != nil {
				return "", "", err
			}
			since, known := backlog.DraftsSince(trail.Labeled, notes, is)
			passed = known && w.Passed(since, now)
		}
		if passed {
			key = "drafts=" + bodyHash(is.Body)
			said = fmt.Sprintf("its drafts got no answer in %s (acts.ask.wait): the %s takes them as written", w.Say(), backlog.Keeper)
		}
	}
	if key == "" || key == rec.Timeout {
		return "", "", nil
	}
	return key, said, nil
}
