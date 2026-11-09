package line

// The per-issue backstop (ADR-0064, §5): the fresh tokens every run on an
// issue spent, all roles, the runs on its pull request included
// (Record.Spent), against backstopTimes its normal, its tier left out: a
// `large` the product owner granted never raises it. A person's `x-large`
// lifts it to backstopXLarge times. At it, the issue waits for a person:
// no role runs on it, the product owner is not asked, and no delay takes
// an answer for the person. Checked before a step runs, and after, so a
// run that takes the issue past it blocks it at once. Each part of a
// split is an issue of its own, its record its own: its own backstop.
// The liveness caps stay as they are: max-hops, the turns, the seconds,
// the machine-error budget.

import (
	"cmp"
	"fmt"
	"strings"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/developer"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/verdict"
)

const (
	// backstopTimes: an issue's backstop, in its normals; backstopXLarge,
	// at a person's `x-large` (4× the x-large normal).
	backstopTimes  = 4
	backstopXLarge = 16
)

// backstop is an issue's: Tokens, its fresh tokens in all; Normal, the
// normal it is counted from, Measured or the seed; Tier, the issue's.
type backstop struct {
	Tokens, Normal int
	Measured       bool
	Tier           string
}

// backstopOf is issue is's backstop: its normal (baseNormal), from the
// normals kept on the forge among open, times backstopTimes, or
// backstopXLarge at `x-large`.
func backstopOf(b forge.Backlog, open []forge.Issue, is forge.Issue) backstop {
	n, measured := baseNormal(b, open, is)
	bs := backstop{Normal: n, Measured: measured, Tier: backlog.SizeTier(is.Labels)}
	bs.Tokens = n * backstopTimes
	if bs.Tier == backlog.TierXLarge {
		bs.Tokens = n * backstopXLarge
	}
	return bs
}

// reason is why an issue that spent spent waits at bs, as its block
// keeps it; said, with what a person does about it.
func (bs backstop) reason(spent int) string {
	normal := Tokens(bs.Normal) + " (seed)"
	if bs.Measured {
		normal = Tokens(bs.Normal) + " (p85)"
	}
	if bs.Tier == backlog.TierXLarge {
		return fmt.Sprintf("%s fresh tokens spent on it, all runs, reaching its backstop of %s, %d× its normal of %s at the top size tier (`%s`)",
			Tokens(spent), Tokens(bs.Tokens), backstopXLarge, normal, backlog.LabelSizeXLarge)
	}
	large := ""
	if bs.Tier == backlog.TierLarge {
		large = fmt.Sprintf("; `%s` never raises it", backlog.LabelSizeLarge)
	}
	return fmt.Sprintf("%s fresh tokens spent on it, all runs, reaching its backstop of %s, %d× its normal of %s%s",
		Tokens(spent), Tokens(bs.Tokens), backstopTimes, normal, large)
}

func (bs backstop) said(spent int) string {
	return bs.reason(spent) + ": " + personDoes(bs.Tier == backlog.TierXLarge, bs.Normal*backstopXLarge)
}

// personDoes is what a person does on an issue at its backstop: at the
// top tier, splits it or finishes it; below, gives it `x-large`, which
// lifts the backstop to lifted, or splits it.
func personDoes(top bool, lifted int) string {
	if top {
		return "a person splits it into new issues, each with its own backstop, or finishes it by hand"
	}
	return fmt.Sprintf("a person gives it `%s`, which lifts the backstop to %s, and moves it back by its label; or splits it into new issues, each with its own backstop",
		backlog.LabelSizeXLarge, Tokens(lifted))
}

