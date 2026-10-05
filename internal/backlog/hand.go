package backlog

import (
	"fmt"
	"regexp"
	"slices"

	"github.com/JN0V/workline/internal/forge"
)

// The person's hand (ADR-0025): a box ticked in the report by a person of
// the project is their yes, applied by the engine at the next run; runs
// nobody answers pause the role.

// PauseAfter is the runs in a row whose report proposed something and that
// no person answered, after which the role pauses (ADR-0018).
const PauseAfter = 3

// KeyResume is the box that resumes a paused role; KeyAct, followed by a
// kind of act, the box that sets that kind back to act.
const (
	KeyResume = "resume"
	KeyAct    = "act/"
)

// TickMarker is the hidden key that ends a line of the report a person
// may tick.
func TickMarker(key string) string { return forge.Marker("proposal=" + key) }

// tickedLine is a ticked box of the report, its key.
var tickedLine = regexp.MustCompile(`(?m)^\s*[-*+] \[[xX]\] .*<!-- workline:proposal=(\S+) -->\s*$`)

// itemKey finds a key in a ticked item as a forge gives it: the marker
// whole, or its text alone (GitLab drops the comment's delimiters).
var itemKey = regexp.MustCompile(`workline:proposal=(\S+?)(?:\s|-->|\*\*|$)`)

// TickBy is a box ticked in the report now, with who ticked it.
type TickBy struct {
	Key string
	forge.Note
	Known bool // the forge says who ticked it: by name, or as a person of the project
}

// Person says whether the tick is a person of the project's: their yes.
func (t TickBy) Person() bool { return t.Known && t.Insider && !t.Bot }

// Who names who ticked, for the report and the issue told.
func (t TickBy) Who() string {
	if t.Author != "" {
		return "@" + t.Author
	}
	return "a person of the project"
}

// why says why a tick is not a person's yes.
func (t TickBy) why() string {
	switch {
	case !t.Known:
		return "the forge does not say who ticked it"
	case t.Bot:
		return "ticked by " + t.Who() + ", a bot"
	}
	return "ticked by " + t.Who() + ", outside the project"
}

// Hand is what people did on the report since the last run.
type Hand struct {
	Report   int
	Record   Record
	Broken   error    // the record does not read
	Ticks    []TickBy // the boxes ticked in the report's body now
	Comments int      // comments of people of the project on the report
	Signs    []string // what a person did since the last run: a tick, a comment, a closing undone, a proposal settled
}

// Paused says whether the role pauses: runs in a row nobody answered, and
// no person's hand since.
func (h *Hand) Paused() bool { return h.Record.Ignored >= PauseAfter && len(h.Signs) == 0 }

// Tick is the box ticked with that key, if any.
func (h *Hand) Tick(key string) (TickBy, bool) {
	for _, t := range h.Ticks {
		if t.Key == key {
			return t, true
		}
	}
	return TickBy{}, false
}

// Ticked is a proposal a person of the project ticked, the engine able to
// do it: what pre gives the run, applied as the proposal recorded says.
func (h *Hand) Ticked() []Pending {
	var out []Pending
	for _, q := range h.Record.Proposed {
		if t, ok := h.Tick(q.TickKey()); ok && t.Person() && q.Doable() {
			out = append(out, q)
		}
	}
	return out
}

// TickKey is the pending proposal's key, as its line in the report carries it.
func (q Pending) TickKey() string {
	if q.Key != "" {
		return q.Key
	}
	return fmt.Sprintf("%d/%s", q.Issue, q.Act)
}

// Doable says whether the engine can do a proposal ticked: what it would
// do recorded, and not one only a person or the import does.
func (q Pending) Doable() bool {
	c := q.Proposal
	return c != nil && c.Do != "open" && !c.Spent && !(c.Do == "milestone" && c.Milestone == "")
}

// ReadHand reads the report: its record, the boxes ticked in its body with
// who ticked them, the people's comments on it, and the signs of a person
// since the last run. No report: an empty hand.
func ReadHand(f forge.Backlog, role string, open []forge.Issue) (*Hand, error) {
	h := &Hand{}
	isOpen := map[int]bool{}
	body := ""
	for _, is := range open {
		isOpen[is.ID] = true
		if is.Title == ReportTitle(role) {
			h.Report, body = is.ID, is.Body
		}
	}
	if h.Report == 0 {
		return h, nil
	}
	notes, err := f.Notes(forge.Target{Kind: "issue", ID: h.Report})
	if err != nil {
		return nil, err
	}
	if _, err := readBlock(forge.Bodies(notes), RecordMarker(role), &h.Record); err != nil {
		h.Broken, h.Record = err, Record{}
	}
	for _, n := range notes {
		if _, engine := EngineMarker(n.Body); !engine && n.Insider && !n.Bot {
			h.Comments++
		}
	}
	if h.Comments > h.Record.Comments {
		h.Signs = append(h.Signs, "a comment on the report")
	}
	var keys []string
	for _, m := range tickedLine.FindAllStringSubmatch(body, -1) {
		keys = append(keys, m[1])
	}
	if len(keys) > 0 {
		events, err := f.Ticks(h.Report)
		if err != nil {
			return nil, err
		}
		for _, key := range keys {
			t := TickBy{Key: key}
			for _, e := range events { // the last word on that box
				if m := itemKey.FindStringSubmatch(e.Item); m != nil && m[1] == key {
					t.Note, t.Known = e.Note, e.Done && (e.Author != "" || e.Insider)
				}
			}
			h.Ticks = append(h.Ticks, t)
			if t.Person() {
				h.Signs = append(h.Signs, "a box ticked by "+t.Who())
			}
		}
	}
	for _, c := range h.Record.Closed {
		if isOpen[c.Issue] {
			h.Signs = append(h.Signs, fmt.Sprintf("#%d reopened", c.Issue))
		}
	}
	for _, q := range h.Record.Proposed {
		if q.Key == "" && !q.Capped && q.Issue > 0 && !isOpen[q.Issue] {
			h.Signs = append(h.Signs, fmt.Sprintf("#%d closed", q.Issue))
		}
	}
	slices.Sort(h.Signs)
	h.Signs = slices.Compact(h.Signs)
	return h, nil
}
