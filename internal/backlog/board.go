package backlog

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/forge"
)

// The report opens with what is next and what is stuck (ADR-0031): the
// first ready issues of the order, and each issue waiting on a person past
// a number of days, with since when — rebuilt at every run from the forge,
// nothing of it stored but the day a proposal was first made.

// The defaults and bounds of `next-max` and `stuck-days`.
const (
	NextMax        = 5
	NextMaxLimit   = 20
	StuckDays      = 14
	StuckDaysLimit = 365
)

// What a stuck issue waits on, in the order an issue is said for the
// first: an issue appears once.
const (
	WaitsReady    = "ready"    // ready, nothing started since
	WaitsAsked    = "asked"    // its reporter's answer
	WaitsProposed = "proposed" // a person's tick on a proposal of the report
	WaitsObsolete = "obsolete" // a second judge, its announcement due
)

var waitsOrder = []string{WaitsReady, WaitsAsked, WaitsProposed, WaitsObsolete}

// Wait is an issue waiting on a person, and since when.
type Wait struct {
	Issue int    `yaml:"issue"`
	Waits string `yaml:"waits"`
	Since string `yaml:"since"`          // the day it started waiting, YYYY-MM-DD
	Act   string `yaml:"act,omitempty"`  // proposed: the kind of act
	Line  string `yaml:"line,omitempty"` // proposed with no issue (an issue to open): its line
}

// Day reads a forge's time — RFC 3339 or YYYY-MM-DD — as its day in UTC;
// "" when it does not read.
func Day(s string) string {
	if t, ok := moment(s); ok {
		return t.UTC().Format(dateLayout)
	}
	return ""
}

// moment reads a forge's time, RFC 3339 or YYYY-MM-DD.
func moment(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, true
	}
	if t, err := time.Parse(dateLayout, s); err == nil {
		return t, true
	}
	return time.Time{}, false
}

// Days is how many days lie between a day and now's, in UTC.
func Days(day string, now time.Time) int {
	t, err := time.Parse(dateLayout, day)
	if err != nil {
		return 0
	}
	today, _ := time.Parse(dateLayout, now.UTC().Format(dateLayout))
	return int(today.Sub(t).Hours() / 24)
}

// ReadyWait is a ready issue's wait, from its trail: since the day it last
// got the label, when no pull or merge request nor commit named it since;
// known false when the forge does not say that day.
func ReadyWait(id int, t forge.Trail) (w *Wait, known bool) {
	labeled, ok := moment(t.Labeled)
	if !ok {
		return nil, false
	}
	for _, l := range t.Links {
		if at, ok := moment(l.At); ok && !at.Before(labeled) {
			return nil, true // something started since
		}
	}
	return &Wait{Issue: id, Waits: WaitsReady, Since: Day(t.Labeled)}, true
}

// AskedWait is an issue's wait on its reporter: since the last round
// written to them, when no person wrote after it; known false when the
// forge does not say that round's day.
func AskedWait(id int, notes []forge.Note, role string) (w *Wait, known bool) {
	if e := ReadExchange(forge.Bodies(notes), role); e.Rounds == 0 || e.Answered {
		return nil, true
	}
	for i := len(notes) - 1; i >= 0; i-- {
		if Round(notes[i].Body, role) {
			if day := Day(notes[i].Created); day != "" {
				return &Wait{Issue: id, Waits: WaitsAsked, Since: day}, true
			}
			return nil, false
		}
	}
	return nil, true
}

// ObsoleteWait is an announced issue's wait on a second judge: since the
// day its delay ended, while due and neither closed nor kept.
func ObsoleteWait(id int, o Obsolete) *Wait {
	if o.Announcement == nil || o.Keep != "" || !o.Due || o.From == "" {
		return nil
	}
	return &Wait{Issue: id, Waits: WaitsObsolete, Since: o.From}
}

// Next lists the first n issues of the order bearing workline:ready that
// wait on no open issue and have no parts, the report left out: where to
// start (ADR-0028, ADR-0031).
func Next(open []forge.Issue, report, n int) []forge.Issue {
	var list []forge.Issue
	isOpen := map[int]bool{}
	for _, is := range open {
		if is.ID != report {
			list = append(list, is)
			isOpen[is.ID] = true
		}
	}
	Order(list)
	var out []forge.Issue
	for _, is := range list {
		if len(out) >= n {
			break
		}
		if Offered(is, isOpen) {
			out = append(out, is)
		}
	}
	return out
}

// Offered says whether an issue may be offered to whoever builds next:
// ready, waiting on no open issue, no parent (ADR-0028, ADR-0029).
func Offered(is forge.Issue, open map[int]bool) bool {
	return slices.Contains(is.Labels, LabelReady) && len(Waiting(is, open)) == 0 && len(Parts(is)) == 0
}

// Board is the report's opening: what is next and what is stuck.
type Board struct {
	NextMax   int
	StuckDays int
	Next      []forge.Issue
	Stuck     []Wait
	titles    map[int]string
	now       time.Time
}

