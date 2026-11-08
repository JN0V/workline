package line

// A parent and its parts (ADR-0029, amended): a split's parent bears no
// state the line wakes a role by once a person lifted its block, or
// stays blocked; its parts close in their own runs. The line tells it
// anyway: a part closed, or a person's yes put on it, since the line
// last read it, wakes the product owner on it with no agent, which
// writes what its parts delivered and, all closed, proposes closing it
// to a person (#1208). Read, it waits: on its parts, or on that person,
// said — never off the line with nothing said (ADR-0063, I1).

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/verdict"
)

// partsInput is what of a parent's parts and labels is the input of its
// step: the parts closed, by number, and a person's yes; "" when no part
// is closed and no yes is on it.
func partsInput(closed []int, is forge.Issue) string {
	words := make([]string, len(closed))
	for i, n := range closed {
		words[i] = strconv.Itoa(n)
	}
	in := strings.Join(words, ",")
	if backlog.Accepted(is) {
		in += "+accepted"
	}
	return in
}

// parentStep reads issue is as a parent, when its state wakes no role
// (s, none or Blocked): its step, the product owner's with no agent, when
// a part closed or a yes was put on since the line last read it; handled
// (true) with nothing to run when it waits on its parts or a person, said
// in res; false when it is no parent, the line reading it as any issue.
// open is listed at first need: an issue read alone does not say its
// parts.
func parentStep(res *Result, cfg *routing.Config, b forge.Backlog, is forge.Issue, s routing.State, open *[]forge.Issue) (*stateStep, bool, error) {
	if !slices.Contains(cfg.Events[IssueEvent], backlog.Keeper) {
		return nil, false, nil // nobody to tell it: the line reads it as any issue
	}
	if *open == nil {
		issues, err := b.Issues()
		if err != nil {
			return nil, false, err
		}
		*open = issues
	}
	listed := map[int]bool{}
	for _, o := range *open {
		listed[o.ID] = true
		if o.ID == is.ID && is.Children == nil {
			is.Children = o.Children // an issue read alone does not say its parts
		}
	}
	parts := backlog.Parts(is)
	if len(parts) == 0 {
		return nil, false, nil
	}
	var closed, left []int
	for _, n := range parts {
		if listed[n] {
			left = append(left, n)
		} else {
			closed = append(closed, n)
		}
	}
	where := fmt.Sprintf("#%d", is.ID)
	rec, _, _, _, _, err := readRecordAll(b, is.ID)
	if err != nil {
		return nil, false, err
	}
	in := partsInput(closed, is)
	if in != rec.Parts {
		now := rec
		now.Parts = in
		if now.Body == "" {
			now.Body = bodyHash(is.Body) // no record yet: its body as read, never taken for its own edit
		}
		said := fmt.Sprintf("a parent, %s of its %d parts closed", plural(len(closed), "one", strconv.Itoa(len(closed))), len(parts))
		if len(closed) == 0 {
			said = fmt.Sprintf("a parent, none of its %d parts closed", len(parts))
		}
		if backlog.Accepted(is) {
			said += ", a person's yes on it"
		}
		res.Findings = append(res.Findings, verdict.Finding{Rule: "state-parent", Level: "info", Where: where,
			Message: said + ": the " + backlog.Keeper + " says what they delivered, with no agent"})
		return &stateStep{role: backlog.Keeper, write: StateWrite{Issue: is.ID, Record: now}, from: s, rec: rec,
			since: never, tier: backlog.SizeTier(is.Labels), ai: "none", parent: true}, true, nil
	}
	if s.Name == BlockedState {
		return nil, false, nil // read again on its news, as any blocked issue (revisit.go)
	}
	msg := fmt.Sprintf("a parent, its %d parts closed: it waits for a person to accept it, by closing it or with %s, which closes it", len(parts), backlog.LabelAccepted)
	if len(left) > 0 {
		msg = fmt.Sprintf("a parent, it waits on its open parts, %s: the %s tells it as each closes", refs(left), backlog.Keeper)
	}
	res.Findings = append(res.Findings, verdict.Finding{Rule: "state-parent-waits", Level: "info", Where: where, Message: msg})
	return nil, true, nil
}
