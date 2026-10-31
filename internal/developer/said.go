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
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
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
// request closes: its label off, why written in its comment. A claim
// whose day the forge does not say is kept, and said in the verdict.
func stale(b forge.Backlog, issues []forge.Issue, mrs []forge.MergeRequest, s Settings, now time.Time) ([]intent.Intention, []verdict.Finding, error) {
	closed := map[int]bool{}
	for _, m := range mrs {
		for _, n := range forge.Closes(m.Title, m.Body) {
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
		if hours < float64(s.ClaimHours) {
			continue
		}
		why := fmt.Sprintf("it took this issue on %s, and no pull request closes it %d hours later (claim-hours: %d)",
			backlog.Day(t.Labeled), int(hours), s.ClaimHours)
		out = append(out,
			intent.Intention{Kind: "label", Value: map[string]any{"issue": is.ID, "remove": []string{s.ClaimLabel}}},
			sayOn(is.ID, backlog.Said(who, "lets this issue go", why,
				"it is free again: the developer may take it at a next run, or a person assigns themselves")))
		found = append(found, verdict.Finding{Rule: "claim-released", Level: "warn", Where: where,
			Message: why + ": " + s.ClaimLabel + " taken off, said on the issue"})
	}
	return out, found, nil
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
