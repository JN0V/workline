package developer

// What the developer writes on an issue it does not take now (ADR-0059,
// decision 1): a wait, or a claim it lets go, in its one comment there,
// the claim's, edited in place (principle 14). Never a job's log alone.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/board"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/verdict"
)

// saidFile holds the verdict of a run that only says, on the issues,
// what waits and what it let go: post writes it, after the engine
// applied the comments pre left (in/fallback.yaml).
const saidFile = "said.json"

// who names the developer in what it writes on an issue.
const who = "The developer"

// sayOn is the developer's comment on issue n: the claim's, one an issue.
func sayOn(n int, body string) intent.Intention {
	return intent.Intention{Kind: "comment", Value: map[string]any{"issue": n, "sticky": "claim", "body": body}}
}

// waits are what the pick writes on the issues it passed over for its own
// reasons: why each waits, and what comes next.
func waits(passed []Passed) []intent.Intention {
	var out []intent.Intention
	for _, p := range passed {
		if p.next != "" {
			out = append(out, sayOn(p.Issue, backlog.Said(who, "waits on this issue", p.Why, p.next)))
		}
	}
	return out
}

// stale lets go each claim older than claim-hours that no open merge
// request closes: its label off, why written in its comment, its card
// back to the board's ready column when there is one (card, ADR-0057;
// ADR-0059). A claim whose day the forge does not say is kept, and said
// in the verdict.
func stale(b forge.Backlog, issues []forge.Issue, mrs []forge.MergeRequest, s Settings, now time.Time, card *board.Move) ([]intent.Intention, []verdict.Finding, error) {
	closed := map[int]bool{}
	for _, m := range mrs {
		for _, n := range forge.ClosesOn(os.Getenv("WORKLINE_FORGE"), m.Title, m.Body) {
			closed[n] = true
		}
	}
	var out []intent.Intention
	var found []verdict.Finding
	for _, is := range issues {
		if !slices.Contains(is.Labels, s.ClaimLabel) || closed[is.ID] {
			continue
		}
		where := fmt.Sprintf("#%d", is.ID)
		t, err := b.Trail(is.ID, s.ClaimLabel)
		if errors.Is(err, forge.ErrUnreachable) {
			return nil, nil, err
		}
		at, ok := backlog.Moment(t.Labeled)
		if err != nil || !ok {
			found = append(found, verdict.Finding{Rule: "claim-age-unknown", Level: "warn", Where: where,
				Message: "the forge does not say when it got " + s.ClaimLabel + ": its claim is kept"})
			continue
		}
		hours := now.Sub(at).Hours()
		why := fmt.Sprintf("it took this issue on %s, and no pull request closes it %d hours later (claim-hours: %d)",
			backlog.Day(t.Labeled), int(hours), s.ClaimHours)
		if hours < float64(s.ClaimHours) {
			// Its agent wrote no code, and the issue's body changed since
			// the claim read it: a new input, let go at once (#1084).
			changed, err := bodyChanged(b, is)
			if errors.Is(err, forge.ErrUnreachable) {
				return nil, nil, err
			}
			if err != nil {
				found = append(found, verdict.Finding{Rule: "claim-body-unknown", Level: "warn", Where: where,
					Message: "its claim's comment could not be read (" + forge.Said(err) + "): kept until claim-hours"})
				continue
			}
			if !changed {
				continue
			}
			why = fmt.Sprintf("its agent wrote no code on this issue when it took it on %s, and its body changed since", backlog.Day(t.Labeled))
		} else {
			// Its agent wrote no code on the body it still has: let go, it
			// would be taken again on the same input. The line blocks it
			// for a person (claimwait.go, #1120): one place decides that.
			a, err := ClaimAct(b, is.ID)
			if errors.Is(err, forge.ErrUnreachable) {
				return nil, nil, err
			}
			if err != nil {
				found = append(found, verdict.Finding{Rule: "claim-body-unknown", Level: "warn", Where: where,
					Message: "its claim's comment could not be read (" + forge.Said(err) + "): kept, read again at the next run"})
				continue
			}
			if NoCodeOnSameBody(a, is.Body) {
				found = append(found, verdict.Finding{Rule: "claim-left-to-line", Level: "info", Where: where,
					Message: why + ", its agent wrote no code on the body it still has: kept, left to the line, which blocks it for a person"})
				continue
			}
		}
		out = append(out,
			intent.Intention{Kind: "label", Value: released(is, s, card)},
			sayOn(is.ID, backlog.Said(who, "lets this issue go", why,
				"it is free again: the developer may take it at a next run, or a person assigns themselves")))
		found = append(found, verdict.Finding{Rule: "claim-released", Level: "warn", Where: where,
			Message: why + ": " + s.ClaimLabel + " taken off, said on the issue"})
	}
	return out, found, nil
}

// bodyChanged says whether is's claim, its record in the developer's
// comment, wrote no code on a body other than the one it has now
// (#1084). A record from before Act.Body says nothing changed.
func bodyChanged(b forge.Backlog, is forge.Issue) (bool, error) {
	a, err := ClaimAct(b, is.ID)
	if err != nil {
		return false, err
	}
	return NoCodeOnOtherBody(a, is.Body), nil
}

