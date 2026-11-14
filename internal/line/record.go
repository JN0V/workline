package line

// What the line knows of an issue it moves (ADR-0059, decision 8): the
// state a step acted on, the body it read, when the state's label was
// put on, and the hops so far. Kept in one comment of the engine's on
// the issue, edited in place. The same record twice runs nothing: a step
// is due only when the issue entered a state anew or its body changed.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/board"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/tester"
)

// Record is what the line keeps of an issue: the state its last step
// acted on, the body's hash then, when that state's label was put on as
// the forge says it, and the state steps run on it so far.
type Record struct {
	State   string `json:"state"`
	Body    string `json:"body"`
	Labeled string `json:"labeled,omitempty"`
	Hops    int    `json:"hops"`
	// Reached: the furthest state on the line the issue was in (#1068).
	// A step into a state further still is progress: its count of hops
	// starts again, so max-hops stops a loop, never the step ending it.
	// "" in a record from before: its state stands for it.
	Reached string `json:"reached,omitempty"`
	// Spec: the hash of the reviewer's comment on the issue's spec, when
	// it has one (#128): a review written since makes the step due again,
	// the product owner reading it then.
	Spec string `json:"spec,omitempty"`
	// Asked: in refine, the body's hash the step ran on with the reviewer
	// chained after the product owner (#840). A spec still waiting on a
	// review with no such mark for its body makes the step due again,
	// once: a record from before the reviewer read it in the same run.
	Asked string `json:"asked,omitempty"`
	// Revise: in refine, the review's hash (Spec) the step ran on for a
	// section the product owner may rewrite now and was not given when it
	// answered that review (backlog.Unrevised, ADR-0060): due once for it.
	Revise string `json:"revise,omitempty"`
	// Waits: the state at its limit the issue's step waits on, said on
	// it once (ADR-0059, decision 9); the step run, the record has none.
	Waits string `json:"waits,omitempty"`
	// Stale: the state the line moved the issue out of as stale
	// (ADR-0059, decision 7); Kept: the labelling, as the forge dates it,
	// whose staleness the line said and left, a person having put the
	// label back. Both kept while the issue's state stays the same.
	Stale string `json:"stale,omitempty"`
	Kept  string `json:"kept,omitempty"`
	// Blocked, read again (ADR-0059, amended; revisit.go): From, the
	// state it was blocked from, when known; Why, the line's reason for a
	// block it made; Since, when the line first saw it blocked; Talk, the
	// hash of the comments the engine did not write, as last read; Closed,
	// the issues it waits on seen closed; Revisited, when the product
	// owner last read it again.
	From      string `json:"from,omitempty"`
	Why       string `json:"why,omitempty"`
	Since     string `json:"since,omitempty"`
	Talk      string `json:"talk,omitempty"`
	Closed    []int  `json:"closed,omitempty"`
	Revisited string `json:"revisited,omitempty"`
	// Retest: the state the line moved the issue out of to its tests,
	// lacking them (wip.go); kept through the states after, so a person
	// putting it back is known. RetestKept: the labelling, as the forge
	// dates it, the line said it left that way.
	Retest     string `json:"retest,omitempty"`
	RetestKept string `json:"retest-kept,omitempty"`
	// Timeout: in refine, what waited on a person past the product
	// owner's delay that the line woke it for (ADR-0060) — `ask=N`, the
	// question of round N; `drafts=<body>`, its drafts —: woken once.
	Timeout string `json:"timeout,omitempty"`
	// Woke: when the line last woke the developer on the issue, RFC 3339;
	// Held: the issues the developer held then, in its state's next
	// (freed.go). A claim put on since and let go, or one of Held held
	// no more, frees the issue for the developer again (#894).
	Woke string `json:"woke,omitempty"`
	Held []int  `json:"held,omitempty"`
	// Unheld: the digest of the developer's comment passing the issue
	// over for an overlap the line woke it again for, that overlap gone
	// (#998): once for that comment, never again while it stands.
	Unheld string `json:"unheld,omitempty"`
	// Lost: when the developer's run the line woke ended with no verdict
	// read, its work lost (#953), in a record from before Errors: read as
	// one machine error of kind run (errorsOf), never written again.
	Lost string `json:"lost,omitempty"`
	// Errors: the machine errors of the step's runs on its input
	// (ADR-0063, §5; outcome.go): run again after a back-off within
	// ErrorBudget, then a person. Never a hop.
	Errors *Errors `json:"errors,omitempty"`
	// Needs: the code the step's agent said it lacked on its input
	// (ADR-0063, §6; outcome.go): given at its next run, once.
	Needs *Needs `json:"needs,omitempty"`
	// Last: by role, the input its last run in a state of the line was
	// on, when it said it lacked code there, or read refine (outcome.go,
	// #1157): kept through every state, a block and its revisit among
	// them, so the role is never run on that same input again as if anew.
	Last map[string]LastRun `json:"last-run,omitempty"`
	// V: the record's version: 1 since runs have typed outcomes; 0, a
	// record from before, whose block a machine error may have made
	// (revisit.go).
	V int `json:"v,omitempty"`
	// Retry: in refine, the times the line woke the product owner again
	// on the same body, its last read having left the issue unrefined
	// (unrefined.go, #951); past RefineRetries, Blocked.
	Retry int `json:"retry,omitempty"`
	// Given: in refine, the code the product owner's state named when
	// the line last woke it, its sources ("-" for none): read again on
	// the same body and the same code, it would get the same input, so it
	// is not woken again (#980). "" in a record from before.
	Given string `json:"given,omitempty"`
	// Unread: in refine, the body's hash the line woke the product owner
	// again on, its state saying it never read the issue (#1041): once a
	// body, so a run that reads nothing is not woken in a loop.
	Unread string `json:"unread,omitempty"`
	// ReviewAgain: in refine, the body's hash the line woke the step
	// again on, the reviewer asked on it already and its spec still
	// waiting on a review (#1070): once a body, never a loop.
	ReviewAgain string `json:"review-again,omitempty"`
	// Alone: the role the last step ran alone, in the place of its
	// state's, the reviewer on drafts not yet reviewed (#1150): a retry
	// of that step runs it again, its errors and block said as its.
	Alone string `json:"alone,omitempty"`
	// Ran: when the line last ran a step on the issue, RFC 3339: a step
	// due again in the same state waits from then, in the reconciler's
	// order (ADR-0059, amended 2026-11-03).
	Ran string `json:"ran,omitempty"`
	// Claim: what the line woke the developer on a claim with no pull
	// request for, to let it go (claimwait.go, #1084) — `hours:<when the
	// claim began>`, `body:<the body's hash>` —: woken once for it.
	Claim string `json:"claim,omitempty"`
	// Spent: the fresh tokens (agent.Call.Fresh) of each run a step made
	// on the issue, by role, in order (ADR-0064, step 2): kept through
	// every state, so the issue's total holds across machines, a run's
	// calls being lost with its CI job. A run that called no agent adds
	// none. Its total is checked against the issue's backstop
	// (backstop.go); the normals are measured from it (size.go).
	Spent map[string][]int `json:"spent,omitempty"`
	// Counted: the runs on a merge request closing the issue already in
	// Spent, each a short hash of its run folder's name (CountRuns): an
	// apply resumed counts none twice.
	Counted []string `json:"counted,omitempty"`
	// Sized: the states a run too big for its role asked the product
	// owner on the issue's size from (crossed.go), kept through every
	// state: once a state, a second crossing from one waits for a person
	// (ADR-0064, §4). SizeOf: the role whose run crossed, while the
	// product owner has not read the issue for it; "" once it did.
	Sized  []string `json:"sized,omitempty"`
	SizeOf string   `json:"size-of,omitempty"`
	// Backstop: the backstop, in fresh tokens, the issue's runs reached
	// when the line blocked it for a person (backstop.go, ADR-0064 §5):
	// no product owner reads it again. Gone once a person moves it out.
	Backstop int `json:"backstop,omitempty"`
	// Labels: a person's labels on the issue that are not a state's —
	// workline:accepted, a size tier (inputLabels) —, as they stood once
	// the step's own run wrote (ADR-0063, §3): one put on since is new
	// input; the role's own is not. "-" for none; "" in a record from
	// before, read against the labels' dates (labelsBefore).
	Labels string `json:"labels,omitempty"`
	// Clean: the review's hash (Spec) a step with no agent ran on, its
	// move to ready left undone (cleanReview): the agent reads it next.
	Clean string `json:"clean,omitempty"`
	// Parts: on a parent, its parts closed and a person's yes on it, as
	// the line last woke the product owner for them (partsInput,
	// parent.go, #1208): one closed since, or a yes put on, wakes it
	// again. Kept through every state.
	Parts string `json:"parts,omitempty"`
	// Heard: in refine, how many of a person's comments on the issue
	// there were when the line last woke the product owner (#1331): one
	// written since, its state saying it has not read it, wakes it once
	// for them — an answer to the reviewer's question among them. Kept
	// from step to step, so a run that reads nothing is no loop.
	Heard int `json:"heard,omitempty"`
	// AnswerHeld: the count of those comments when the line woke the
	// product owner once more, its `ready` held for them (#1331): never
	// again on the same comments.
	AnswerHeld int `json:"answer-held,omitempty"`
}

