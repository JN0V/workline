package report

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/JN0V/workline/internal/line"
	"github.com/JN0V/workline/internal/verdict"
)

// MainTitle is the one issue main's run after a merge keeps its findings
// in (ADR-0050): the whole repository's checks, and what the gates found
// main gained, for whoever reads why main changed.
const MainTitle = "workline: findings on the default branch"

// mainMarker opens the line of the issue's body holding what it lists,
// by key: what the next run reads back. backlog.Tracking reads it too.
const mainMarker = "<!-- workline:main-findings "

// mainShown caps the entries each section lists; the rest are counted.
const mainShown = 100

const mainADR = "https://github.com/JN0V/workline/blob/main/docs/adr/0050-each-act-has-its-moment-and-a-wait-budget.md"

// mainEntry is one finding the issue lists.
type mainEntry struct {
	Key     string `json:"key"`
	Step    string `json:"step"`
	Rule    string `json:"rule,omitempty"`
	Where   string `json:"where,omitempty"`
	Message string `json:"message,omitempty"`
	Since   string `json:"since,omitempty"` // the commit of main it was first found on
}

// mainState is what the issue's marker holds: every key it lists, and the
// gates' findings, kept from one run to the next while main holds them.
type mainState struct {
	Keys  []string    `json:"keys"`
	Gates []mainEntry `json:"gates,omitempty"`
}

// MainIssue is the body of the issue main's findings are kept in, from
// main's run and the issue's body now ("" when none is open): the gates
// red, the findings they found main gained while main holds them, then
// the whole repository's. Changed: what it lists differs, by key, from
// what the issue lists; else it is left as it is. Any: something to list.
// What a run wrote is data: cut and defused.
func MainIssue(r *line.Result, now string) (body string, changed, any bool) {
	prev := readMainState(now)
	head, before := "", ""
	var commits []string
	if r.Main != nil {
		head, before, commits = r.Main.Head, r.Main.Before, r.Main.Commits
	}
	since := map[string]string{}
	for _, e := range prev.Gates {
		since[e.Key] = e.Since
	}
	var red []string
	var gates, docs []mainEntry
	var keys []string
	for _, s := range r.Steps {
		gate := strings.HasPrefix(s.Name, "gate:")
		switch {
		case gate && s.Status != verdict.Pass && s.Gate != nil:
			for _, f := range s.Gate.Findings {
				if f.Level == "" || f.Level == verdict.Block {
					said := markedIn.ReplaceAllString(f.Message, ")")
					red = append(red, fmt.Sprintf("%s: %s", span(s.Name), inert(strings.TrimSpace(f.Where+" "+said), 300)))
					keys = append(keys, "red\x00"+s.Name+"\x00"+f.Rule+"\x00"+f.Where)
				}
			}
		}
		if gate {
			kept := map[string]bool{}
			if s.New != nil {
				for _, p := range s.New.Present {
					kept[p] = true
				}
			}
			// Found on main earlier, still there; with nothing read of
			// this run, kept as it was.
			for _, e := range prev.Gates {
				if e.Step == s.Name && (s.New == nil || kept[e.Key]) {
					gates = append(gates, e)
				}
			}
			if s.New != nil {
				for _, f := range s.New.Findings {
					if slices.ContainsFunc(gates, func(e mainEntry) bool { return e.Key == f.Fingerprint }) {
						continue
					}
					where := f.Path
					if f.Line > 0 {
						where = fmt.Sprintf("%s:%d", f.Path, f.Line)
					}
					rule := f.Check
					if f.Rule != "" {
						rule += "/" + f.Rule
					}
					gates = append(gates, mainEntry{Key: f.Fingerprint, Step: s.Name, Rule: rule, Where: where, Message: f.Message, Since: orSince(since[f.Fingerprint], head)})
				}
			}
			continue
		}
		if s.Result == nil {
			continue
		}
		for _, f := range s.Result.Findings {
			said := firstLine(strings.TrimSpace(f.Message))
			docs = append(docs, mainEntry{Key: s.Name + "\x00" + f.Rule + "\x00" + f.Where + "\x00" + digits.ReplaceAllString(said, "#"),
				Step: s.Name, Rule: f.Rule, Where: f.Where, Message: said})
		}
	}
	for _, e := range gates {
		keys = append(keys, "gate\x00"+e.Key)
	}
	for _, e := range docs {
		keys = append(keys, "step\x00"+e.Key)
	}
	for i, k := range keys { // short, and holding nothing a run wrote
		sum := sha256.Sum256([]byte(k))
		keys[i] = hex.EncodeToString(sum[:8])
	}
	sort.Strings(keys)
	keys = slices.Compact(keys)
	any = len(keys) > 0
	changed = !slices.Equal(keys, prev.Keys) || now == "" && any

	var b strings.Builder
	// The count by kind first; the lists folded below: on day one, the old
	// debt is shown once, then only what changes (ADR-0049, section 6).
	if any {
		fmt.Fprintf(&b, "**%d %s on the default branch**: %s.\n\n", len(gates)+len(docs), plural(len(gates)+len(docs), "finding", "findings"), byKind(gates, docs, len(red) > 0))
	} else {
		fmt.Fprintf(&b, "**Nothing found on the default branch** at %s.\n\n", span(short7(head)))
	}
	fmt.Fprintf(&b, "The checks of the whole repository, run once on the default branch after each merge, and what its gates found it gained ([ADR-0050](%s)). Nothing here holds a merge request. workline keeps this issue: it is edited when what it lists changes, never commented on.\n\n", mainADR)
	if len(red) > 0 {
		fmt.Fprintf(&b, "**The default branch's run is red** at %s: the merges since %s add these findings together, though each merge request was checked alone, against its own base.\n\n", span(short7(head)), span(short7(before)))
		for _, l := range red {
			b.WriteString("- " + l + "\n")
		}
		if len(commits) > 0 {
			fmt.Fprintf(&b, "\nWhat %s gained since %s:\n\n", span(short7(head)), span(short7(before)))
			for _, c := range commits {
				b.WriteString("- " + inert(c, 200) + "\n")
			}
		}
		b.WriteString("\n")
	}
	if len(gates) > 0 {
		fmt.Fprintf(&b, "<details><summary>Gates: %d %s the default branch gained</summary>\n\n", len(gates), plural(len(gates), "finding", "findings"))
		b.WriteString("Each was found new after a merge, and stays here while the branch holds it.\n\n")
		list(&b, gates, func(e mainEntry) string {
			s := fmt.Sprintf("%s %s", span(e.Where), span(e.Rule))
			if m := inert(e.Message, 200); m != "" {
				s += " — " + m
			}
			if e.Since != "" {
				s += fmt.Sprintf(" (since %s)", span(short7(e.Since)))
			}
			return s
		})
	}
	if len(docs) > 0 {
		fmt.Fprintf(&b, "<details><summary>The whole repository: %d %s</summary>\n\n", len(docs), plural(len(docs), "finding", "findings"))
		list(&b, docs, func(e mainEntry) string {
			s := span(strings.TrimPrefix(e.Step+" "+e.Rule, " "))
			if e.Where != "" {
				s = span(e.Where) + " " + s
			}
			if m := inert(e.Message, 200); m != "" {
				s += " — " + m
			}
			return s
		})
	}
	if any {
		fmt.Fprintf(&b, "Last changed at %s.\n", span(short7(head)))
	}
	state, _ := json.Marshal(mainState{Keys: keys, Gates: gates})
	fmt.Fprintf(&b, "\n%s%s -->\n", mainMarker, strings.ReplaceAll(string(state), "--", `-\u002d`))
	return b.String(), changed, any
}

