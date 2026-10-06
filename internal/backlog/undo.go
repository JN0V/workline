package backlog

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/forge"
)

// Undoing (ADR-0026): a person undoing one of the role's acts, of any kind,
// puts that kind back to propose, as a closing reopened does (ADR-0025).
// Each is found mechanically, at the next run, from what the record says
// the role set and what the forge shows now.

// undoable are the kinds of act, other than closings, the record keeps to
// find undone.
var undoable = []string{"rename", "order", "milestone", "ready", "split", "depend"}

// doneMax bounds the acts the record keeps to find undone: the newest.
const doneMax = 200

// Done is one act of the role's a person may undo: its issue, its kind, the
// value it had and the value the role set.
type Done struct {
	Issue int    `yaml:"issue"`
	Act   string `yaml:"act"`
	Was   string `yaml:"was,omitempty"`
	Set   string `yaml:"set,omitempty"`
	Level string `yaml:"level,omitempty"` // the autonomy level it was done at
}

// Undo is an act a person undid, and what shows it.
type Undo struct {
	Issue    int    `yaml:"issue"`
	Act      string `yaml:"act"`
	Evidence string `yaml:"evidence"`
	Day      string `yaml:"day,omitempty"` // when a run found it, YYYY-MM-DD: the weekly sample counts it against one act
}

// recordDone keeps an act of the role's to find it undone at a later run:
// the newest of each issue and kind. A move the engine made on its own (a
// slip) is not the role's choice, and a person's tick is theirs.
func (p *Plan) recordDone(c Proposal) {
	if !slices.Contains(undoable, c.Do) || c.Ticked != "" || c.From != "" {
		return
	}
	is := p.issues[c.Issue]
	d := Done{Issue: c.Issue, Act: c.Do, Level: p.config.Level}
	switch c.Do {
	case "rename":
		d.Was, d.Set = is.Title, c.Title
	case "order":
		d.Was, d.Set = strconv.Itoa(Priority(is)), strconv.Itoa(c.Priority)
	case "milestone":
		d.Was, d.Set = is.Milestone, c.Milestone
	case "depend":
		// Each depend on an issue kept: the blockers it added, together.
		for _, o := range p.Record.Done {
			if o.Issue == c.Issue && o.Act == "depend" && o.Set != "" {
				d.Set = o.Set + ","
			}
		}
		d.Set += joinIDs(c.BlockedBy)
	}
	p.Record.Done = slices.DeleteFunc(p.Record.Done, func(o Done) bool { return o.Issue == d.Issue && o.Act == d.Act })
	p.Record.Done = append(p.Record.Done, d)
	if n := len(p.Record.Done); n > doneMax {
		p.Record.Done = p.Record.Done[n-doneMax:]
	}
}

// findUndone reads what a person did to the acts the record keeps: those
// undone, with their evidence, and those that still stand. An act whose
// issue is closed, or whose value a person set to a third, is no longer
// the role's to watch.
func findUndone(f forge.Backlog, role string, open map[int]forge.Issue, done []Done) (standing []Done, undone []Undo, err error) {
	var all map[int]forge.Issue // read once, when a split is watched
	for _, d := range done {
		is, isOpen := open[d.Issue]
		if !isOpen {
			continue
		}
		var evidence string
		gone := false
		switch d.Act {
		case "rename":
			switch {
			case is.Title == d.Was:
				evidence = fmt.Sprintf("#%d renamed back to %q by a person; the role had set %q", d.Issue, d.Was, d.Set)
			case is.Title != d.Set:
				gone = true
			}
		case "order":
			switch now := strconv.Itoa(Priority(is)); {
			case now == d.Was:
				evidence = fmt.Sprintf("#%d's priority put back to %s by a person; the role had set %s", d.Issue, priorityName(d.Was), priorityName(d.Set))
			case now != d.Set:
				gone = true
			}
		case "milestone":
			switch {
			case is.Milestone == d.Was:
				evidence = fmt.Sprintf("#%d put back in %s by a person; the role had put it in %q", d.Issue, milestoneName(d.Was), d.Set)
			case is.Milestone != d.Set:
				gone = true
			}
		case "depend":
			// A blocker the role added, still open, no longer one: a person
			// took the link or the line off. All closed: nothing to watch.
			now, left := Blockers(is), 0
			for _, f := range strings.Split(d.Set, ",") {
				b, err := strconv.Atoi(f)
				if _, isOpen := open[b]; err != nil || !isOpen {
					continue
				}
				left++
				if !slices.Contains(now, b) && evidence == "" {
					evidence = fmt.Sprintf("#%d no longer waits on #%d: a person took off the link the role set", d.Issue, b)
				}
			}
			gone = left == 0
		case "ready":
			if !slices.Contains(is.Labels, LabelReady) {
				evidence = fmt.Sprintf("#%d's label %s taken off by a person; the role had moved it to ready", d.Issue, LabelReady)
			}
		case "split":
			notes, err := f.Notes(forge.Target{Kind: "issue", ID: d.Issue})
			if err != nil {
				return nil, nil, err
			}
			st, found, err := ReadState(forge.Bodies(notes), role)
			if !found || err != nil || len(st.Split) == 0 {
				break // its children not recorded yet: watched again
			}
			if all == nil {
				list, err := f.AllIssues()
				if err != nil {
					return nil, nil, err
				}
				all = map[int]forge.Issue{}
				for _, x := range list {
					all[x.ID] = x
				}
			}
			closed := 0
			for _, id := range st.Split {
				ch, ok := all[id]
				switch {
				case ok && ch.Closed && ch.Reason == "not_planned":
					evidence = fmt.Sprintf("#%d, split from #%d by the role, closed as not planned by a person", id, d.Issue)
				case !ok || ch.Closed: // closed, or gone from the forge: deleted, moved
					closed++
				}
				if evidence != "" {
					break
				}
			}
			gone = evidence == "" && closed == len(st.Split) // no child left open: nothing left to undo
		}
		switch {
		case evidence != "":
			undone = append(undone, Undo{Issue: d.Issue, Act: d.Act, Evidence: evidence})
		case !gone:
			standing = append(standing, d)
		}
	}
	return standing, undone, nil
}

func priorityName(n string) string {
	if n == "0" || n == "" {
		return "none"
	}
	return n
}

func milestoneName(m string) string {
	if m == "" {
		return "no milestone"
	}
	return fmt.Sprintf("%q", m)
}

// splitDrafts parts a refine at a cautious level: what the code shows
// (Scope, Verification, their sources), written; the Need and Validation
// drafts, proposed. Not when the text is a person's already — agreed to in a
// reply, accepted by the label — nor proposed to an outsider in a comment;
// ok is false then, and when it holds no draft.
func (p *Plan) splitDrafts(c Proposal, role string) (facts, drafts Proposal, ok bool) {
	is := p.issues[c.Issue]
	if c.ToReporter || c.Agreed != "" || Accepted(is) {
		return c, c, false
	}
	facts, drafts = c, c
	facts.Need, facts.Validation = "", ""
	drafts.Scope, drafts.Verification, drafts.Sources = "", "", nil
	_, facts.Added, _ = Refine(is.Body, facts, role)
	_, drafts.Added, _ = Refine(is.Body, drafts, role)
	return facts, drafts, len(drafts.Added) > 0
}
