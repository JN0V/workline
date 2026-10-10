package backlog

import (
	"fmt"
	"slices"
	"time"
	"unicode/utf8"

	"github.com/JN0V/workline/internal/forge"
)

// The acts the role did, each kept in its issue's state with its day and
// the autonomy level it was done at (ADR-0038): said in the role's comment
// on the issue, and read by `workline sample --apply` when the weekly
// sample is on (ADR-0033), which draws a week's acts done alone for a
// person to judge and suggests a level from them. An act done on a
// person's yes is kept with who said it, and never drawn: it is theirs.
// The engine's own move of a slipped milestone is not kept.

// Did is one act the role did.
type Did struct {
	Issue int    `yaml:"issue,omitempty"` // left out in the issue's own state
	Act   string `yaml:"act"`             // its kind, as the settings name it: close-duplicate, rename…
	Level string `yaml:"level"`           // the autonomy level it was done at
	Day   string `yaml:"day"`             // the day, YYYY-MM-DD, UTC
	Line  string `yaml:"line,omitempty"`
	Yes   string `yaml:"yes,omitempty"` // who accepted it: done on their yes, not alone
	// to, this run's only: the comment it wrote to the reporter, a question
	// or a proposal, found by its marker once posted, its line then linking
	// to it (Persist) — the question written once, there.
	to *comment
}

// comment is a comment to an issue's reporter: an ask, or a proposal to
// an outsider, and its round.
type comment struct {
	proposal bool
	round    int
}

// marker is what the comment carries, for the role.
func (c comment) marker(role string) string {
	if c.proposal {
		return ProposalMarker(role, c.Round())
	}
	return AskMarker(role, c.Round())
}

// Round is the comment's round, the first when it was not said.
func (c comment) Round() int { return max(c.round, 1) }

// Below is where a line of the role's comment says the comment it wrote
// to the reporter is, until it links to it.
const Below = "in the comment below"

// The acts kept: those of the last DidDays days, at most didMax over the
// backlog, each said in at most didLine characters.
const (
	DidDays = 35
	didMax  = 200
	didLine = 240
)

// recordDid keeps an act the role did, with today's day.
func (p *Plan) recordDid(c Proposal) {
	if c.Do == "keep" || c.Do == "undepend" || c.From != "" {
		return
	}
	now := time.Now().UTC()
	line := describe(c, "Closed")
	var to *comment
	switch {
	case c.Do == "refine" && !c.ToReporter:
		line = refined(c)
	case c.Do == "ask" || c.Do == "refine":
		// What it asked or proposed is in that comment, which notifies
		// them: said here, never written twice.
		line, to = wrote(c), &comment{proposal: c.Do == "refine", round: c.Round}
	}
	if utf8.RuneCountInString(line) > didLine {
		line = string([]rune(line)[:didLine-1]) + "…"
	}
	oldest := now.AddDate(0, 0, -DidDays).Format(dateLayout)
	p.Record.Did = slices.DeleteFunc(p.Record.Did, func(d Did) bool { return d.Day < oldest })
	p.Record.Did = append(p.Record.Did, Did{Issue: c.Issue, Act: c.Kind(), Level: p.config.Level, Day: now.Format(dateLayout), Line: line, Yes: c.Ticked, to: to})
	if n := len(p.Record.Did); n > didMax {
		p.Record.Did = p.Record.Did[n-didMax:]
	}
}

// wrote says, on the issue, that the role wrote to its reporter, and
// where: the question or the proposal is in that comment, not repeated.
func wrote(c Proposal) string {
	switch {
	case c.Do == "refine":
		return "Proposed sections to the reporter, who is outside the project, " + Below + ": nothing is written in the issue until they or a maintainer agree."
	case c.Round > 1:
		return "Asked the reporter again, after their answer, " + Below + "."
	}
	return "Asked the reporter a question, " + Below + "."
}

// Acts is what the weekly sample reads of the role's acts on the forge:
// those done alone each issue's state keeps, the level of the last, and
// the acts a person undid, with what shows it.
type Acts struct {
	Level  string
	Did    []Did
	undone map[int]string // the place in Did of an act undone → what shows it
}

// ReadActs reads the role's acts done alone from every issue's state, open
// or closed — one forge call an issue —, oldest first; nil when no issue
// holds any.
func ReadActs(f forge.Backlog, role string) (*Acts, error) {
	all, err := f.AllIssues()
	if err != nil {
		return nil, err
	}
	a := &Acts{undone: map[int]string{}}
	var undone []Undo
	for _, is := range all {
		notes, err := f.Notes(forge.Target{Kind: "issue", ID: is.ID})
		if err != nil {
			return nil, err
		}
		st, found, err := ReadState(forge.Bodies(notes), role)
		if !found || err != nil {
			continue
		}
		for _, d := range st.Did {
			if d.Yes == "" {
				d.Issue = is.ID
				a.Did = append(a.Did, d)
			}
		}
		for _, u := range st.Undone {
			u.Issue = is.ID
			undone = append(undone, u)
		}
		if st.Closed != "" && !is.Closed {
			undone = append(undone, Undo{Issue: is.ID, Act: st.Closed, Evidence: fmt.Sprintf("#%d reopened", is.ID)})
		}
	}
	if len(a.Did) == 0 {
		return nil, nil
	}
	slices.SortStableFunc(a.Did, func(x, y Did) int {
		if x.Day != y.Day {
			if x.Day < y.Day {
				return -1
			}
			return 1
		}
		return x.Issue - y.Issue
	})
	a.Level = a.Did[len(a.Did)-1].Level
	// Each undo counts against one act: the newest of its issue and kind
	// done by the day it was found.
	for _, u := range undone {
		for i := len(a.Did) - 1; i >= 0; i-- {
			d := a.Did[i]
			if d.Issue == u.Issue && d.Act == u.Act && (u.Day == "" || d.Day <= u.Day) {
				if _, taken := a.undone[i]; !taken {
					a.undone[i] = u.Evidence
				}
				break
			}
		}
	}
	return a, nil
}

// Undone says whether a person undid the i-th act, and what shows it.
func (a *Acts) Undone(i int) (string, bool) {
	why, ok := a.undone[i]
	return why, ok
}

// SuggestAfter is the acts done alone at a level before the sample
// suggests another.
const SuggestAfter = 10

// SuggestUndone is the share of the acts done alone, in percent, a person
// may undo before the sample suggests the level below.
const SuggestUndone = 10

// SuggestFromActs is the level the weekly sample suggests from the acts
// done alone at a level and those a person undid, and why; "" when none.
// Below SuggestAfter acts, none: too few to say. At cautious, the role does
// alone only what checks facts: nothing to suggest from.
func SuggestFromActs(level string, done, undone int) (string, string) {
	if done < SuggestAfter || level == Cautious {
		return "", ""
	}
	if undone*100 > SuggestUndone*done {
		below := map[string]string{Enterprising: Normal, Normal: Cautious}[level]
		if below == "" {
			return "", ""
		}
		return below, fmt.Sprintf("%d of the %d acts done alone at %s were undone (more than %d%%)", undone, done, level, SuggestUndone)
	}
	if level == Normal && undone == 0 {
		return Enterprising, fmt.Sprintf("none of the %d acts done alone at %s was undone", done, level)
	}
	return "", ""
}
