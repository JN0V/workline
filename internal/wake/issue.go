package wake

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"
)

// Title is the one issue the wake keeps on the forge.
const Title = "workline: where the line may be drifting"

// Kept is how long a trigger stays listed: a person reading the issue
// once a week reads each one.
const Kept = 7 * 24 * time.Hour

// shown caps the entries a section lists; the rest are counted.
const shown = 50

const adrLink = "https://github.com/JN0V/workline/blob/main/docs/adr/0068-the-line-steps-back-on-its-own-drift.md#8-the-wake-script-print-only"

// Marker opens the line of the issue's body holding what it lists, read
// back by the next run.
const Marker = "<!-- workline:wake "

// sections, in order, with their headings.
var sections = []struct{ kind, title string }{
	{Contract, "Growth of the contract"},
	{Prompt, "Prompts that steer the line"},
	{Symptom, "Symptoms without cause"},
	{Need, "Laundered needs"},
	{Bypass, "Nobody bypasses"},
	{Cord, "The cord"},
}

// state is what the issue's marker holds.
type state struct {
	Entries []Entry `json:"entries"`
	Head    string  `json:"head,omitempty"` // the commit the last read ended at
}

// KeptHead is the commit the issue's last read ended at, "" when none.
func KeptHead(body string) string { return readState(body).Head }

// Issue is the body of the wake's issue, from this read and the issue's
// body now ("" when none is open): the triggers met in the last 7 days,
// the newest read of each. Changed: what it lists differs from what the
// issue lists. Any: something to list.
func Issue(r *Result, now string) (body string, changed, any bool) {
	prev := readState(now)
	day, _ := time.Parse(time.DateOnly, r.Day)
	var entries []Entry
	for _, e := range prev.Entries {
		at, err := time.Parse(time.DateOnly, e.Day)
		if err != nil || day.Sub(at) >= Kept || slices.ContainsFunc(r.Entries, func(n Entry) bool { return n.Key == e.Key }) {
			continue
		}
		entries = append(entries, e)
	}
	for _, e := range r.Entries {
		if i := slices.IndexFunc(entries, func(n Entry) bool { return n.Key == e.Key }); i >= 0 {
			entries[i] = e
			continue
		}
		entries = append(entries, e)
	}
	any = len(entries) > 0
	// The head read moved, the issue is edited: the next read starts there.
	changed = !sameEntries(entries, prev.Entries) || prev.Head != "" && prev.Head != r.Head

	body = render(r, entries, 0)
	if len(body) <= limit {
		return body, changed, any
	}
	// Past the forge's limit, the oldest triggers are left out, of the
	// list and of the record, as few as keep the body under it: said in
	// the issue, never a body the forge refuses.
	order := make([]int, len(entries))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int { return strings.Compare(entries[b].Day, entries[a].Day) })
	keep := func(k int) []Entry {
		in := map[int]bool{}
		for _, i := range order[:k] {
			in[i] = true
		}
		var kept []Entry
		for i, e := range entries {
			if in[i] {
				kept = append(kept, e)
			}
		}
		return kept
	}
	k := sort.Search(len(entries)+1, func(k int) bool {
		return len(render(r, keep(k), len(entries)-k)) > limit
	}) - 1
	return render(r, keep(max(k, 0)), len(entries)-max(k, 0)), changed, any
}

// limit is the size the issue's body stays under: GitHub refuses one
// past 65,536 characters; the margin is for what is escaped on the way.
const limit = 60000

// render is the issue's body listing entries, left saying how many
// triggers it left out to stay under the forge's limit.
func render(r *Result, entries []Entry, left int) string {
	var b strings.Builder
	if total := len(entries) + left; total > 0 {
		var counts []string
		for _, s := range sections {
			if n := count(entries, s.kind); n > 0 {
				counts = append(counts, fmt.Sprintf("%s %d", strings.ToLower(s.title), n))
			}
		}
		fmt.Fprintf(&b, "**%d %s met** in the 7 days to %s: %s.\n\n", total, plural(total, "trigger", "triggers"), r.Day, strings.Join(counts, ", "))
	} else {
		fmt.Fprintf(&b, "**No trigger met** in the 7 days to %s.\n\n", r.Day)
	}
	if left > 0 {
		fmt.Fprintf(&b, "**%d oldest %s left out** of this issue, listed and kept: past the forge's size limit.\n\n", left, plural(left, "trigger", "triggers"))
	}
	fmt.Fprintf(&b, "Where the line may be drifting, read after each merge to the default branch, with no AI ([ADR-0068 §8](%s)). Nothing here holds a merge. To step back, put the label `%s` on any issue or merge request. workline keeps this issue: edited in place, never commented on; each trigger is listed 7 days.\n\n", adrLink, Label)
	for _, s := range sections {
		var list []Entry
		for _, e := range entries {
			if e.Kind == s.kind {
				list = append(list, e)
			}
		}
		if len(list) == 0 {
			continue
		}
		fmt.Fprintf(&b, "### %s\n\n", s.title)
		for i, e := range list {
			if i == shown {
				fmt.Fprintf(&b, "- and %d more, not listed here\n", len(list)-shown)
				break
			}
			fmt.Fprintf(&b, "- %s (`%s`, %s)\n", e.Text, short(e.At), e.Day)
		}
		b.WriteString("\n")
	}
	if r.Facts != "" {
		b.WriteString(r.Facts + "\n\n")
	}
	if len(r.Unread) > 0 {
		b.WriteString("**Not read** at " + short(r.Head) + ":\n\n")
		for _, u := range r.Unread {
			b.WriteString("- " + u + "\n")
		}
		b.WriteString("\n")
	}
	// json.Marshal escapes `>`: no `-->` ends the comment early.
	kept, _ := json.Marshal(state{Entries: entries, Head: r.Head})
	fmt.Fprintf(&b, "%s%s -->\n", Marker, kept)
	return b.String()
}

func count(es []Entry, kind string) int {
	n := 0
	for _, e := range es {
		if e.Kind == kind {
			n++
		}
	}
	return n
}

// sameEntries: the same triggers, each said the same, in any order.
func sameEntries(a, b []Entry) bool {
	if len(a) != len(b) {
		return false
	}
	for _, e := range a {
		if !slices.ContainsFunc(b, func(o Entry) bool { return o.Key == e.Key && o.Text == e.Text }) {
			return false
		}
	}
	return true
}

func readState(body string) state {
	var s state
	_, rest, ok := strings.Cut(body, Marker)
	if !ok {
		return s
	}
	data, _, _ := strings.Cut(rest, " -->")
	if json.Unmarshal([]byte(data), &s) != nil {
		return state{}
	}
	return s
}
