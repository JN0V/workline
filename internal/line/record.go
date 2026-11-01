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
}

// StateWrite is what the line writes on an issue once its step ran:
// the record; with Blocked, the reason it moves the issue to the
// blocked state instead, its hops past the cap. Written at once, or by
// `workline apply --line` when the line judged with no write token.
type StateWrite struct {
	Issue   int    `json:"issue"`
	Record  Record `json:"record"`
	Blocked string `json:"blocked,omitempty"`
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
	spec := ""
	for _, c := range slices.Backward(comments) {
		if holdsSpecMark(c) {
			spec = bodyHash(c)
			break
		}
	}
	r, had := recordIn(comments)
	return r, had, spec, tester.Refused(comments), nil
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
		role, r.State, r.Hops, plural(r.Hops, "step", "steps"))
	if role == "" {
		said = fmt.Sprintf("**workline's line** blocked this issue after %d state steps; moved out of Blocked, its count starts again.", r.Hops)
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
	if rec.State == BlockedState {
		now.Hops = 0 // a person moved it out of Blocked: its count starts again
	}
	if !had || rec.State != now.State {
		now.Hops++
	}
	return now, true
}

// writeState writes w on the forge: the record in its comment; past the
// cap, the issue moved to Blocked, every other state's label off, its
// card to the board's Blocked column, and why said in that comment.
func writeState(f forge.Forge, cfg *routing.Config, repo string, w StateWrite, role string) error {
	t := forge.Target{Kind: "issue", ID: w.Issue}
	body := recordBody(w.Record, role)
	switch {
	case w.Blocked != "":
		add, remove, err := cfg.Move(BlockedState)
		if err != nil {
			return err
		}
		card := board.Along(f, repo, BlockedState)
		add, remove = card.Fold(add, remove)
		if err := f.Label(t, add, remove); err != nil {
			return err
		}
		card.Write(f, w.Issue)
		body = backlog.Said("workline's line", "blocks this issue", w.Blocked,
			"a person reads why on the board's Blocked column, and moves it back to a state by its label") +
			"\n" + recordBody(w.Record, "")
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