// StateWrite is what the line writes on an issue once its step ran:
// the record; with Blocked, the reason it moves the issue to the
// blocked state instead, its hops past the cap; with Waits, why its step
// waits for room, the record as it was; with Stale, why it moves the
// issue back to refine; with Kept, why it leaves a stale issue a person
// put back; with Seen, that it saw the issue blocked first, its record
// the base news is told against; with Retest, why it moves a ready
// issue to its tests; with RetestKept, why it leaves one a person put
// back. Written at once, or by `workline apply --line` when the
// line judged with no write token.
type StateWrite struct {
	Issue   int    `json:"issue"`
	Record  Record `json:"record"`
	Blocked string `json:"blocked,omitempty"`
	Waits   string `json:"waits,omitempty"`
	Stale   string `json:"stale,omitempty"`
	Kept    string `json:"kept,omitempty"`
	Seen    string `json:"seen,omitempty"`
	// Own: the runs the step itself added to Record.Spent, by role: the
	// rest of each list is what the record had when the step read it.
	Own map[string]int `json:"own,omitempty"`
	// Retest, RetestKept: see the comment above (wip.go).
	Retest     string `json:"retest,omitempty"`
	RetestKept string `json:"retest-kept,omitempty"`
	// Holds: why the product owner held the issue in refine for a spec
	// review, said on it with the record (#840): what it waits for.
	Holds string `json:"holds,omitempty"`
	// Repro: why a bug the tester could not reproduce goes back to
	// refine, its reporter having answered (repro.go); Off, the labels a
	// move takes off besides the states' — workline:needs-repro.
	Repro string   `json:"repro,omitempty"`
	Off   []string `json:"off,omitempty"`
	// Failed: the step's run ended with a machine error, its Errors in
	// the record: said on the issue with it, its labels and state kept
	// (ADR-0063, I3).
	Failed bool `json:"failed,omitempty"`
	// After: the record is of a step's run: its body and labels are those
	// of the issue as it stands when written, the role's own writes in
	// (afterWrite), so its own edit is no new input for it (ADR-0063, §3).
	After bool `json:"after,omitempty"`
	// Parent: the step told a parent its parts (parent.go): its record
	// as read, a person's edit made meanwhile kept as news, never taken
	// for the role's own write.
	Parent bool `json:"parent,omitempty"`
	// Wait: the product owner's delay (acts.ask.wait) on its step, as the
	// setting writes it, read where the record is written, maybe a job
	// with no roles folder: what waits on a person after its run is said
	// with the day it ends (#1195). "" for another role's step.
	Wait string `json:"wait,omitempty"`
}

