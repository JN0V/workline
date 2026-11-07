package line

// Blocked, read again (ADR-0059, amended): a blocked issue wakes no role
// by its state, but the reconciler, with no agent, tells whether there is
// news to judge on it since the line last read it — a comment written or
// edited by someone other than the engine, its body edited, an issue it
// waits on closed — or a week (routing.revisit) gone by with none. Then
// it wakes the product owner on it, which sends it back to a state or
// keeps it blocked with a fresh reason. With no news, no agent is called.
//
// What it costs: the issue's comments, one call a blocked issue, its
// record and its news read from them; the open issues, listed once a
// run, already by the reconciler, by an issue's own run otherwise.

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/verdict"
	"github.com/JN0V/workline/internal/work"
)

// engineMark is in every comment the engine writes: a role's, the line's.
const engineMark = "<!-- workline:"

// talkHash is the hash of an issue's comments the engine did not write,
// in their order: a person's comment, written or edited, changes it; ""
// when there are none.
func talkHash(comments []string) string {
	var theirs []string
	for _, c := range comments {
		if !strings.Contains(c, engineMark) {
			theirs = append(theirs, strings.TrimSpace(c))
		}
	}
	if len(theirs) == 0 {
		return ""
	}
	sum := sha256.Sum256([]byte(strings.Join(theirs, "\n\x00\n")))
	return hex.EncodeToString(sum[:6])
}

// whyLine reads the reason the line's own comment gave a block it wrote
// before it kept the reason in its record.
var whyLine = regexp.MustCompile(`(?m)^- \*\*Why\*\*: (.+?)\.?$`)

// oldWhy is the reason in the line's comment among comments, "" none.
func oldWhy(comments []string) string {
	sticky := forge.Marker("sticky=" + recordKey)
	for _, c := range slices.Backward(comments) {
		if strings.Contains(c, sticky) {
			if m := whyLine.FindStringSubmatch(c); m != nil {
				return m[1]
			}
			return ""
		}
	}
	return ""
}

// revisits says whether the project reads its blocked issues again: a
// number of days in routing.revisit, and the product owner routed on
// `issue`; why not, when not.
func revisits(cfg *routing.Config) (bool, string) {
	switch {
	case cfg.RevisitDays() == 0:
		return false, "routing.revisit is 0: a person moves it out"
	case !slices.Contains(cfg.Events[IssueEvent], backlog.Keeper):
		return false, "the project does not route the " + backlog.Keeper + " on issue, who would read it again: a person moves it out"
	}
	return true, ""
}

// closedBlockers are the issues is waits on that are not open: the
// forge's relation, read from the listing, and its body's "Blocked by"
// lines (ADR-0028). open is listed at first need.
func closedBlockers(b forge.Backlog, is forge.Issue, open *[]forge.Issue) ([]int, error) {
	if *open == nil && (len(backlog.BodyBlockers(is.Body)) > 0 || is.BlockedBy == nil) {
		issues, err := b.Issues()
		if err != nil {
			return nil, err
		}
		*open = issues
	}
	listed := map[int]bool{}
	for _, o := range *open {
		listed[o.ID] = true
		if o.ID == is.ID && is.BlockedBy == nil {
			is.BlockedBy = o.BlockedBy // an issue read alone does not say what it waits on
		}
	}
	var out []int
	for _, n := range backlog.Blockers(is) {
		if !listed[n] {
			out = append(out, n)
		}
	}
	return out, nil
}

