package backlog

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/verdict"
)

// What became of an item of a file an import read, as its map says
// (docs/spec/backlog-acts.md, "Importing a file").
const (
	ItemOpened      = "opened"       // opened by this import
	ItemWouldOpen   = "would-open"   // without --apply: would be opened
	ItemAlreadyOpen = "already-open" // an open issue held it before the import
	ItemClosed      = "closed"       // a closed issue holds it: not opened again
	ItemPastCap     = "past-cap"     // proposed in the report, past the cap of open; the import run again opens it
	ItemDone        = "done"         // judged done, the words that say so quoted
	ItemNotItem     = "not-item"     // not a requirement: an introduction, a history, a heading
)

// Skip is an item of a file an import read and does not open, and why, as
// the agent answers it: done, the words that say so quoted; held, the
// issue that holds it; not-item, why.
type Skip struct {
	Lines  any    `yaml:"lines"` // "12", or "12-14"
	Reason string `yaml:"reason"`
	Quote  *Quote `yaml:"quote,omitempty"`
	Issue  int    `yaml:"issue,omitempty"`
	Why    string `yaml:"why,omitempty"`
}

// ImportOpen is an item the agent proposed to open, and what the engine
// decided of it.
type ImportOpen struct {
	Title   string
	Quote   Quote
	Capped  bool // proposed in the report, past the cap of open
	Dropped bool // refused by the engine, a finding saying why
	// Opening is what opening it did, as the run recorded it: the issue
	// that holds it, and whether it was opened then.
	Opening *Recorded
}

// Recorded is what one opening of a run did (RecordOpening).
type Recorded struct {
	Index   int    `yaml:"index"` // the intention's place in the run
	Outcome string `yaml:"outcome"`
	Issue   int    `yaml:"issue"`
}

// openingsFile keeps, in a run folder, what each opening did.
const openingsFile = "openings.yaml"

// RecordOpening adds what opening the intention at index did to the run
// folder: a forge's list may not show an issue opened a moment ago, and an
// import's map reads it from here.
func RecordOpening(runDir string, index int, outcome string, issue int) error {
	f, err := os.OpenFile(filepath.Join(runDir, "out", openingsFile), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "- {index: %d, outcome: %s, issue: %d}\n", index, outcome, issue)
	return err
}

// RecordedOpenings reads what the openings of a run did, by intention index.
func RecordedOpenings(runDir string) map[int]Recorded {
	out := map[int]Recorded{}
	data, err := os.ReadFile(filepath.Join(runDir, "out", openingsFile))
	if err != nil {
		return out
	}
	var all []Recorded
	if yaml.Unmarshal(data, &all) == nil {
		for _, r := range all {
			out[r.Index] = r
		}
	}
	return out
}

// ImportShare is one share of the file an import read, and the agent's
// answer on it.
type ImportShare struct {
	From, To int // the lines it was given
	// Own is its last line to answer for: those after it are read again by
	// the next share, which answers for them.
	Own   int
	Opens []ImportOpen
	Skips []Skip
}

// Mapped is one entry of an import's map: an item, by its lines and first
// words, and the issue that holds it or why none does.
type Mapped struct {
	Lines string `json:"lines"`
	From  int    `json:"from"`
	To    int    `json:"to"`
	Words string `json:"words"`
	State string `json:"state,omitempty"`
	Issue int    `json:"issue,omitempty"`
	Why   string `json:"why,omitempty"`
}

// Coverage is an import's map: every item of the file to its issue or its
// reason, and the lines left with neither.
type Coverage struct {
	File       string   `json:"file"`
	Items      []Mapped `json:"items"`
	NotCovered []Mapped `json:"not-covered"`
}

