package scale

// An issue's priority and effort are read by name (ADR-0040): a level
// counts for what its name says, never for where an option sits in a
// list, through one table — the one place in the engine that names a
// level, a label carrying one, or the way the role says one. Where a
// project keeps each axis is detected when its settings ask (detect.go,
// ADR-0040's step 2); unset, the engine reads and writes workline's own
// priority labels, and reads no effort.

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Scale is an axis's levels in the order's direction: level 1 is the
// first name, the most pressing priority or the lightest effort, the
// last name the last level; 0 is none.
type Scale struct {
	Axis  string
	Names []string
	// None is where an issue with no value ranks: after the last
	// priority; an effort not estimated as the middle one.
	None int
	// words are the names read for each level, the level's own first, in
	// lower case: a project's or a forge's word for it.
	words [][]string
}

// The two scales (ADR-0040, "Two axes, the forge's levels, read by
// name").
var (
	Priority = Scale{Axis: "priority", Names: []string{"Urgent", "High", "Medium", "Low"}, None: 5, words: [][]string{
		{"urgent", "critical", "critical-urgent"},
		{"high", "important", "important-soon"},
		{"medium", "important-longterm"},
		{"low", "backlog"},
	}}
	Effort = Scale{Axis: "effort", Names: []string{"Low", "Medium", "High"}, None: 2, words: [][]string{
		{"low", "small", "hours"},
		{"medium", "days"},
		{"high", "large", "x-large", "weeks"},
	}}
)

// Count is the scale's number of levels.
func (s Scale) Count() int { return len(s.Names) }

// Valid says whether l is one of the scale's levels.
func (s Scale) Valid(l int) bool { return l >= 1 && l <= s.Count() }

// Name is a level's name; "" for none.
func (s Scale) Name(l int) string {
	if !s.Valid(l) {
		return ""
	}
	return s.Names[l-1]
}

// Rank is where a level sorts: the level itself, or None for none.
func (s Scale) Rank(l int) int {
	if !s.Valid(l) {
		return s.None
	}
	return l
}

// Word is the level a word names, whatever its case: a level's own name,
// or one a project or a forge uses for it (critical, small, weeks); ok is
// false when no table knows it.
func (s Scale) Word(name string) (int, bool) {
	name = strings.ToLower(strings.TrimSpace(name))
	for i, ws := range s.words {
		if slices.Contains(ws, name) {
			return i + 1, true
		}
	}
	return 0, false
}

// Number is the level of the number n in a family numbered from first, 0
// or 1: first is the first level, each next number the next one, every
// number past the last level the last (P0 to P5: Urgent, High, Medium,
// then Low thrice); 0 below first.
func (s Scale) Number(n, first int) int {
	if n < first {
		return 0
	}
	return min(n-first+1, s.Count())
}

// Label is one name of a family and the level it carries. Color, About
// and Rank, for a label the role creates: its colour, what it says of
// itself, and its rank where the forge ranks labels — GitLab's
// prioritized labels, 1 listed first —, 0 for none.
type Label struct {
	Name  string
	Level int
	Color string
	About string
	Rank  int
}

// Describe is what a label the role creates says of itself, by the role
// named.
func (l Label) Describe(by string) string {
	return l.About + ": set by the " + by + " or a person"
}

// Family is where a project keeps an axis: labels that carry its levels,
// each read by its name, or a field's options. Prefix starts each label
// of a family of labels; "" for a field's options. Its labels are in the
// order of their levels: the first of a level is the one written.
type Family struct {
	Scale  Scale
	Prefix string
	Labels []Label
}

// Of is the level a name of the family carries; 0 when it carries none.
func (f Family) Of(name string) int {
	for _, l := range f.Labels {
		if l.Name == name {
			return l.Level
		}
	}
	return 0
}

// Levels are the levels names carry in the family, in the names' order;
// a name outside it carries none.
func (f Family) Levels(names []string) []int {
	var out []int
	for _, n := range names {
		if l := f.Of(n); l > 0 {
			out = append(out, l)
		}
	}
	return out
}

// First is the first level in the order's direction names carry — the
// most pressing priority, the lightest effort —; 0 when they carry none.
func (f Family) First(names []string) int {
	set := f.Levels(names)
	if len(set) == 0 {
		return 0
	}
	return slices.Min(set)
}

// Label is the label written for level l: the family's first carrying
// it; ok is false when none does.
func (f Family) Label(l int) (Label, bool) {
	for _, x := range f.Labels {
		if x.Level == l {
			return x, true
		}
	}
	return Label{}, false
}