// recordV is the version every record is written with (Record.V).
const recordV = 1

const (
	recordKey  = "line/state"   // the sticky comment's key: the engine's, not a role's
	recordMark = "line-record " // the record's line in it
	// BlockedState names the state a capped issue goes to.
	BlockedState = "blocked"
	// refineState is the state whose step the reviewer's spec review
	// follows, and whose record keeps that review's hash (#128).
	refineState = "refine"
)

// bodyHash is a body's short hash, as the record keeps it.
func bodyHash(body string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(body)))
	return hex.EncodeToString(sum[:6])
}

// readRecordSpec is the line's record on issue id, false when there is
// none, the hash of the reviewer's comment on the issue's spec (""
// with none), and why the tester's last word refused it ("" when not).
func readRecordSpec(b forge.Backlog, id int) (Record, bool, string, string, error) {
	comments, err := b.Comments(forge.Target{Kind: "issue", ID: id})
	if err != nil {
		return Record{}, false, "", "", err
	}
	r, had, spec, refused := recordFrom(comments)
	return r, had, spec, refused, nil
}

// readRecordAll is readRecordSpec, the issue's comments with it, each
// with when the forge dates it: a question's delay is counted from it
// (ADR-0060).
func readRecordAll(b forge.Backlog, id int) (Record, bool, string, string, []forge.Note, error) {
	notes, err := b.Notes(forge.Target{Kind: "issue", ID: id})
	if err != nil {
		return Record{}, false, "", "", nil, err
	}
	r, had, spec, refused := recordFrom(forge.Bodies(notes))
	return r, had, spec, refused, notes, nil
}

