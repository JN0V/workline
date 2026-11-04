package line

// A ready issue freed for the developer again (#894; ADR-0059,
// amended 2026-11-02): the line woke the developer on it once, and the
// record holding that state, body and labelling ran nothing more. Yet
// the developer may leave an issue free for a reason that ends: it took
// it and let the claim go, no pull request after claim-hours; or it
// passed it over for an issue it held, whose Scope overlaps. Nothing
// else wakes it then, and a full Ready of such issues stops the line.
//
// Due again, once for each end, never on a clock: its own claim put on
// after the wake (back from Doing, one hop more, so a take-and-let-go
// cycle stops at max-hops, Blocked); or an issue held at the wake held no
// more. A record from before the line kept the wake is due once when the
// developer said a word on the issue: it let it go or passed it over.

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/developer"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/work"
)

// developerSaid is the developer's one comment on an issue: its claim,
// a wait, or a claim let go (internal/developer, said.go).
var developerSaid = forge.Marker("sticky=" + developerRole + "/claim")

// freed says why issue is, in the developer's state s, is free for the
// developer again though its record holds its state, body and labelling:
// "" when it is not. rec comes back as the step's record, a hop more for
// a claim let go. open: the open issues, listed at first need.
func freed(cfg *routing.Config, b forge.Backlog, tl *tally, s routing.State, is forge.Issue, rec Record, comments []string, open *[]forge.Issue) (Record, string, error) {
	next, ok := cfg.State(s.Next)
	if s.Role != developerRole || !ok || s.Next == "" || len(is.Assignees) > 0 {
		return rec, "", nil // a person's issue is theirs
	}
	t, err := b.Trail(is.ID, next.Label)
	if err != nil {
		return rec, "", err
	}
	if t.Labeled != "" && (rec.Woke == "" || later(t.Labeled, rec.Woke)) {
		rec.Hops++ // back from Doing: a state step
		return rec, fmt.Sprintf("its claim, %s put on %s, was let go since the line woke the developer on it: free to take again, back from %s a state step",
			next.Label, backlog.Day(t.Labeled), next.Name), nil
	}
	if rec.Woke == "" {
		if slices.ContainsFunc(comments, func(c string) bool { return strings.Contains(c, developerSaid) }) {
			return rec, "the developer left it free, and the line's record, from before it kept when it woke the developer, cannot say whether what held it still does: woken once more", nil
		}
		return rec, "", nil
	}
	// Passed over for an overlap, the developer's last word on it: what
	// ends that wait may be its own Scope, read as a guess (#998). Once
	// for that comment: a developer run that leaves it as it is does not
	// wake the issue again (Unheld).
	overlapped, said := false, ""
	for _, c := range slices.Backward(comments) {
		if strings.Contains(c, developerSaid) {
			said = bodyHash(c)
			overlapped = developer.PassedForOverlap(c) && rec.Unheld != said
			break
		}
	}
	if len(rec.Held) == 0 && !overlapped {
		return rec, "", nil
	}
	held, claims, err := heldIn(b, tl, next, is.ID, open)
	if err != nil {
		return rec, "", err
	}
	for _, n := range rec.Held {
		if !slices.Contains(held, n) {
			return rec, fmt.Sprintf("#%d, held when the line woke the developer on it, is held no more: free to take again", n), nil
		}
	}
	if n, _ := developer.Overlapping(is, claims, tl.files); overlapped && n == 0 {
		rec.Unheld = said
		return rec, fmt.Sprintf("the developer passed it over for an overlap, but its %s overlaps no issue held now (a %[1]s its own words call a guess holds nothing): free to take again", work.Name(work.Scope)), nil
	}
	return rec, "", nil
}

// heldIn is the open issues in state next, the developer's claims that
// hold their Scope (a claim with no code holds none, #952), but id: what
// it held when woken on id, and each one's Scope's paths. open: listed
// at first need.
func heldIn(b forge.Backlog, tl *tally, next routing.State, id int, open *[]forge.Issue) ([]int, map[int][]string, error) {
	if *open == nil {
		issues, err := b.Issues()
		if err != nil {
			return nil, nil, err
		}
		*open = issues
	}
	claims, err := tl.held(b, next, *open)
	if err != nil {
		return nil, nil, err
	}
	var held []int
	for n := range claims {
		if n != id {
			held = append(held, n)
		}
	}
	slices.Sort(held)
	return held, claims, nil
}

// woken is the record of a step that wakes the developer: when, and the
// issues it holds then.
func woken(cfg *routing.Config, b forge.Backlog, tl *tally, s routing.State, id int, rec Record, open *[]forge.Issue) (Record, error) {
	next, ok := cfg.State(s.Next)
	if s.Role != developerRole || !ok || s.Next == "" {
		return rec, nil
	}
	held, _, err := heldIn(b, tl, next, id, open)
	if err != nil {
		return rec, err
	}
	rec.Woke, rec.Held = clock().UTC().Format(time.RFC3339), held
	return rec, nil
}

// later says whether a, a moment as the forge says it (RFC 3339 or a
// day), is after b; false when either does not read.
func later(a, b string) bool {
	ta, ok := backlog.Moment(a)
	tb, ok2 := backlog.Moment(b)
	return ok && ok2 && ta.After(tb)
}