// Cover builds an import's map from the shares the engine cut and the
// agent's answers, checking each reason: a quote found in the file, an
// issue that is there. Before and after are the forge's issues, open and
// closed, before the import and after it; applied, whether it wrote. A line
// not blank that no item nor reason holds is not covered, and the share
// that was to answer for it is flagged.
func Cover(repo, file string, lines []string, shares []ImportShare, before, after []forge.Issue, applied bool) (*Coverage, []verdict.Finding) {
	if n := len(lines); n > 0 && lines[n-1] == "" {
		lines = lines[:n-1]
	}
	c := &Coverage{File: file, Items: []Mapped{}, NotCovered: []Mapped{}}
	var findings []verdict.Finding
	existed := map[int]bool{}
	for _, is := range before {
		existed[is.ID] = true
	}
	byID := map[int]forge.Issue{}
	for _, is := range after {
		byID[is.ID] = is
	}
	holder := func(key string) *forge.Issue {
		var closed *forge.Issue
		for i, is := range after {
			if !strings.Contains(is.Body, forge.Marker(key)) {
				continue
			}
			if !is.Closed {
				return &after[i]
			}
			if closed == nil {
				closed = &after[i]
			}
		}
		return closed
	}
	held := func(m *Mapped, is forge.Issue) {
		m.Issue, m.State = is.ID, ItemAlreadyOpen
		switch {
		case is.Closed:
			m.State = ItemClosed
		case !existed[is.ID]:
			m.State = ItemOpened
		}
	}
	var items []Mapped
	add := func(m Mapped) {
		m.Words = firstWords(lines[m.From-1])
		items = append(items, m)
	}
	for _, s := range shares {
		for _, o := range s.Opens {
			if o.Quote.Path != file {
				continue
			}
			from, to, _, ok := LocateIn(lines, o.Quote.Text)
			if !ok {
				continue // dropped by the engine: no-quote
			}
			m := Mapped{Lines: span(from, to), From: from, To: to}
			switch is := holder(ImportKey(o.Quote)); {
			case o.Opening != nil && o.Opening.Issue > 0:
				// As the run recorded it: a forge's list may lag behind.
				m.Issue, m.State = o.Opening.Issue, ItemAlreadyOpen
				switch {
				case o.Opening.Outcome == Settled || o.Opening.Outcome == FoundAgain:
					m.State = ItemClosed
				case !existed[o.Opening.Issue]:
					m.State = ItemOpened // by this run, or an earlier share of it
				}
			case is != nil:
				held(&m, *is) // dropped as already open, or opened
			case o.Dropped:
				continue // refused, a finding saying why: not covered
			case o.Capped:
				m.State, m.Why = ItemPastCap, "proposed in the report: the import run again opens it"
			case !applied:
				m.State = ItemWouldOpen
			default:
				continue
			}
			add(m)
		}
		for _, k := range s.Skips {
			from, to, ok := parseLines(fmt.Sprint(k.Lines), len(lines))
			where := fmt.Sprintf("%s:%v", file, k.Lines)
			if !ok {
				findings = append(findings, verdict.Finding{Rule: "skip-unread", Where: where,
					Message: fmt.Sprintf("a skip names its lines in the file, 1 to %d: %q does not read", len(lines), fmt.Sprint(k.Lines))})
				continue
			}
			m := Mapped{Lines: span(from, to), From: from, To: to}
			switch k.Reason {
			case "done":
				q := Quote{Path: file}
				if k.Quote != nil {
					q = *k.Quote
					if q.Path == "" {
						q.Path = file
					}
				}
				found := false
				if strings.TrimSpace(q.Text) != "" {
					if q.Path == file {
						_, _, _, found = LocateIn(lines, q.Text)
					} else {
						_, _, _, found = Locate(repo, q.Path, q.Text)
					}
				}
				if !found {
					findings = append(findings, verdict.Finding{Rule: "no-quote", Where: where,
						Message: fmt.Sprintf("judged done, but the words quoted to say so are not found in %s: the item is not covered", q.Path)})
					continue
				}
				m.State, m.Why = ItemDone, clip(squeeze(q.Text), 120)
				if q.Path != file {
					m.Why = q.Path + ": " + m.Why
				}
			case "held":
				is, ok := byID[k.Issue]
				if !ok {
					findings = append(findings, verdict.Finding{Rule: "skip-unread", Where: where,
						Message: fmt.Sprintf("held by #%d, which is not an issue of this forge: the item is not covered", k.Issue)})
					continue
				}
				held(&m, is)
				if m.State == ItemOpened {
					m.State = ItemAlreadyOpen // named by the agent, not opened by the import
				}
				m.Why = "the agent says this issue holds it"
			case "not-item":
				if strings.TrimSpace(k.Why) == "" {
					findings = append(findings, verdict.Finding{Rule: "skip-unread", Where: where,
						Message: "not an item, but why is not said: the lines are not covered"})
					continue
				}
				m.State, m.Why = ItemNotItem, clip(squeeze(k.Why), 120)
			default:
				findings = append(findings, verdict.Finding{Rule: "skip-unread", Where: where,
					Message: fmt.Sprintf("a skip's reason is done, held or not-item, not %q", k.Reason)})
				continue
			}
			add(m)
		}
	}
	// One entry an item: read in two shares, the issue's word kept over a
	// reason, the first answer over a later one.
	rank := map[string]int{ItemOpened: 0, ItemAlreadyOpen: 0, ItemClosed: 0, ItemPastCap: 1, ItemWouldOpen: 1, ItemDone: 2, ItemNotItem: 3}
	covered := make([]bool, len(lines)+1)
	best := map[[2]int]int{}
	for _, m := range items {
		for l := m.From; l <= m.To; l++ {
			covered[l] = true
		}
		k := [2]int{m.From, m.To}
		if i, ok := best[k]; ok {
			if rank[m.State] < rank[c.Items[i].State] {
				c.Items[i] = m
			}
			continue
		}
		best[k] = len(c.Items)
		c.Items = append(c.Items, m)
	}
	slices.SortStableFunc(c.Items, func(a, b Mapped) int { return a.From - b.From })
	// The lines no answer holds, a paragraph of them an entry, each listed
	// under the share that was to answer for it.
	omitted := map[int][]string{}
	for l := 1; l <= len(lines); l++ {
		if covered[l] || strings.TrimSpace(lines[l-1]) == "" {
			continue
		}
		to := l
		for to < len(lines) && !covered[to+1] && strings.TrimSpace(lines[to]) != "" {
			to++
		}
		c.NotCovered = append(c.NotCovered, Mapped{Lines: span(l, to), From: l, To: to, Words: firstWords(lines[l-1])})
		for i, s := range shares {
			if l >= s.From && (l <= s.Own || i == len(shares)-1) {
				omitted[i] = append(omitted[i], span(l, to))
				break
			}
		}
		l = to
	}
	for i, s := range shares {
		if len(omitted[i]) == 0 {
			continue
		}
		findings = append(findings, verdict.Finding{Rule: "items-omitted", Where: fmt.Sprintf("lines %d to %d", s.From, s.To),
			Message: fmt.Sprintf("the answer for this share names no item nor reason for lines %s: listed under Not covered", strings.Join(omitted[i], ", "))})
	}
	return c, findings
}