// backstopped is st, the step due on issue is, unless its runs reached
// the issue's backstop: then no step, and what the line writes instead —
// the issue blocked for a person; one blocked already, the product
// owner not woken on it, its record kept. st keeps how to read the
// backstop for the check after its run (passed): an issue with nothing
// spent reads nothing more before it.
func backstopped(res *Result, b forge.Backlog, is forge.Issue, open *[]forge.Issue, t *tally, st *stateStep) (*stateStep, *StateWrite, error) {
	var read *backstop
	st.backstop = func() (backstop, error) {
		if read != nil {
			return *read, nil
		}
		if *open == nil {
			issues, err := b.Issues()
			if err != nil {
				return backstop{}, err
			}
			*open = issues
		}
		if !t.effort {
			// The issues' effort, read as the product owner reads it: a
			// scheme that does not read is said, the seed not doubled.
			t.effort = true
			if note := effortScheme(t.repo, b, *open); note != nil {
				res.Findings = append(res.Findings, *note)
			}
		}
		bs := backstopOf(b, *open, is)
		read = &bs
		return bs, nil
	}
	spent := st.write.Record.Fresh()
	if spent == 0 {
		return st, nil, nil
	}
	bs, err := st.backstop()
	if err != nil {
		return nil, nil, err
	}
	if bs.Tokens == 0 || spent < bs.Tokens {
		return st, nil, nil
	}
	where, why, said := fmt.Sprintf("#%d", is.ID), bs.reason(spent), bs.said(spent)
	if st.write.Record.State == BlockedState {
		// Blocked already: the product owner is not woken on it, no
		// delay answers for the person (ADR-0064, §5). Said, with no
		// write: a quiet line stays quiet. Its record is written once,
		// when it does not say its backstop yet, or holds a size
		// question for the product owner.
		said = "blocked, past its backstop: it waits for a person, the " + roleSaid(backlog.Keeper) + " not woken; " + said
		res.Findings = append(res.Findings, verdict.Finding{Rule: "backstop-waits", Level: "info", Where: where, Message: said})
		if st.rec.SizeOf == "" && st.write.Record.Backstop == bs.Tokens {
			return nil, nil, nil
		}
		rec := st.write.Record
		rec.SizeOf, rec.Backstop, rec.Why = "", bs.Tokens, why
		return nil, &StateWrite{Issue: is.ID, Record: rec, Seen: said}, nil
	}
	comments, err := b.Comments(forge.Target{Kind: "issue", ID: is.ID})
	if err != nil {
		return nil, nil, err
	}
	res.Findings = append(res.Findings, verdict.Finding{Rule: "backstop-crossed", Level: "warn", Where: where, Message: said + ": moved to " + BlockedState})
	rec := blocking(st.write.Record, st.from.Name, why, comments)
	rec.Backstop = bs.Tokens
	return nil, &StateWrite{Issue: is.ID, Record: rec, Blocked: said}, nil
}

// passed blocks st's issue for a person when its run took it to its
// backstop (backstopped kept it): over whatever its run's outcome asked,
// the product owner on its size first, and wherever its role moved it on
// the line — a state that wakes no role, as the developer's pull request
// waiting, would leave it there unsaid. Its role's work is kept: a pull
// request it opened stays open. A closed issue is left closed.
func passed(res *Result, f forge.Forge, cfg *routing.Config, st *stateStep) {
	spent := st.write.Record.Fresh()
	if st.backstop == nil || spent == 0 {
		return
	}
	bs, err := st.backstop()
	if err != nil {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "backstop-unread", Level: "warn", Where: fmt.Sprintf("#%d", st.write.Issue),
			Message: "its backstop was not read after its run: its next step's check reads it again, " + forge.Said(err)})
		return
	}
	if bs.Tokens == 0 || spent < bs.Tokens {
		return
	}
	is, err := f.Issue(st.write.Issue)
	if err != nil {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "backstop-unread", Level: "warn", Where: fmt.Sprintf("#%d", st.write.Issue),
			Message: "its state was not read after its run took it to its backstop: its next step's check blocks it, " + forge.Said(err)})
		return
	}
	if _, ok := cfg.StateOf(is.Labels); !ok || is.Closed {
		return // closed, or taken off the line: no step runs on it
	}
	var comments []string
	if b, ok := f.(forge.Backlog); ok {
		comments, _ = b.Comments(forge.Target{Kind: "issue", ID: st.write.Issue})
	}
	res.Findings = append(res.Findings, verdict.Finding{Rule: "backstop-crossed", Level: "warn", Where: fmt.Sprintf("#%d", st.write.Issue),
		Message: bs.said(spent) + ": moved to " + BlockedState})
	from := cmp.Or(st.from.Name, st.write.Record.From)
	st.write.Record = blocking(st.write.Record, from, bs.reason(spent), comments)
	st.write.Record.SizeOf, st.write.Record.Backstop = "", bs.Tokens
	st.write.Blocked, st.write.Failed = bs.said(spent), false
}

