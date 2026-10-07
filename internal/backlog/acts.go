package backlog

import (
	"errors"
	"fmt"
	"slices"
	"time"
	"unicode/utf8"

	"github.com/JN0V/workline/internal/forge"
)

// The acts the role did alone, for the weekly sample (ADR-0033): each kept
// in the record with its day and the autonomy level it was done at, read by
// `workline sample --apply`, which draws a week's for a person to judge and
// suggests a level from them. A person's tick and the engine's own move of
// a slipped milestone are not the role's choice: not kept.

// Did is one act the role did alone.
type Did struct {
	Issue int    `yaml:"issue"`
	Act   string `yaml:"act"`   // its kind, as the settings name it: close-duplicate, rename…
	Level string `yaml:"level"` // the autonomy level it was done at
	Day   string `yaml:"day"`   // the day, YYYY-MM-DD, UTC
	Line  string `yaml:"line,omitempty"`
}

// The acts the record keeps: those of the last DidDays days, at most
// didMax, each said in at most didLine characters — the record is one
// comment.
const (
	DidDays = 35
	didMax  = 200
	didLine = 120
)

// recordDid keeps an act the role decided to do alone, with today's day.
func (p *Plan) recordDid(c Proposal) {
	if c.Do == "keep" || c.From != "" || c.Ticked != "" {
		return
	}
	now := time.Now().UTC()
	line := describe(c, "Closed")
	if utf8.RuneCountInString(line) > didLine {
		line = string([]rune(line)[:didLine-1]) + "…"
	}
	oldest := now.AddDate(0, 0, -DidDays).Format(dateLayout)
	p.Record.Did = slices.DeleteFunc(p.Record.Did, func(d Did) bool { return d.Day < oldest })
	p.Record.Did = append(p.Record.Did, Did{Issue: c.Issue, Act: c.Kind(), Level: p.config.Level, Day: now.Format(dateLayout), Line: line})
	if n := len(p.Record.Did); n > didMax {
		p.Record.Did = p.Record.Did[n-didMax:]
	}
}

// Acts is what the weekly sample reads of the role's acts on the forge:
// those its record keeps, the level in force at its last run, and the
// acts a person undid, with what shows it.
type Acts struct {
	Report int
	Level  string
	Did    []Did
	undone map[int]string // the place in Did of an act undone → what shows it
}

// ErrRecordBroken is a record on the report that does not read.
var ErrRecordBroken = errors.New("the record of what the role did does not read")

// ReadActs reads the role's acts from the record on its report: nil when
// the project has no report; an error when its record does not read.
func ReadActs(f forge.Backlog, role string) (*Acts, error) {
	open, err := f.Issues()
	if err != nil {
		return nil, err
	}
	h, err := ReadHand(f, role, open)
	if err != nil {
		return nil, err
	}
	if h.Report == 0 {
		return nil, nil
	}
	if h.Broken != nil {
		return nil, fmt.Errorf("%w on #%d: %w", ErrRecordBroken, h.Report, h.Broken)
	}
	a := &Acts{Report: h.Report, Did: h.Record.Did}
	if m := h.Record.Measure; m != nil && m.Level != "" {
		a.Level = m.Level
	} else if n := len(a.Did); n > 0 {
		a.Level = a.Did[n-1].Level
	} else {
		a.Level = Normal
	}
	isOpen := map[int]bool{}
	for _, is := range open {
		isOpen[is.ID] = true
	}
	a.undone = undoneActs(h.Record, isOpen, h.Undone)
	return a, nil
}

// undoneActs maps each act of the record a person undid, by its place in
// Did, to what shows it: a closing found wrong, or reopened now (isOpen);
// an undo the record keeps, or found now. Each undo counts against one
// act: the newest of its issue and kind done by the day it was found — one
// found before the record kept days, or by this read, against the newest.
func undoneActs(r Record, isOpen map[int]bool, found []Undo) map[int]string {
	out := map[int]string{}
	undo := func(issue int, kind, day, evidence string) {
		for i := len(r.Did) - 1; i >= 0; i-- {
			d := r.Did[i]
			if d.Issue == issue && d.Act == kind && (day == "" || d.Day <= day) {
				if _, taken := out[i]; !taken {
					out[i] = evidence
				}
				return
			}
		}
	}
	for _, c := range r.Wrong {
		undo(c.Issue, c.Act, "", fmt.Sprintf("#%d reopened", c.Issue))
	}
	for _, c := range r.Closed {
		if isOpen[c.Issue] {
			undo(c.Issue, c.Act, "", fmt.Sprintf("#%d reopened", c.Issue))
		}
	}
	for _, u := range r.Undone {
		undo(u.Issue, u.Act, u.Day, u.Evidence)
	}
	for _, u := range found {
		undo(u.Issue, u.Act, "", u.Evidence)
	}
	return out
}

// SuggestFromRecord is the level the acts done alone suggest, as the
// weekly sample counts them, over every act the record keeps at the level
// in force: for the role's own report, beside the suggestion from the
// proposals settled (ADR-0033). A run's record holds its closings found
// wrong and its undos found already.
func SuggestFromRecord(r Record) (string, string) {
	if r.Measure == nil || r.Measure.Level == "" {
		return "", ""
	}
	undone := undoneActs(r, nil, nil)
	done, undid := 0, 0
	for i, d := range r.Did {
		if d.Level != r.Measure.Level {
			continue
		}
		done++
		if _, ok := undone[i]; ok {
			undid++
		}
	}
	return SuggestFromActs(r.Measure.Level, done, undid)
}

// Undone says whether a person undid the record's i-th act, and what
// shows it.
func (a *Acts) Undone(i int) (string, bool) {
	why, ok := a.undone[i]
	return why, ok
}

// SuggestUndone is the share of the acts done alone, in percent, a person
// may undo before the sample suggests the level below.
const SuggestUndone = 10

// SuggestFromActs is the level the weekly sample suggests from the acts
// done alone at a level and those a person undid, and why; "" when none.
// Below SuggestAfter acts, none: too few to say. At cautious, the role does
// alone only what checks facts: the report suggests from the proposals.
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