// Others are the family's names but the one written for level l: those
// taken off when it is set.
func (f Family) Others(l int) []string {
	w, _ := f.Label(l)
	var out []string
	for _, x := range f.Labels {
		if x.Name != w.Name {
			out = append(out, x.Name)
		}
	}
	return out
}

// Labels reads a project's family of labels by name: each label starting
// with prefix, its rest a number or a word. Numbers count from the
// family's own start, 0 when it has a 0, else 1 (Scale.Number): a family
// created a label at a time, its first label never made, still reads its
// second as the second level. A prefix ending in a letter or a digit
// takes numbers only — P0, effort1: hours —, so P-critical belongs to
// another family and Performance to none. A word no table knows carries
// no level, and the error names it, to be said once.
func Labels(s Scale, prefix string, labels []string) (Family, error) {
	return named(s, prefix, labels, nil)
}

// named reads a family of labels as Labels does, a name the project's
// settings map (Setting.Names) carrying the level they give it.
func named(s Scale, prefix string, labels []string, names map[string]int) (Family, error) {
	type read struct {
		name   string
		number int
		word   string
		level  int // mapped by the settings
	}
	var in []read
	last, _ := utf8.DecodeLastRuneInString(prefix)
	numbersOnly := prefix != "" && isAlnum(last)
	first := 1
	for _, name := range labels {
		rest, ok := strings.CutPrefix(name, prefix)
		if !ok || rest == "" {
			continue
		}
		if l := names[name]; s.Valid(l) {
			in = append(in, read{name: name, number: -1, level: l})
			continue
		}
		if n, ok := number(rest); ok {
			in = append(in, read{name: name, number: n})
			first = min(first, n)
			continue
		}
		if numbersOnly {
			continue
		}
		in = append(in, read{name: name, number: -1, word: rest})
	}
	// Numbers in their order, P9 before P10: the lowest of a level is the
	// one written; words as they came.
	slices.SortStableFunc(in, func(a, b read) int { return a.number - b.number })
	f := Family{Scale: s, Prefix: prefix}
	var unknown []string
	for _, r := range in {
		l := r.level
		if r.number >= 0 {
			l = s.Number(r.number, first)
		} else if w, ok := s.Word(r.word); ok {
			l = w
		}
		if l == 0 {
			unknown = append(unknown, r.name)
			continue
		}
		f.Labels = append(f.Labels, Label{Name: r.name, Level: l})
	}
	f.sort()
	return f, unknownErr(s, unknown)
}

// Options reads a field's options by name, never by their position: a
// forge lists them as its owner arranged them, an Effort field heaviest
// first. An option no table knows carries no level, and the error names
// it, to be said once.
func Options(s Scale, options []string) (Family, error) {
	f := Family{Scale: s}
	var unknown []string
	for _, o := range options {
		l, ok := s.Word(o)
		if !ok {
			unknown = append(unknown, o)
			continue
		}
		f.Labels = append(f.Labels, Label{Name: o, Level: l})
	}
	f.sort()
	return f, unknownErr(s, unknown)
}

// sort puts a family's labels in the order of their levels, those of one
// level as they came.
func (f *Family) sort() {
	slices.SortStableFunc(f.Labels, func(a, b Label) int { return a.Level - b.Level })
}

func unknownErr(s Scale, names []string) error {
	if len(names) == 0 {
		return nil
	}
	q := make([]string, len(names))
	for i, n := range names {
		q[i] = strconv.Quote(n)
	}
	return fmt.Errorf("%s: %s names no level (%s): read as none", s.Axis, strings.Join(q, ", "), strings.Join(s.Names, ", "))
}

// number reads a label's rest as a number: its digits, then nothing or a
// text after a separator ("1: hours").
func number(rest string) (int, bool) {
	end := strings.IndexFunc(rest, func(r rune) bool { return r < '0' || r > '9' })
	if end == 0 {
		return 0, false
	}
	if end < 0 {
		end = len(rest)
	} else if r, _ := utf8.DecodeRuneInString(rest[end:]); isAlnum(r) {
		return 0, false
	}
	n, err := strconv.Atoi(rest[:end])
	return n, err == nil
}

func isAlnum(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }

// Weight is the effort level of a GitLab weight, a whole number: 0 or 1
// Low, 2 or 3 Medium, 4 or more High; none for a weight below 0.
func Weight(w int) int {
	switch {
	case w < 0:
		return 0
	case w <= 1:
		return 1
	case w <= 3:
		return 2
	}
	return 3
}

// WeightOf is the weight written for an effort level: 1, 3 or 5, on both
// common point scales (1, 2, 3, 5, 8 and 1 to 5); 0 for none.
func WeightOf(l int) int {
	if !Effort.Valid(l) {
		return 0
	}
	return 2*l - 1
}