// ClaimAct is the record the developer's claim comment on issue id
// holds, its last; the zero Act when none reads.
func ClaimAct(b forge.Backlog, id int) (Act, error) {
	comments, err := b.Comments(forge.Target{Kind: "issue", ID: id})
	if err != nil {
		return Act{}, err
	}
	return ClaimActIn(comments), nil
}

// ClaimActIn is the last developer's record among comments; the zero Act
// when none reads.
func ClaimActIn(comments []string) Act {
	var last Act
	for _, c := range comments {
		if a, ok := ReadAct(c); ok {
			last = a
		}
	}
	return last
}

// NoCodeOnOtherBody says whether a claim's record a wrote no code on a
// body other than body (#1084); false for a record from before Act.Body.
func NoCodeOnOtherBody(a Act, body string) bool {
	return a.NoCode && a.Body != "" && BodyChanged(a.Body, body)
}

// NoCodeOnSameBody says whether a claim's record a wrote no code on the
// body it has now, or on one it did not keep (a record from before
// Act.Body): run again, its agent would get the same input. Past
// claim-hours, the line blocks such an issue for a person instead.
func NoCodeOnSameBody(a Act, body string) bool {
	return a.NoCode && !NoCodeOnOtherBody(a, body)
}

// say ends pre with v: final when nothing is to be written on an issue;
// else the comments are left to the engine to apply, no agent asked, and
// post writes v.
func say(runDir string, v verdict.Verdict, said []intent.Intention) int {
	if len(said) == 0 {
		return final(runDir, v)
	}
	if err := intent.Write(filepath.Join(runDir, "in", "fallback.yaml"), said); err != nil {
		return fail(err)
	}
	if err := writeIn(runDir, saidFile, v); err != nil {
		return fail(err)
	}
	return 0
}

// saidPost writes the verdict pre left, the comments applied as they are.
func saidPost(runDir string) int {
	var v verdict.Verdict
	if err := readIn(runDir, saidFile, &v); err != nil {
		return fail(err)
	}
	return writeVerdict(runDir, v)
}

// hasSaid says whether pre left a verdict for post to write.
func hasSaid(runDir string) bool {
	_, err := os.Stat(filepath.Join(runDir, "in", saidFile))
	return err == nil
}

// unopenedBefore is the times its pull request did not open for issue
// id, as the record in the developer's comment there says (#856); 0 for
// none.
func unopenedBefore(b forge.Backlog, id int) (int, error) {
	comments, err := b.Comments(forge.Target{Kind: "issue", ID: id})
	if err != nil {
		return 0, err
	}
	n := 0
	for _, c := range comments {
		if a, ok := ReadAct(c); ok {
			n = max(n, a.Unopened)
		}
	}
	return n, nil
}

// unopened is what the engine applies when its pull request does not
// open (#856): the claim given back, the reason on the issue in the
// claim's comment, edited in place, so the next run takes it again. Its
// second time on the issue, it is blocked instead: no run takes it again
// until a person moves it back, so a failure that repeats is no loop.
func unopened(a Act, s Settings, repo string) []map[string]any {
	rec := Act{Act: "unopened", At: a.At, Issue: a.Issue, From: a.From, Unopened: a.Unopened + 1}
	why := "its pull request did not open: " + intent.WhyMark
	if rec.Unopened > 1 {
		add, remove := []string{backlog.LabelBlocked}, []string{backlog.LabelReady}
		if line, err := routing.Load(repo); err == nil {
			if ad, rm, err := line.Move("blocked"); err == nil {
				add, remove = ad, rm
			}
		}
		if !slices.Contains(remove, s.ClaimLabel) {
			remove = append(remove, s.ClaimLabel)
		}
		return []map[string]any{
			{"label": map[string]any{"issue": a.Issue, "add": add, "remove": remove, "card": "blocked"}},
			{"comment": map[string]any{"issue": a.Issue, "sticky": "claim", "body": backlog.Said(who, "blocks this issue",
				fmt.Sprintf("%s; it did not open the time before either (%d times)", why, rec.Unopened),
				"a person reads why, fixes the cause, and moves the issue back by its label") + "\n" + rec.Line()}},
		}
	}
	return []map[string]any{
		{"label": map[string]any{"issue": a.Issue, "add": []string{backlog.LabelReady}, "remove": []string{s.ClaimLabel}, "card": "ready"}},
		{"comment": map[string]any{"issue": a.Issue, "sticky": "claim", "body": backlog.Said(who, "gives this issue back", why,
			"it is free again: the developer takes it again at a next run, once; should its pull request not open again, the issue is blocked") + "\n" + rec.Line()}},
	}
}

// released is the label a claim let go moves, back from Doing to Ready:
// the claim's off, the ready label on, which the claim took off
// (ADR-0059, §2; #1081); on a board, its card out of Doing, back to the
// ready column — GitLab's column labels swapped in the same call, a
// GitHub project's Status set.
func released(is forge.Issue, s Settings, card *board.Move) map[string]any {
	v := map[string]any{"issue": is.ID, "add": []string{backlog.LabelReady}, "remove": []string{s.ClaimLabel}}
	switch {
	case card.Labels():
		remove := []string{s.ClaimLabel}
		for _, l := range card.Off {
			if slices.Contains(is.Labels, l) {
				remove = append(remove, l)
			}
		}
		v["add"], v["remove"] = []string{backlog.LabelReady, card.Column}, remove
	case card.Writes():
		v["status"] = map[string]any{"project": card.Project, "option": card.Column}
	}
	return v
}