// recordFrom reads the record, the reviewer's spec hash and the
// tester's refusal from an issue's comments.
func recordFrom(comments []string) (Record, bool, string, string) {
	spec := ""
	for _, c := range slices.Backward(comments) {
		if holdsSpecMark(c) {
			spec = bodyHash(c)
			break
		}
	}
	r, had := recordIn(comments)
	return r, had, spec, tester.Refused(comments)
}

// holdsSpecMark says whether comment c carries the reviewer's mark as
// a line of its own: a quoted mark (after "> ") is a person's.
func holdsSpecMark(c string) bool {
	for _, l := range strings.Split(c, "\n") {
		if strings.TrimSpace(l) == backlog.SpecMarker {
			return true
		}
	}
	return false
}

// recordIn is the line's record among an issue's comments.
func recordIn(comments []string) (Record, bool) {
	sticky := forge.Marker("sticky=" + recordKey)
	for _, c := range slices.Backward(comments) {
		if !strings.Contains(c, sticky) {
			continue
		}
		_, after, ok := strings.Cut(c, "<!-- workline:"+recordMark)
		if !ok {
			continue
		}
		data, _, _ := strings.Cut(after, " -->")
		var r Record
		if json.Unmarshal([]byte(data), &r) == nil {
			return r, true
		}
	}
	return Record{}, false
}