// Own is workline's own priority labels, workline:priority/1, the most
// pressing, to /4, the most pressing the warmest: the family the role
// writes until ADR-0040's detection finds a project's.
var Own = own()

func own() Family {
	colors := []string{"b60205", "d93f0b", "fbca04", "c5def5"}
	f := Family{Scale: Priority, Prefix: "workline:priority/"}
	for l := 1; l <= Priority.Count(); l++ {
		f.Labels = append(f.Labels, Label{Name: f.Prefix + strconv.Itoa(l), Level: l, Color: colors[l-1],
			About: fmt.Sprintf("Priority %d of %d, 1 the most pressing", l, Priority.Count())})
	}
	return f
}

// Scheme is where the engine reads each axis on a project.
type Scheme struct {
	Priority Source
	Effort   Source
	// Detected: the project's settings asked where it keeps them
	// (ADR-0040): each value is then set once, only where empty, and Next
	// shows each issue's cell of the grid.
	Detected bool
}

// Before is the scheme before ADR-0040's detection, a project's setting
// unset: workline's own priority labels; no effort, read nowhere.
var Before = Scheme{Priority: Source{Family: Own, Said: ownSaid}, Effort: Source{Family: Family{Scale: Effort}}}

// Used is the scheme the engine reads and writes issues with: Before,
// until Use sets the one detected on the project a process reads.
var Used = Before

// Use sets the scheme the engine reads and writes issues with, detected
// on the project this process reads (Detect): a process reads one.
func Use(s Scheme) { Used = s }

// Value is a priority level as the role's texts say it: its number, 1
// the most pressing, as workline's own labels write it, and the label
// that carries it when the project keeps its own ("1 (P0)"); "none" for
// none.
func Value(l int) string {
	if !Priority.Valid(l) {
		return "none"
	}
	if f := Used.Priority.Family; f.Prefix != Own.Prefix {
		if label, ok := f.Label(l); ok {
			return fmt.Sprintf("%d (%s)", l, label.Name)
		}
	}
	return strconv.Itoa(l)
}

// Said is a priority level in a sentence: "priority 2"; "no priority" for
// none.
func Said(l int) string {
	if !Priority.Valid(l) {
		return "no priority"
	}
	return "priority " + Value(l)
}

// EffortValue is an effort level as the role's texts say it: its name,
// and the label that carries it ("Low (effort/small)"); "not estimated"
// for none.
func EffortValue(l int) string {
	if !Effort.Valid(l) {
		return "not estimated"
	}
	if label, ok := Used.Effort.Family.Label(l); ok {
		return fmt.Sprintf("%s (%s)", Effort.Name(l), label.Name)
	}
	return Effort.Name(l)
}

// EffortAnchor is what an effort level means to build: hours, days,
// weeks; "" for none.
func EffortAnchor(l int) string {
	if !Effort.Valid(l) {
		return ""
	}
	return anchors[l-1]
}

// cells name four corners of the value-vs-effort grid (ADR-0040, "The
// order"), by priority then effort level.
var cells = map[[2]int]string{{2, 1}: "quick win", {2, 3}: "big bet", {4, 1}: "fill-in", {4, 3}: "time sink"}

// PriorityName is a priority level by its name; "no priority" for none.
func PriorityName(l int) string {
	if !Priority.Valid(l) {
		return "no priority"
	}
	return Priority.Name(l)
}

// Cell is an issue's place in the value-vs-effort grid, as Next says it:
// "High · Effort Low → quick win"; a value missing said so, "no
// priority, effort not estimated".
func Cell(p, e int) string {
	prio, eff := PriorityName(p), "effort not estimated"
	if Effort.Valid(e) {
		eff = "Effort " + Effort.Name(e)
	}
	if !Priority.Valid(p) && !Effort.Valid(e) {
		return prio + ", " + eff
	}
	if name, ok := cells[[2]int{p, e}]; ok {
		return prio + " · " + eff + " → " + name
	}
	return prio + " · " + eff
}

// Stuck says an axis's estimates stuck on one value (ADR-0040): past 10
// values set, one over 60% of them, said "Medium on 8 of the 11"; "" when
// not.
func Stuck(s Scale, levels []int) string {
	if len(levels) <= 10 {
		return ""
	}
	count := map[int]int{}
	for _, l := range levels {
		count[l]++
	}
	for l := 1; l <= s.Count(); l++ {
		if count[l]*100 > 60*len(levels) {
			return fmt.Sprintf("%s on %d of the %d", s.Name(l), count[l], len(levels))
		}
	}
	return ""
}