// revisit reads a blocked issue: the step waking the product owner on
// it when there is news, or a week gone by; the line's record to write
// once when it first sees it blocked; nothing when nothing changed.
func revisit(res *Result, cfg *routing.Config, b forge.Backlog, is forge.Issue, open *[]forge.Issue) (*stateStep, *StateWrite, error) {
	where := fmt.Sprintf("#%d", is.ID)
	if ok, why := revisits(cfg); !ok {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "state-wakes-none", Level: "info", Where: where,
			Message: "its state, " + BlockedState + ", wakes no role: " + why})
		return nil, nil, nil
	}
	comments, err := b.Comments(forge.Target{Kind: "issue", ID: is.ID})
	if err != nil {
		return nil, nil, err
	}
	rec, had := recordIn(comments)
	closed, err := closedBlockers(b, is, open)
	if err != nil {
		return nil, nil, err
	}
	now := clock().UTC()
	days := cfg.RevisitDays()
	if !had || rec.State != BlockedState || rec.Since == "" {
		// First seen blocked: what it is now is the base news is told
		// against, written once, no agent.
		seen := Record{State: BlockedState, Body: bodyHash(is.Body), Hops: rec.Hops, Reached: rec.Reached, From: rec.From, Why: rec.Why,
			Since: now.Format(time.RFC3339), Talk: talkHash(comments), Closed: closed, Spent: rec.Spent, Counted: rec.Counted}
		if had && rec.State != BlockedState {
			seen.From = rec.State // a role blocked it: the line last acted on it there
		}
		if had && rec.State == BlockedState && seen.Why == "" {
			seen.Why = oldWhy(comments) // the line blocked it before it kept why in its record
		}
		if files := backlog.ScopeAsked(comments); len(files) > 0 {
			// The developer blocked it to ask for files outside its
			// Scope (#921): the product owner answers now, not on news.
			said := fmt.Sprintf("the developer asks to widen its %s with `%s`", work.Name(work.Scope), strings.Join(files, "`, `"))
			res.Findings = append(res.Findings, verdict.Finding{Rule: "state-revisit", Level: "info", Where: where,
				Message: fmt.Sprintf("blocked, read again by the %s: %s", backlog.Keeper, said)})
			seen.Revisited = seen.Since
			return &stateStep{role: backlog.Keeper, write: StateWrite{Issue: is.ID, Record: seen},
				inputs: map[string]string{engine.RevisitInput: said, engine.RevisitFromInput: seen.From}}, nil, nil
		}
		said := fmt.Sprintf("blocked: the %s reads it again when someone comments, its body changes, an issue it waits on closes, or in %d days",
			backlog.Keeper, days)
		res.Findings = append(res.Findings, verdict.Finding{Rule: "state-blocked-seen", Level: "info", Where: where, Message: said})
		return nil, &StateWrite{Issue: is.ID, Record: seen, Seen: said}, nil
	}
	var news []string
	if bodyHash(is.Body) != rec.Body {
		news = append(news, "its body was edited")
	}
	if talkHash(comments) != rec.Talk {
		news = append(news, "a comment was written or edited on it")
	}
	var fresh []int
	for _, n := range closed {
		if !slices.Contains(rec.Closed, n) {
			fresh = append(fresh, n)
		}
	}
	if len(fresh) > 0 {
		news = append(news, fmt.Sprintf("%s, which it waits on, %s", refs(fresh), plural(len(fresh), "is closed", "are closed")))
	}
	if machineBlocked(rec) {
		news = append(news, "the line blocked it for a run it read as a verdict, before runs had typed outcomes: it may have been the machine's error")
	}
	last := rec.Since
	if rec.Revisited != "" {
		last = rec.Revisited
	}
	at, known := labeledAt(last)
	due := !known || now.Sub(at) >= time.Duration(days)*24*time.Hour
	if len(news) == 0 && !due {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "state-blocked-quiet", Level: "info", Where: where,
			Message: fmt.Sprintf("blocked, nothing new since %s: the %s reads it again from %s, or at the first news",
				at.UTC().Format("2006-01-02"), backlog.Keeper, at.Add(time.Duration(days)*24*time.Hour).UTC().Format("2006-01-02"))})
		return nil, nil, nil
	}
	if len(news) == 0 {
		news = append(news, fmt.Sprintf("nothing new since %s, %d days or more (routing.revisit)", at.UTC().Format("2006-01-02"), days))
	}
	said := strings.Join(news, "; ")
	res.Findings = append(res.Findings, verdict.Finding{Rule: "state-revisit", Level: "info", Where: where,
		Message: fmt.Sprintf("blocked, read again by the %s: %s", backlog.Keeper, said)})
	next := rec
	next.Body, next.Talk, next.Closed, next.Revisited = bodyHash(is.Body), talkHash(comments), closed, now.Format(time.RFC3339)
	return &stateStep{role: backlog.Keeper, write: StateWrite{Issue: is.ID, Record: next},
		inputs: map[string]string{engine.RevisitInput: said, engine.RevisitFromInput: rec.From}}, nil, nil
}

// The reasons the line gave a block it made of a tester's run: its tests
// refused (#794), its "no test possible" unsettled (#1053).
const (
	refusedWhy   = "the tester wrote no tests for it in its state "
	unsettledWhy = "the tester found no test possible, and the product owner's run picked no proof nor rewrote its text"
)

// machineBlocked says whether the line blocked an issue, in a record from
// before runs had typed outcomes (V 0), for a tester's run it read as a
// verdict: a test that did not build, a patch that did not apply, a
// tester short of code (#1048, #1075, #1047). Read again at once, not in
// a week (ADR-0063, migration): the record then written has a version,
// so once.
func machineBlocked(rec Record) bool {
	return rec.V == 0 && rec.State == BlockedState && (strings.HasPrefix(rec.Why, refusedWhy) || strings.HasPrefix(rec.Why, unsettledWhy))
}

// refs names issues: #3, #4 and #5.
func refs(ids []int) string {
	words := make([]string, len(ids))
	for i, n := range ids {
		words[i] = fmt.Sprintf("#%d", n)
	}
	if len(words) == 1 {
		return words[0]
	}
	return strings.Join(words[:len(words)-1], ", ") + " and " + words[len(words)-1]
}

// blockedBody is the line's comment on a blocked issue: why, and when it
// is read again, then the record.
func blockedBody(cfg *routing.Config, r Record) string {
	why := r.Why
	if why == "" {
		why = backlog.BlockedElsewhere + "; their comment on this issue says why"
	}
	if r.From != "" && !strings.Contains(why, "its state was") {
		why += "; its state was " + r.From
	}
	next := "a person reads why on the board's Blocked column, and moves it back to a state by its label"
	if ok, _ := revisits(cfg); ok {
		next = fmt.Sprintf("the %s reads it again when someone comments, its body changes or an issue it waits on closes, else in %d days, and sends it back or says why it stays; a person may move it back by its label at any time, as the board's Blocked column shows",
			strings.ReplaceAll(backlog.Keeper, "-", " "), cfg.RevisitDays())
	}
	if r.Revisited != "" {
		next += fmt.Sprintf(" (last read again on %s; the %s's comment says what it decided)", r.Revisited[:min(len(r.Revisited), 10)], strings.ReplaceAll(backlog.Keeper, "-", " "))
	}
	return backlog.Said("workline's line", "blocks this issue", why, next) + "\n" + recordBody(r, "")
}