// list writes entries, mainShown at most, the rest counted, folded in
// the <details> its caller opened.
func list(b *strings.Builder, es []mainEntry, say func(mainEntry) string) {
	for i, e := range es {
		if i == mainShown {
			fmt.Fprintf(b, "- and %d more, not listed here\n", len(es)-mainShown)
			break
		}
		b.WriteString("- " + say(e) + "\n")
	}
	b.WriteString("\n</details>\n\n")
}

// byKind counts the findings by rule, the most first: `53
// section-too-long, 30 doc-too-long`; main red said first.
func byKind(gates, docs []mainEntry, red bool) string {
	n := map[string]int{}
	var kinds []string
	for _, e := range append(slices.Clone(gates), docs...) {
		if n[e.Rule] == 0 {
			kinds = append(kinds, e.Rule)
		}
		n[e.Rule]++
	}
	sort.SliceStable(kinds, func(i, j int) bool { return n[kinds[i]] > n[kinds[j]] })
	var out []string
	if red {
		out = append(out, "main's run red")
	}
	for _, k := range kinds {
		out = append(out, fmt.Sprintf("%d %s", n[k], span(k)))
	}
	return strings.Join(out, ", ")
}

var digits = regexp.MustCompile(`[0-9]+`)

// markedIn is where a gate's finding says its results are marked: a
// folder of the job's machine, nothing for the issue's reader.
var markedIn = regexp.MustCompile(`; each marked in [^)]*\)`)

// readMainState reads what an issue's body lists, from its marker; none
// when it holds none, or one that does not read.
func readMainState(body string) mainState {
	var s mainState
	_, rest, ok := strings.Cut(body, mainMarker)
	if !ok {
		return s
	}
	data, _, _ := strings.Cut(rest, " -->")
	if json.Unmarshal([]byte(data), &s) != nil {
		return mainState{}
	}
	return s
}

func orSince(s, head string) string {
	if s != "" {
		return s
	}
	return head
}