// MakeBoard reads the board from the open issues, the report left out;
// the waits pre found on the forge; the proposals the report holds. An
// issue appears once, in its first list; a wait not past stuck-days, an
// announcement aside, is not stuck.
func MakeBoard(open []forge.Issue, report int, waits []Wait, proposed []Pending, cfg Config, now time.Time) Board {
	b := Board{NextMax: cfg.NextMax, StuckDays: cfg.StuckDays, titles: map[int]string{}, now: now}
	isOpen := map[int]bool{}
	for _, is := range open {
		b.titles[is.ID], isOpen[is.ID] = is.Title, is.ID != report
	}
	b.Next = Next(open, report, cfg.NextMax)
	seen := map[int]bool{}
	for _, is := range b.Next {
		seen[is.ID] = true
	}
	all := slices.Clone(waits)
	for _, q := range proposed {
		if q.Since != "" && (q.Issue == 0 || isOpen[q.Issue]) {
			w := Wait{Issue: q.Issue, Waits: WaitsProposed, Since: q.Since, Act: q.Act}
			if q.Issue == 0 {
				w.Line = q.Line
			}
			all = append(all, w)
		}
	}
	slices.SortStableFunc(all, func(x, y Wait) int {
		return cmp.Or(cmp.Compare(slices.Index(waitsOrder, x.Waits), slices.Index(waitsOrder, y.Waits)),
			cmp.Compare(x.Since, y.Since), cmp.Compare(x.Issue, y.Issue))
	})
	for _, w := range all {
		switch {
		case w.Issue != 0 && (!isOpen[w.Issue] || seen[w.Issue]):
		case w.Waits != WaitsObsolete && Days(w.Since, now) <= cfg.StuckDays:
		default:
			if w.Issue != 0 {
				seen[w.Issue] = true
			}
			b.Stuck = append(b.Stuck, w)
		}
	}
	return b
}

// Empty says whether the board lists no issue.
func (b Board) Empty() bool { return len(b.Next) == 0 && len(b.Stuck) == 0 }

// Text is the board as the report opens with it.
func (b Board) Text() string {
	var s strings.Builder
	if b.NextMax > 0 {
		fmt.Fprintf(&s, "\n## Next\n\nWhere to start: the first ready issues in the backlog's order, waiting on no open issue (at most %d, the setting `next-max`).\n\n", b.NextMax)
		if len(b.Next) == 0 {
			fmt.Fprintf(&s, "None: no issue bearing `%s` waits on nothing.\n", LabelReady)
		}
		for i, is := range b.Next {
			fmt.Fprintf(&s, "%d. #%d %s — %s\n", i+1, is.ID, is.Title, Place(is))
		}
	}
	s.WriteString("\n## Stuck\n\n")
	if len(b.Stuck) == 0 {
		fmt.Fprintf(&s, "Nothing has waited on a person for more than %s (the setting `stuck-days`).\n", plural(b.StuckDays, "day"))
		return s.String()
	}
	fmt.Fprintf(&s, "Waiting on a person for more than %s (the setting `stuck-days`), since the day given. An issue announced obsolete waits on a second judge from the day its delay ended.\n\n", plural(b.StuckDays, "day"))
	for _, w := range b.Stuck {
		since := fmt.Sprintf("%s (%s)", w.Since, plural(Days(w.Since, b.now), "day"))
		who := fmt.Sprintf("#%d %s", w.Issue, b.titles[w.Issue])
		switch w.Waits {
		case WaitsReady:
			fmt.Fprintf(&s, "- %s — ready since %s; no pull request nor commit names it since: take it, or take `%s` off.\n", who, since, LabelReady)
		case WaitsAsked:
			fmt.Fprintf(&s, "- %s — its reporter written to on %s; no answer since.\n", who, since)
		case WaitsProposed:
			if w.Issue == 0 {
				fmt.Fprintf(&s, "- %s — proposed here since %s; not ticked, not settled.\n", strings.TrimSuffix(w.Line, "."), since)
				continue
			}
			fmt.Fprintf(&s, "- %s — `%s` proposed here since %s; not ticked, not settled.\n", who, w.Act, since)
		case WaitsObsolete:
			fmt.Fprintf(&s, "- %s — announced obsolete, its delay past since %s; no second judge closed or kept it: close it, or take `%s` off.\n", who, since, LabelObsolete)
		}
	}
	return s.String()
}

// plural says a count of a thing: "1 day", "14 days".
func plural(n int, thing string) string {
	if n == 1 {
		return "1 " + thing
	}
	return fmt.Sprintf("%d %ss", n, thing)
}

// Place says an issue's milestone and priority: "milestone v1.0,
// priority 2", "no milestone, no priority".
func Place(is forge.Issue) string {
	m, p := "no milestone", "no priority"
	if is.Milestone != "" {
		m = "milestone " + is.Milestone
	}
	if n := Priority(is); n > 0 {
		p = fmt.Sprintf("priority %d", n)
	}
	return m + ", " + p
}
