// Package inspectmark is what the inspector leaves on a merge request that
// another role reads: its triage of the gates' findings, kept as a record
// hidden in its one comment (ADR-0070 §6 e). The inspector writes with the
// same words; the developer reads them beside a gate's finding.
package inspectmark

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/JN0V/workline/internal/forge"
)

// StickyKey keys the inspector's one comment on a merge request, edited
// on each run, the record hidden in it.
const StickyKey = "triage"

// Marker is the comment's, as the engine writes it: it keys the
// inspector's comment on a merge request.
var Marker = forge.Marker("sticky=inspector/" + StickyKey)

// Found is one finding the gates report the change adds, as the inspector
// reads it from their result: placed, named by its fingerprint, which
// holds no line (ADR-0049, step 2).
type Found struct {
	ID          string `json:"id"` // t1 to triage now, r1 triaged before: what the task names it by
	Gate        string `json:"gate"`
	Check       string `json:"check"`
	Tool        string `json:"tool,omitempty"`
	Rule        string `json:"rule,omitempty"`
	Level       string `json:"level,omitempty"`
	Path        string `json:"path,omitempty"`
	Line        int    `json:"line,omitempty"`
	Message     string `json:"message,omitempty"`
	Fingerprint string `json:"fingerprint"`
}

// Where is the finding's place, `path:line`.
func (f Found) Where() string {
	if f.Line > 0 {
		return fmt.Sprintf("%s:%d", f.Path, f.Line)
	}
	return f.Path
}

// Name is the rule a finding is shown by, its tool beside it.
func (f Found) Name() string {
	n := f.Rule
	if n == "" {
		n = f.Check
	}
	if f.Tool != "" && f.Tool != n {
		return fmt.Sprintf("%s (%s)", n, f.Tool)
	}
	return n
}

// Triaged is one finding the agent triaged, kept by its fingerprint, which
// holds no line: while it only moves, it is not asked again (ADR-0013).
// At, the head it was triaged at; Model, who triaged it: what the weekly
// sample draws (ADR-0046).
type Triaged struct {
	Fingerprint   string `json:"fingerprint"`
	Gate          string `json:"gate,omitempty"`
	Check         string `json:"check"`
	Tool          string `json:"tool,omitempty"`
	Rule          string `json:"rule,omitempty"`
	Where         string `json:"where"`
	At            string `json:"at"`
	Model         string `json:"model,omitempty"`
	Matters       string `json:"matters"`
	Fix           string `json:"fix,omitempty"`
	FalsePositive bool   `json:"false-positive,omitempty"`
	Why           string `json:"why,omitempty"`
	Order         int    `json:"order,omitempty"`
}

func (t Triaged) RuleName() string {
	n := t.Rule
	if n == "" {
		n = t.Check
	}
	if t.Tool != "" && t.Tool != n {
		return fmt.Sprintf("%s (%s)", n, t.Tool)
	}
	return n
}

// Record is every finding triaged on the merge request, the oldest first:
// hidden in the comment, a line each; on a machine, in the git directory.
type Record struct {
	Triaged []Triaged `json:"triaged,omitempty"`
}

// Find is the finding of fingerprint the record holds, if any.
func (r Record) Find(fingerprint string) (Triaged, bool) {
	for _, t := range r.Triaged {
		if t.Fingerprint == fingerprint {
			return t, true
		}
	}
	return Triaged{}, false
}

// With is the record holding t, in place of one of its fingerprint.
func (r Record) With(t Triaged) Record {
	out := Record{}
	for _, o := range r.Triaged {
		if o.Fingerprint != t.Fingerprint {
			out.Triaged = append(out.Triaged, o)
		}
	}
	out.Triaged = append(out.Triaged, t)
	return out
}

// Ordered sets each finding's place at the head: the agent's, given;
// else the place it had, after those given; else the gates' order.
func (r Record) Ordered(found []Found, given map[string]int) Record {
	type place struct {
		given, had, at int
		fp             string
	}
	var places []place
	for i, f := range found {
		t, ok := r.Find(f.Fingerprint)
		if !ok {
			continue
		}
		p := place{given: given[f.Fingerprint], had: t.Order, at: i, fp: f.Fingerprint}
		places = append(places, p)
	}
	key := func(p place) (int, int, int) {
		if p.given > 0 {
			return 0, p.given, p.at
		}
		if p.had > 0 {
			return 1, p.had, p.at
		}
		return 2, 0, p.at
	}
	less := func(a, b place) bool {
		a1, a2, a3 := key(a)
		b1, b2, b3 := key(b)
		if a1 != b1 {
			return a1 < b1
		}
		if a2 != b2 {
			return a2 < b2
		}
		return a3 < b3
	}
	for i := 1; i < len(places); i++ {
		for j := i; j > 0 && less(places[j], places[j-1]); j-- {
			places[j], places[j-1] = places[j-1], places[j]
		}
	}
	rank := map[string]int{}
	for i, p := range places {
		rank[p.fp] = i + 1
	}
	out := Record{}
	for _, t := range r.Triaged {
		if n, ok := rank[t.Fingerprint]; ok {
			t.Order = n
		}
		out.Triaged = append(out.Triaged, t)
	}
	return out
}

var triagedLine = regexp.MustCompile(`<!-- workline:triaged (.*?) -->`)

// RecordOpening opens each line of the record: a text holding one that
// does not read holds a record damaged, never none.
const RecordOpening = "<!-- workline:triaged"

// ReadRecord reads the record a text holds; none when it holds no line.
func ReadRecord(text string) (Record, error) {
	var r Record
	lines := triagedLine.FindAllStringSubmatch(text, -1)
	for _, m := range lines {
		var t Triaged
		if err := json.Unmarshal([]byte(m[1]), &t); err != nil || t.Fingerprint == "" {
			return Record{}, fmt.Errorf("a line of the record does not read: %.80s", m[0])
		}
		r = r.With(t)
	}
	if len(lines) != strings.Count(text, RecordOpening) {
		return Record{}, errors.New("a line of the record is cut")
	}
	return r, nil
}

// ReadTriage is the record the inspector's comment holds: what the
// developer's review round reads beside a gate's finding, why it matters
// and how to fix it. Marker keys that comment.
func ReadTriage(text string) (Record, error) { return ReadRecord(text) }

// Lines is the record as the comment hides it: a line each, the oldest
// gone first past room, those of the findings at the head kept; past it
// still, the oldest of those too, counted: they are asked again.
func (r Record) Lines(keep map[string]bool, room int) (string, int) {
	all := make([]string, len(r.Triaged))
	size := 0
	for i, t := range r.Triaged {
		all[i] = "\n<!-- workline:triaged " + Hidden(t) + " -->"
		size += len(all[i])
	}
	gone, left := map[int]bool{}, 0
	for _, theirs := range []bool{false, true} {
		for i := 0; i < len(all) && size > room; i++ {
			if !gone[i] && keep[r.Triaged[i].Fingerprint] == theirs {
				gone[i] = true
				size -= len(all[i])
				if theirs {
					left++
				}
			}
		}
	}
	var b strings.Builder
	for i, l := range all {
		if !gone[i] {
			b.WriteString(l)
		}
	}
	return b.String(), left
}

// Hidden is a value as JSON on one line, safe in a comment hidden in
// Markdown: a dash written as its escape, and `<`, `>` escaped by the
// JSON library, so nothing the agent wrote ends the comment hiding it.
func Hidden(v any) string {
	data, _ := json.Marshal(v)
	return strings.ReplaceAll(string(data), "-", `\u002d`)
}