// issueBackstop reads issue id's record, its comments and its backstop,
// as a step reads them, for a run outside the line's steps: a review
// round on its pull request. spent: its runs' fresh tokens so far.
func issueBackstop(f forge.Forge, b forge.Backlog, repo string, id int) (is *forge.Issue, rec Record, comments []string, bs backstop, spent int, err error) {
	if is, err = f.Issue(id); err != nil {
		return nil, Record{}, nil, backstop{}, 0, err
	}
	if comments, err = b.Comments(forge.Target{Kind: "issue", ID: id}); err != nil {
		return nil, Record{}, nil, backstop{}, 0, err
	}
	rec, _ = recordIn(comments)
	if spent = rec.Fresh(); spent == 0 {
		return is, rec, comments, backstop{}, 0, nil
	}
	open, err := b.Issues()
	if err != nil {
		return nil, Record{}, nil, backstop{}, 0, err
	}
	effortScheme(repo, b, open) // a scheme not read leaves the seed undoubled, said by the line's own steps
	return is, rec, comments, backstopOf(b, open, *is), spent, nil
}

// BackstopSaid says why issue id's runs reached its backstop, and what
// a person does about it; "" when they did not, or when f lists no
// comments. The developer's review round reads it (developer.Backstop).
func BackstopSaid(f forge.Forge, repo string, id int) (string, error) {
	b, ok := f.(forge.Backlog)
	if !ok {
		return "", nil
	}
	_, _, _, bs, spent, err := issueBackstop(f, b, repo, id)
	if err != nil || bs.Tokens == 0 || spent < bs.Tokens {
		return "", err
	}
	return bs.said(spent), nil
}

// SpentSaid says what issue is spent so far, its record read from
// comments, against its normal (IssueNormal, its tier in) and its
// backstop: the facts the product owner asked on its size judges on
// (#1193, round 6's RC3), never "too big" alone.
func SpentSaid(b forge.Backlog, open []forge.Issue, is forge.Issue, comments []string) string {
	rec, _ := recordIn(comments)
	n, measured := IssueNormal(b, open, is)
	from := "the seed, until about 10 issues of its kind are done here"
	if measured {
		from = "this project's own, its 85th percentile"
	}
	if tier := backlog.SizeTier(is.Labels); tier != "" {
		from += fmt.Sprintf(", ×%d its tier `%s`", backlog.TierFactor(tier), backlog.TierLabel(tier))
	}
	return fmt.Sprintf("%s fresh tokens, all runs, against its normal of %s an issue of its kind (%s), and its backstop of %s, where it waits for a person",
		Tokens(rec.Fresh()), Tokens(n), from, Tokens(backstopOf(b, open, is).Tokens))
}

// roundStopped says whether the review round due on mr is stopped, its
// issue's runs at their backstop (ADR-0064, §5, the panel of
// 2026-11-08): said on the pull request, kept open, and the write that
// blocks its issue, nil when the line blocked it at its backstop already.
// One not read is said, and runs: the round's own check reads it again.
func roundStopped(res *Result, f forge.Forge, cfg *routing.Config, repo string, mr forge.MergeRequest) (bool, *StateWrite) {
	id := developer.BranchIssue(mr.Branch)
	b, ok := f.(forge.Backlog)
	if id == 0 || !ok {
		return false, nil
	}
	where := fmt.Sprintf("merge request %d", mr.ID)
	is, rec, comments, bs, spent, err := issueBackstop(f, b, repo, id)
	if err != nil {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "backstop-unread", Level: "warn", Where: where,
			Message: fmt.Sprintf("#%d's backstop was not read: its round's own check reads it again, %s", id, forge.Said(err))})
		return false, nil
	}
	if bs.Tokens == 0 || spent < bs.Tokens {
		return false, nil
	}
	res.Findings = append(res.Findings, verdict.Finding{Rule: "backstop-crossed", Level: "warn", Where: where,
		Message: fmt.Sprintf("%s: no review round runs; #%d moved to %s, this pull request kept open", bs.said(spent), id, BlockedState)})
	if rec.State == BlockedState && rec.Backstop == bs.Tokens {
		return true, nil
	}
	from := rec.State
	if s, ok := cfg.StateOf(is.Labels); ok {
		from = s.Name
	}
	r := blocking(rec, from, bs.reason(spent), comments)
	r.Backstop = bs.Tokens
	return true, &StateWrite{Issue: id, Record: r, Blocked: bs.said(spent)}
}

// backstopNext is what a person does on an issue blocked at its
// backstop, as the line's comment says it.
func backstopNext(r Record) string {
	top := strings.Contains(r.Why, "at the top size tier")
	return personDoes(top, r.Backstop/backstopTimes*backstopXLarge) + "; no product owner reads it again, and no delay answers for a person"
}