// recordBody is the comment holding r: what a person reads, then the
// record the engine reads back.
func recordBody(r Record, role string) string {
	data, _ := json.Marshal(r)
	in := "in its state " + r.State
	if r.State == "" {
		in = "with no state label" // a parent, told as its parts close (parent.go)
	}
	said := fmt.Sprintf("**workline's line** last woke the %s on this issue, %s (%d state %s so far).",
		strings.ReplaceAll(role, "-", " "), in, r.Hops, plural(r.Hops, "step", "steps"))
	if r.Retry > 0 && role != "" {
		said += fmt.Sprintf(" Woken again on the same text, %d of %d: its last read left the issue unrefined; past that, the issue is blocked for a person.", r.Retry, RefineRetries)
	}
	if role == "" {
		said = fmt.Sprintf("**workline's line** keeps this record while the issue is blocked, %d state %s before; moved out of Blocked, its count starts again.", r.Hops, plural(r.Hops, "step", "steps"))
	}
	if n := r.Fresh(); n > 0 {
		said += fmt.Sprintf(" %s fresh tokens spent on it so far.", Tokens(n))
	}
	return said + " Edited by the engine, not by hand.\n\n<!-- workline:" + recordMark + string(data) + " -->"
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// inputLabels is what of labels is a step's input besides its state's
// (ADR-0063, §3): a person's yes, workline:accepted in either spelling,
// and a size tier; sorted, comma-joined, "-" for none.
func inputLabels(labels []string) string {
	var in []string
	for _, l := range labels {
		l = strings.ToLower(strings.TrimSpace(l))
		if backlog.Accepted(forge.Issue{Labels: []string{l}}) || backlog.SizeTier([]string{l}) != "" {
			in = append(in, l)
		}
	}
	if len(in) == 0 {
		return "-"
	}
	slices.Sort(in)
	return strings.Join(slices.Compact(in), ",")
}

// labelsBefore is the input labels a record from before Labels saw,
// read from the trail its step already asked (no call more): a size tier
// on the issue was seen — the line or the product owner puts it on, in
// that last run as often as not —; a person's yes was seen unless the
// forge dates it after the line's last run on the issue (rec.Ran). A yes
// it cannot date, or a record with no run dated, was seen: never a wake
// on every issue at once.
func labelsBefore(is forge.Issue, rec Record, t forge.Trail) string {
	fresh := rec.Ran != "" && t.Accepted != "" && later(t.Accepted, rec.Ran)
	var seen []string
	for _, l := range is.Labels {
		if fresh && backlog.Accepted(forge.Issue{Labels: []string{l}}) {
			continue // put on since its last run: new input
		}
		seen = append(seen, l)
	}
	return inputLabels(seen)
}

// afterWrite records in r the issue as it stands once the step's run
// wrote (is): its body and size tiers. A person's yes is kept as the step
// started on it (r.Labels): no role puts it on, and one put on while the
// run's writes wait for their apply job is new input. The marks the step
// kept on the body it was woken on follow it, so the role's own edit
// wakes neither it nor them (ADR-0063, §3).
func afterWrite(r *Record, is forge.Issue) {
	var in []string
	for _, l := range strings.Split(r.Labels, ",") {
		if backlog.Accepted(forge.Issue{Labels: []string{l}}) {
			in = append(in, l)
		}
	}
	for _, l := range is.Labels {
		if backlog.SizeTier([]string{l}) != "" {
			in = append(in, l)
		}
	}
	r.Labels = inputLabels(in)
	was, now := r.Body, bodyHash(is.Body)
	if was == now {
		return
	}
	for _, h := range []*string{&r.Body, &r.Asked, &r.Unread, &r.ReviewAgain} {
		if *h == was {
			*h = now
		}
	}
	for _, k := range []*string{&r.Timeout, &r.Claim} {
		if p, h, ok := strings.Cut(*k, "="); ok && h == was {
			*k = p + "=" + now
		}
		if p, h, ok := strings.Cut(*k, ":"); ok && h == was {
			*k = p + ":" + now
		}
	}
}

// due says whether the step of state s is due on is: false when the
// record says that step ran on the same state, body and labelling; else
// the hops it will be, the record's plus one for a state not acted on
// before — one, a count started again, for a state further on the line
// (cfg) than any it reached (#1068).
func due(cfg *routing.Config, s routing.State, is forge.Issue, labeled, spec string, rec Record, had bool) (Record, bool) {
	now := Record{State: s.Name, Body: bodyHash(is.Body), Labeled: labeled, Hops: rec.Hops, Spec: spec, Reached: rec.Reached, Spent: rec.Spent, Counted: rec.Counted, Sized: rec.Sized, Parts: rec.Parts, Heard: rec.Heard, AnswerHeld: rec.AnswerHeld, Labels: inputLabels(is.Labels), Last: rec.Last}
	if had && rec.State == now.State && rec.Body == now.Body && rec.Labeled == now.Labeled && rec.Spec == now.Spec && sameLabels(rec, now) {
		return rec, false
	}
	if had && rec.State == now.State {
		now.Stale, now.Kept = rec.Stale, rec.Kept // a stale move undone stays known in its state
		now.Timeout = rec.Timeout                 // a delay passed wakes the role once in its state
	}
	now.Retest, now.RetestKept = rec.Retest, rec.RetestKept // a move to its tests stays known through the states after
	if rec.State == BlockedState {
		now.Hops = 0 // a person moved it out of Blocked: its count starts again
	}
	if !had || rec.State != now.State {
		now.Hops++
	}
	reached := rec.Reached
	if reached == "" && had {
		reached = rec.State // a record from before: its state stands for the furthest
		if reached == BlockedState {
			reached = rec.From // the state it was blocked from
		}
	}
	switch {
	case reached == "" || reached == BlockedState:
		now.Reached = s.Name
	case had && rec.State != now.State && cfg != nil && cfg.Ahead(reached, s.Name):
		now.Reached, now.Hops = s.Name, 1 // progress: the count starts again
	default:
		now.Reached = reached
	}
	return now, true
}

// sameLabels says whether the input labels of now are those rec ran on:
// a record from before Labels, read by labelsBefore, holds none to tell.
func sameLabels(rec, now Record) bool {
	return rec.Labels == "" || rec.Labels == now.Labels
}

// specWaits says whether issue is's spec waits on the reviewer: its
// sections written, not ready nor accepted, no review on its body as it
// is, the review's rounds not spent (backlog.SpecHold); comments: the
// issue's, as the record was read from them.
func specWaits(is forge.Issue, comments []string) bool {
	// Its sections read as the product owner and the reviewer read them,
	// with its state: text on top may be its Need (#1062, #1070).
	st, _, err := backlog.ReadState(comments, backlog.Keeper)
	if err != nil {
		st = nil
	}
	if backlog.Readied(is) || len(backlog.NotReady(is, st, true)) > 0 {
		return false
	}
	rule, _ := backlog.SpecHold(is, comments)
	return rule == "spec-not-reviewed"
}

// writeState writes w on the forge: the record in its comment; past the
// cap, the issue moved to Blocked, every other state's label off, its
// card to the board's Blocked column, and why said in that comment; a
// step waiting for room, why, its record as it was.
func writeState(f forge.Forge, cfg *routing.Config, repo string, w StateWrite, role string) error {
	t := forge.Target{Kind: "issue", ID: w.Issue}
	w.Record.V = recordV
	var after *forge.Issue // the issue as the step's run left it
	if w.After {
		// The issue as the step's run left it, its own writes in.
		is, err := f.Issue(w.Issue)
		if err != nil {
			return err
		}
		if !w.Parent {
			afterWrite(&w.Record, *is)
		}
		after = is
	}
	body := recordBody(w.Record, role)
	switch {
	case w.Seen != "":
		body = blockedBody(cfg, w.Record) // seen blocked: its labels as they are
	case w.Blocked != "":
		add, remove, err := cfg.Move(BlockedState)
		if err != nil {
			return err
		}
		card := board.Along(f, repo, BlockedState)
		add, remove = card.Fold(add, append(remove, w.Off...))
		if err := f.Label(t, add, remove); err != nil {
			return err
		}
		card.Write(f, w.Issue)
		if w.Record.Why == "" {
			w.Record.Why = w.Blocked
		}
		body = blockedBody(cfg, w.Record)
		if w.Record.Backstop > 0 {
			// At its backstop: the label a person lifts it with is in
			// the forge's list (ADR-0064, §5).
			if b, ok := f.(forge.Backlog); ok {
				if err := backlog.EnsureTierLabel(b, backlog.TierXLarge); err != nil {
					body += fmt.Sprintf("\n\nThe label `%s` could not be made in the forge's list: %v", backlog.LabelSizeXLarge, err)
				}
			}
		}
	case w.Waits != "":
		data, _ := json.Marshal(w.Record)
		body = backlog.Said("workline's line", "waits to start this issue's next step", w.Waits,
			"the next event on it, or the hourly reconcile, starts it when there is room; a person may raise the limit in routing.wip") +
			"\n\n<!-- workline:" + recordMark + string(data) + " -->"
	case w.Repro != "":
		var err error
		if body, err = reproBody(f, cfg, repo, t, w); err != nil {
			return err
		}
	case w.Retest != "" || w.RetestKept != "":
		var err error
		if body, err = retestBody(f, cfg, repo, t, w); err != nil {
			return err
		}
	case w.Stale != "" || w.Kept != "":
		var err error
		if body, err = staleBody(f, cfg, repo, t, w); err != nil {
			return err
		}
	case w.Record.State == BlockedState:
		// Read again by the product owner: still the line's word on the
		// block, unless its run moved the issue out of it.
		is, err := f.Issue(w.Issue)
		if err != nil {
			return err
		}
		if s, ok := cfg.StateOf(is.Labels); ok && s.Name == BlockedState {
			body = blockedBody(cfg, w.Record)
		}
	case w.Holds != "":
		body = backlog.Said("workline's line", "waits for the reviewer to read this issue's spec", w.Holds,
			"the reviewer's comment on its spec, then the line wakes the product owner to read it") + "\n" + body
	case w.Failed && w.Record.Errors != nil:
		body = failedBody(w, role) + "\n" + body
	case w.Record.Needs != nil && role != "":
		who, next := strings.ReplaceAll(role, "-", " "), "the line moves it to Blocked at its next run, for a person to name its files in a comment"
		if w.Record.Needs.Times < 2 {
			next = "the line gives it at the next run"
		}
		body = backlog.Said("workline's line", "saw the "+who+" say it lacks code", w.Record.Needs.Said, next) + "\n" + body
	case role == backlog.Keeper && after != nil:
		// Its run left what a person decides — a proposal, a question or
		// drafts —: said, why and what comes next, never only who it last
		// woke (#1195).
		if said := personWaits(f, cfg, *after, w.Wait); said != "" {
			body = said + "\n" + body
		}
	}
	keepCounted(f, t, &w.Record, w.Own, &body)
	return f.Sticky(t, backlog.Inert(body), forge.Marker("sticky="+recordKey), true)
}

// keepCounted keeps in the record about to be written the runs counted
// on the issue since the step read its record: the tokens of the runs
// the comment now holds, the step's record having none of them, are added
// to rec and its body's record, found after the runs the step read, by
// own, the runs it added itself. A comment it cannot read again leaves
// the record as it is.
func keepCounted(f forge.Forge, t forge.Target, rec *Record, own map[string]int, body *string) {
	b, ok := f.(forge.Backlog)
	if !ok {
		return
	}
	comments, err := b.Comments(t)
	if err != nil {
		return
	}
	_, now, had := recordCommentIn(comments)
	if !had {
		return
	}
	var added []string
	for _, id := range now.Counted {
		if !slices.Contains(rec.Counted, id) {
			added = append(added, id)
		}
	}
	if len(added) == 0 {
		return
	}
	spent := map[string][]int{}
	for role, ns := range rec.Spent {
		spent[role] = slices.Clone(ns)
	}
	for role, ns := range now.Spent {
		// What the step read of this role's list: its own runs are no
		// part of it, and a counted run's tokens come after it.
		if read := len(spent[role]) - own[role]; read >= 0 && len(ns) > read {
			spent[role] = append(spent[role], ns[read:]...)
		}
	}
	rec.Spent = spent
	rec.Counted = append(slices.Clone(rec.Counted), added...)
	if recordLine.MatchString(*body) {
		*body = withRecord(*body, *rec)
	}
}

// WriteStates writes what a line judged with no write token left on its
// issues (Result.States), in the job that holds the token.
func WriteStates(f forge.Forge, repo string, res *Result) error {
	if len(res.States) == 0 {
		return nil
	}
	if f == nil {
		return errors.New("the line's records need a forge to be written on their issues")
	}
	cfg, err := routing.Load(repo)
	if err != nil {
		return err
	}
	for _, w := range res.States {
		if err := writeState(f, cfg, repo, w, res.Woke[w.Issue]); err != nil {
			return fmt.Errorf("#%d: %w", w.Issue, err)
		}
	}
	return nil
}

// personWaits is the line's word on an issue the product owner's run
// left waiting on a person, its delay wait as the setting writes it (""
// its default): why, and what of a person ends it. In this order: what
// its proposal offers, with its open question if any; its question or drafts, with the day they time
// out — or none, at cautious; what else keeps a proposed issue from
// ready, its drafts among them. "" when nothing waits on a person, or
// its comments do not read.
func personWaits(f forge.Forge, cfg *routing.Config, is forge.Issue, wait string) string {
	b, ok := f.(forge.Backlog)
	if !ok {
		return ""
	}
	notes, err := b.Notes(forge.Target{Kind: "issue", ID: is.ID})
	if err != nil {
		return ""
	}
	comments := forge.Bodies(notes)
	yes := fmt.Sprintf("a person labels it `%s` to do it, which wakes the product owner, or takes `%s` off to set it aside",
		backlog.AcceptedLabel(f), backlog.ProposedLabel(f))
	proposed := backlog.Proposed(is)
	var waits, next string // in refine, a question or drafts wait on a person's answer
	if s, ok := cfg.StateOf(is.Labels); ok && s.Name == refineState {
		var setting any // unset: the default delay
		if wait != "" {
			setting = wait
		}
		if w, err := backlog.ReadWait(setting); err == nil {
			t := newTally()
			t.wait = &w
			if _, _, ws, nx, err := timedOut(b, t, is, notes, Record{}); err == nil {
				waits, next = ws, nx
			}
		}
	}
	if offers := backlog.Proposes(comments, backlog.Keeper); proposed && len(offers) > 0 {
		why, then := "the product owner proposes what a person decides: "+strings.Join(offers, " "), yes
		q, open := backlog.Unanswered(notes, backlog.Keeper, is)
		st, _, _ := backlog.ReadState(comments, backlog.Keeper)
		if open && (st == nil || st.Assumed < q.Round) && waits != "" {
			// a question too, not yet assumed: timedOut says it first, never dropped
			why, then = why+"; "+waits, then+"; "+next
		}
		return backlog.Said("workline's line", "waits for a person on this issue", why, then)
	}
	if waits != "" {
		return backlog.Said("workline's line", "waits for a person on this issue", waits, next)
	}
	if st, found, err := backlog.ReadState(comments, backlog.Keeper); proposed && found && err == nil {
		if left := backlog.NotReady(is, st, false); len(left) > 0 {
			return backlog.Said("workline's line", "waits for a person on this issue",
				"the product owner proposes what it wrote, which a person decides: "+strings.Join(left, "; "), yes)
		}
	}
	return ""
}

// waitWord is a delay as acts.ask.wait writes it, read back by
// backlog.ReadWait.
func waitWord(w backlog.Delay) string {
	switch {
	case w.Never:
		return backlog.WaitNever
	case w.D == 0:
		return "0"
	}
	return w.D.String()
}
