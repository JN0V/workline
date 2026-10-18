package backlog

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/scale"
)

// Undoing (ADR-0026): a person undoing one of the role's acts, of any kind,
// puts that kind back to propose, as a closing reopened does (ADR-0025).
// Each is found mechanically, at the next run, from what the record says
// the role set and what the forge shows now.

// undoable are the kinds of act, other than closings, the record keeps to
// find undone.
var undoable = []string{"rename", "order", "effort", "milestone", "ready", "split", "depend"}

// doneMax bounds the acts the record keeps to find undone: the newest.
const doneMax = 200

// Done is one act of the role's a person may undo: its issue, its kind, the
// value it had and the value the role set.
type Done struct {
	Issue int    `yaml:"issue,omitempty"` // left out in the issue's own state
	Act   string `yaml:"act"`
	Was   string `yaml:"was,omitempty"`
	Set   string `yaml:"set,omitempty"`
	Level string `yaml:"level,omitempty"` // the autonomy level it was done at
}

// Undo is an act a person undid, and what shows it.
type Undo struct {
	Issue    int    `yaml:"issue,omitempty"` // left out in the issue's own state
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
	if src, ok := axisOf(c); ok && src.Hidden() {
		return // its values would show in the issue's state: not watched (ADR-0040)
	}
	is := p.issues[c.Issue]
	d := Done{Issue: c.Issue, Act: c.Do, Level: p.config.Level}
	switch c.Do {
	case "rename":
		d.Was, d.Set = is.Title, c.Title
	case "order":
		d.Was, d.Set = strconv.Itoa(Priority(is)), strconv.Itoa(c.Priority)
	case "effort":
		d.Was, d.Set = strconv.Itoa(Effort(is)), strconv.Itoa(c.Effort)
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
// undone, with their evidence, and those that still stand; and the
// blockers the role set on each open issue (own). An act whose issue is
// closed, or whose value a person set to a third, is no longer the role's
// to watch.
func findUndone(f forge.Backlog, role string, open map[int]forge.Issue, done []Done) (standing []Done, undone []Undo, own map[int][]int, err error) {
	var all map[int]forge.Issue // read once, when a split is watched
	own = map[int][]int{}
	for _, d := range done {
		is, isOpen := open[d.Issue]
		if !isOpen {
			if d.Act != "split" {
				continue
			}
			kept, err := closedSplit(f, role, open, d.Issue, own)
			if err != nil {
				return nil, nil, nil, err
			}
			if kept {
				standing = append(standing, d)
			}
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
			case now != d.Set && scale.Used.Detected:
				// Changed after the role set it: corrected, as cleared
				// (ADR-0040), never written again there.
				evidence = fmt.Sprintf("#%d's priority changed to %s by a person; the role had set %s", d.Issue, priorityName(now), priorityName(d.Set))
			case now != d.Set:
				gone = true
			}
		case "effort":
			if now := strconv.Itoa(Effort(is)); now != d.Set && scale.Used.Effort.Said != "" {
				evidence = fmt.Sprintf("#%d's effort changed to %s by a person; the role had set %s", d.Issue, effortName(now), effortName(d.Set))
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
				if err == nil {
					own[d.Issue] = append(own[d.Issue], b)
				}
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
			st, err := splitState(f, role, d.Issue)
			if err != nil {
				return nil, nil, nil, err
			}
			if st == nil || len(st.Split) == 0 {
				break // its children not recorded yet, or its state unread: watched again
			}
			for child, blockers := range st.After {
				own[child] = append(own[child], blockers...)
			}
			if all == nil {
				list, err := f.AllIssues()
				if err != nil {
					return nil, nil, nil, err
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
	return standing, undone, own, nil
}

// closedSplit reads a split parent closed while its parts still wait on
// one another: those links are still the role's own, read from its state
// into own until no part that waits is open (ADR-0028). kept says it is
// still watched: a part waits, or its state does not read — read again at
// the next run. A parent the forge no longer gives is let go.
func closedSplit(f forge.Backlog, role string, open map[int]forge.Issue, parent int, own map[int][]int) (kept bool, err error) {
	st, err := splitState(f, role, parent)
	switch {
	case errors.Is(err, forge.ErrUnreachable):
		return false, err
	case err != nil:
		return false, nil
	case st == nil:
		return true, nil
	}
	for child, blockers := range st.After {
		if _, ok := open[child]; ok {
			own[child], kept = append(own[child], blockers...), true
		}
	}
	return kept, nil
}

// closedParents are the split parents, closed, that a part still open and
// waiting on an issue names, each once and in order (#393): the parent's
// state holds the links the split set among its parts, and the issues
// read are the open ones only. Those the record holds already are left
// out (done: an old report's split).
func closedParents(open map[int]forge.Issue, done []Done) []int {
	var out []int
	for _, is := range open {
		id := partOf(is.Body)
		if _, isOpen := open[id]; id == 0 || isOpen || len(Blockers(is)) == 0 {
			continue
		}
		if !slices.ContainsFunc(done, func(d Done) bool { return d.Issue == id && d.Act == "split" }) {
			out = append(out, id)
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// partOf is the parent a split's part names: the marker it was opened
// with (SplitKey), else the line the engine writes first (ChildBody); 0
// for none.
func partOf(body string) int {
	m := splitMarker.FindStringSubmatch(body)
	if m == nil {
		m = partOfLine.FindStringSubmatch(body)
	}
	if m == nil {
		return 0
	}
	id, _ := strconv.Atoi(m[1])
	return id
}

var (
	splitMarker = regexp.MustCompile(`<!-- workline:split=(\d+)/`)
	partOfLine  = regexp.MustCompile(`^\s*Part of #(\d+)\.`)
)

// splitState is a split parent's state as its comment holds it: empty
// when it has none; nil when it does not read — never taken for none.
func splitState(f forge.Backlog, role string, id int) (*State, error) {
	notes, err := f.Notes(forge.Target{Kind: "issue", ID: id})
	if err != nil {
		return nil, err
	}
	st, found, err := ReadState(forge.Bodies(notes), role)
	switch {
	case err != nil:
		return nil, nil
	case !found:
		return &State{}, nil
	}
	return st, nil
}

// priorityName says a priority the ledger keeps as its level's number.
func priorityName(n string) string {
	l, _ := strconv.Atoi(n)
	return scale.Value(l)
}

// effortName says an effort the ledger keeps as its level's number.
func effortName(n string) string {
	l, _ := strconv.Atoi(n)
	return scale.EffortValue(l)
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
	facts.Need, facts.Example, facts.Steps, facts.Validation, facts.Description = "", "", "", "", ""
	drafts.Scope, drafts.Verification, drafts.Sources = "", "", nil
	_, facts.Added, _ = Refine(is.Body, facts, role)
	_, drafts.Added, _ = Refine(is.Body, drafts, role)
	return facts, drafts, len(drafts.Added) > 0
}