// span writes lines from to to as the map does: "12", or "12-14".
func span(from, to int) string {
	if to > from {
		return fmt.Sprintf("%d-%d", from, to)
	}
	return strconv.Itoa(from)
}

// parseLines reads a skip's lines, "12" or "12-14", within a file of n.
func parseLines(s string, n int) (from, to int, ok bool) {
	a, b, two := strings.Cut(strings.TrimSpace(s), "-")
	from, err := strconv.Atoi(strings.TrimSpace(a))
	if err != nil {
		return 0, 0, false
	}
	to = from
	if two {
		if to, err = strconv.Atoi(strings.TrimSpace(b)); err != nil {
			return 0, 0, false
		}
	}
	return from, to, from >= 1 && to >= from && to <= n
}

// firstWords are the words an item starts with, for a person to find it.
func firstWords(line string) string {
	return clip(squeeze(line), 60)
}

func clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// Text writes the map for a person: each item to its issue or its reason,
// then those not covered.
func (c *Coverage) Text() string {
	var b strings.Builder
	fmt.Fprintf(&b, "The map of %s: %d items\n", c.File, len(c.Items))
	for _, m := range c.Items {
		fmt.Fprintf(&b, "  line %-9s %-62s → %s\n", m.Lines, quoteWords(m.Words), m.say())
	}
	if len(c.NotCovered) > 0 {
		fmt.Fprintf(&b, "Not covered: %d, no issue and no reason — read them, then run the import again or open them by hand\n", len(c.NotCovered))
		for _, m := range c.NotCovered {
			fmt.Fprintf(&b, "  line %-9s %s\n", m.Lines, quoteWords(m.Words))
		}
	}
	return b.String()
}

// Entries are the map's lines for a job's summary: each item to its issue
// or its reason, then each item not covered, said so.
func (c *Coverage) Entries() []string {
	var out []string
	for _, m := range c.Items {
		out = append(out, fmt.Sprintf("line %s %s → %s", m.Lines, quoteWords(m.Words), m.say()))
	}
	for _, m := range c.NotCovered {
		out = append(out, fmt.Sprintf("line %s %s → not covered: no issue, no reason", m.Lines, quoteWords(m.Words)))
	}
	return out
}

func quoteWords(w string) string { return "\"" + w + "\"" }

func (m Mapped) say() string {
	switch m.State {
	case ItemOpened:
		return fmt.Sprintf("#%d, opened", m.Issue)
	case ItemAlreadyOpen:
		return fmt.Sprintf("#%d, already open", m.Issue)
	case ItemClosed:
		return fmt.Sprintf("#%d, closed", m.Issue)
	case ItemWouldOpen:
		return "would be opened"
	case ItemPastCap:
		return "past the cap: " + m.Why
	case ItemDone:
		return "done: \"" + m.Why + "\""
	case ItemNotItem:
		return "not an item: " + m.Why
	}
	return m.State
}
