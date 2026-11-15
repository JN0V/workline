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
	"time"

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
// record holds that key already. waits: what the line says of a wait on
// a person within the delay, or with none (acts.ask.wait: never), the
// day it times out (#1082): never silent while it waits; "" when
// nothing waits on a person, or the delay passed. next: what of a
// person ends that wait, said with it on the issue (#1195).
func timedOut(b forge.Backlog, t *tally, is forge.Issue, notes []forge.Note, rec Record) (key, said, waits, next string, err error) {
	w := t.askWait()
	now := clock().UTC()
	s, open := backlog.Unanswered(notes, backlog.Keeper, is)
	st, _, _ := backlog.ReadState(forge.Bodies(notes), backlog.Keeper)
	if open && (st == nil || st.Assumed < s.Round) && !s.Due(w, st, now) {
		waits = questionWaits(w, s, is.ID)
		next = "its reporter's answer in a comment wakes the " + roleSaid(backlog.Keeper)
	}
	switch {
	case open && s.Due(w, st, now):
		key = fmt.Sprintf("ask=%d", s.Round)
		said = fmt.Sprintf("its question of %s got no answer in %s (acts.ask.wait): the %s takes its assumption",
			day(s.Since), w.Say(), roleSaid(backlog.Keeper))
	case !backlog.Readied(is) && backlog.OnlyDrafts(is, st):
		at, since, known := w.D == 0 && !w.Never, time.Time{}, false
		if !at {
			trail, err := b.Trail(is.ID, backlog.LabelDraft)
			if err != nil {
				return "", "", "", "", err
			}
			since, known = backlog.DraftsSince(trail.Labeled, notes, is)
		}
		if at || known && w.Passed(since, now) {
			key = "drafts=" + bodyHash(is.Body)
			said = fmt.Sprintf("its drafts got no answer in %s (acts.ask.wait): the %s takes them as written", w.Say(), roleSaid(backlog.Keeper))
		} else if waits == "" {
			waits = draftsWait(w, since, known, is.ID)
			next = fmt.Sprintf("a person labels it `%s` to take them, or edits them, which wakes the %s",
				backlog.AcceptedLabel(b), roleSaid(backlog.Keeper))
		}
	}
	if key != "" && key == rec.Timeout {
		key, said = "", "" // woken once for it already
	}
	if key != "" {
		waits, next = "", "" // due, not waiting
	}
	return key, said, waits, next, nil
}

// draftsHeld says whether refine's step on issue, due on a new body or
// spec review (line.due), is due on its own drafts and their review
// alone while they wait on a person (#1127): the same state and
// labelling, all it lacks its drafts, the body the one the product
// owner last read or wrote (its state's digest: a person's edit since is
// news, even read by a night review), the reviewer's review of that body
// with nothing important open, no person's comment the product owner
// has not read. Then it could write nothing: the step waits out
// acts.ask.wait, or a person, with no agent call. Whether the delay
// passed is timedOut's.
//
// unreviewed: the same drafts, no review on this body yet, the review's
// rounds not spent (specWaits) — as when CI's judge job ran the reviewer
// before its apply wrote them (#1150). The reviewer's step, alone: the
// product owner could write nothing until it lands.
func draftsHeld(is forge.Issue, labeled string, comments []string, notes []forge.Note, rec Record, had bool) (held, unreviewed bool) {
	if !had || rec.State != refineState || rec.Labeled != labeled || backlog.Readied(is) {
		return false, false
	}
	st, _, err := backlog.ReadState(comments, backlog.Keeper)
	if err != nil || !backlog.OnlyDrafts(is, st) || st.Body != backlog.BodyDigest(is.Body) {
		return false, false
	}
	if counted := len(backlog.Counted(notes, is)); counted > backlog.Heard(st, counted) {
		return false, false // a person's comment the product owner has not read
	}
	if _, cleared := backlog.SpecCleared(is, comments); cleared {
		return true, false
	}
	return false, specWaits(is, comments)
}

// questionWaits is what the line says of a question s waiting on its
// reporter, the delay w not passed, or none: the day it times out.
func questionWaits(w backlog.Delay, s backlog.Silence, id int) string {
	switch {
	case w.Never && !s.Dated:
		return fmt.Sprintf("its question waits on its reporter, until it is answered (acts.ask.wait: %s)", backlog.WaitNever)
	case !s.Dated:
		return fmt.Sprintf("its question waits on #%d's reporter: the forge does not date it, so no assumption is taken until it is answered", id)
	case w.Never:
		return fmt.Sprintf("its question of %s waits on its reporter, until it is answered (acts.ask.wait: %s)", day(s.Since), backlog.WaitNever)
	}
	return fmt.Sprintf("its question of %s waits on its reporter: the %s takes its assumption on %s (acts.ask.wait: %s)",
		day(s.Since), roleSaid(backlog.Keeper), day(s.Since.Add(w.D)), w.Say())
}

// draftsWait is what the line says of drafts waiting on a person's yes
// since since (known), the delay w not passed, or none.
func draftsWait(w backlog.Delay, since time.Time, known bool, id int) string {
	switch {
	case w.Never && !known:
		return fmt.Sprintf("its drafts wait on a person's yes (acts.ask.wait: %s)", backlog.WaitNever)
	case !known:
		return fmt.Sprintf("its drafts wait on a person's yes on #%d: the forge does not say since when, so they are not taken as written until one is given", id)
	case w.Never:
		return fmt.Sprintf("its drafts, since %s, wait on a person's yes (acts.ask.wait: %s)", day(since), backlog.WaitNever)
	}
	return fmt.Sprintf("its drafts wait on a person's yes: the %s takes them as written on %s (acts.ask.wait: %s)",
		roleSaid(backlog.Keeper), day(since.Add(w.D)), w.Say())
}

// day is a moment as the line says it.
func day(t time.Time) string { return t.UTC().Format("2006-01-02 15:04 UTC") }
