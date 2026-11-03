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
	// Lost: when the developer's run the line woke ended with no verdict
	// read, its work lost (#953): due again, once, a hop more, so a run
	// lost each time stops at max-hops, Blocked.
	Lost string `json:"lost,omitempty"`
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
	// Lost: why the developer's run ended with no verdict read, said on
	// the issue with the record, which makes its step due again (#953).
	Lost string `json:"lost,omitempty"`
}

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
	said := fmt.Sprintf("**workline's line** last woke the %s on this issue, in its state %s (%d state %s so far).",
		strings.ReplaceAll(role, "-", " "), r.State, r.Hops, plural(r.Hops, "step", "steps"))
	if role == "" {
		said = fmt.Sprintf("**workline's line** keeps this record while the issue is blocked, %d state %s before; moved out of Blocked, its count starts again.", r.Hops, plural(r.Hops, "step", "steps"))
	}
	return said + " Edited by the engine, not by hand.\n\n<!-- workline:" + recordMark + string(data) + " -->"
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// due says whether the step of state s is due on is: false when the
// record says that step ran on the same state, body and labelling; else
// the hops it will be, the record's plus one for a state not acted on
// before.
func due(s routing.State, is forge.Issue, labeled, spec string, rec Record, had bool) (Record, bool) {
	now := Record{State: s.Name, Body: bodyHash(is.Body), Labeled: labeled, Hops: rec.Hops, Spec: spec}
	if had && rec.State == now.State && rec.Body == now.Body && rec.Labeled == now.Labeled && rec.Spec == now.Spec {
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
	return now, true
}

// specWaits says whether issue is's spec waits on the reviewer: its
// sections written, not ready nor accepted, no review on its body as it
// is, the review's rounds not spent (backlog.SpecHold); comments: the
// issue's, as the record was read from them.
func specWaits(is forge.Issue, comments []string) bool {
	if backlog.Readied(is) || len(backlog.NotReady(is, nil, true)) > 0 {
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
	case w.Lost != "":
		body = backlog.Said("workline's line", "lost the developer's run on this issue", w.Lost,
			"the next event on it, or the hourly reconcile, wakes the developer again; a run lost each time stops at max-hops, the issue moved to Blocked") + "\n" + body
	}
	return f.Sticky(t, backlog.Inert(body), forge.Marker("sticky="+recordKey), true)
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
