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
)

// dryRunStart and dryRunEnd bound the dry run's part of the default
// branch's issue (MainTitle): the part main's run after a merge leaves as
// it is, and the other way round.
// The end's line holds what the part lists, by key.
const (
	dryRunStart = "<!-- workline:release-dry-run -->"
	dryRunEnd   = "<!-- workline:release-dry-run-end "
)

// dryRunShown caps the holds listed; the rest are counted.
const dryRunShown = 50

// dryRunState is what the end's line holds: every key the part lists.
type dryRunState struct {
	Keys []string `json:"keys"`
}

// dryRunKeys are what a dry run lists, by key: short, and holding
// nothing a run wrote; the numbers of a message left out, so that a count
// moving alone is no change.
func dryRunKeys(d *line.DryRun) []string {
	var keys []string
	for _, h := range d.Holds {
		k := h.Step + "\x00" + h.Rule + "\x00" + h.Where + "\x00" + digits.ReplaceAllString(firstLine(strings.TrimSpace(h.Message)), "#")
		sum := sha256.Sum256([]byte(k))
		keys = append(keys, hex.EncodeToString(sum[:8]))
	}
	sort.Strings(keys)
	return slices.Compact(keys)
}

// dryRunText says what a release's dry run found, for a person: what
// would hold the release, each on a line, or that nothing would. What a
// run wrote is data: cut and defused.
func dryRunText(d *line.DryRun) string {
	var b strings.Builder
	from := "from " + span(short7(d.Head))
	if d.Since != "" {
		from += ", since " + span(d.Since)
	}
	if len(d.Holds) == 0 {
		fmt.Fprintf(&b, "**A release's dry run**: nothing would hold the release if it were cut now, %s.\n\n", from)
	} else {
		fmt.Fprintf(&b, "**A release's dry run**: %d %s hold the release if it were cut now, %s:\n\n", len(d.Holds), plural(len(d.Holds), "finding would", "findings would"), from)
		for i, h := range d.Holds {
			if i == dryRunShown {
				fmt.Fprintf(&b, "- and %d more, not listed here\n", len(d.Holds)-dryRunShown)
				break
			}
			s := span(strings.TrimSpace(h.Step + " " + h.Rule))
			if h.Where != "" {
				s = span(h.Where) + " " + s
			}
			if m := inert(heldFor(h.Message), 200); m != "" {
				s += " — " + m
			}
			b.WriteString("- " + s + "\n")
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "Run each night with no agent, nothing applied: a warning, never a block ([ADR-0050](%s)). workline keeps this text, edited when what it lists changes.\n", mainADR)
	return b.String()
}

// WithDryRun is the default branch's issue body with the dry run's part:
// put in, replaced, or taken out when nothing holds (d nil: taken out).
// Changed: the body differs, by what the part lists.
func WithDryRun(now string, d *line.DryRun) (body string, changed bool) {
	rest, part := cutDryRun(now)
	prev, had := readDryRunState(part)
	var keys []string
	if d != nil {
		keys = dryRunKeys(d)
	}
	switch {
	case len(keys) == 0 && !had:
		return now, false
	case len(keys) == 0 && strings.TrimSpace(rest) == "" && len(prev.Keys) == 0:
		return now, false // said already: nothing holds
	case len(keys) == 0 && strings.TrimSpace(rest) == "":
		// The issue held this part alone: it says nothing holds now,
		// kept as the issue workline keeps.
		return dryRunStart + "\n\n### The next release\n\nNothing would hold the release now, by the night's dry run ([ADR-0050](" + mainADR + ")).\n\n" + dryRunEndLine(nil), true
	case len(keys) == 0:
		return strings.TrimRight(rest, "\n") + "\n", true
	}
	if had && slices.Equal(keys, prev.Keys) {
		return now, false
	}
	part = dryRunStart + "\n\n### The next release\n\n" + dryRunText(d) + "\n" + dryRunEndLine(keys)
	if strings.TrimSpace(rest) == "" {
		return part, true
	}
	return strings.TrimRight(rest, "\n") + "\n\n" + part, true
}

// cutDryRun splits an issue's body into the rest and the dry run's part,
// "" when it holds none.
func cutDryRun(body string) (rest, part string) {
	i := strings.Index(body, dryRunStart)
	if i < 0 {
		return body, ""
	}
	j := strings.Index(body[i:], dryRunEnd)
	if j < 0 {
		return body[:i], body[i:]
	}
	end := i + j
	if k := strings.Index(body[end:], "-->"); k >= 0 {
		end += k + len("-->")
	} else {
		end = len(body)
	}
	return body[:i] + strings.TrimLeft(body[end:], "\n"), body[i:end]
}

// keptDryRun is the dry run's part of an issue's body, "" for none: what
// main's run keeps as it is when it rewrites the issue.
func keptDryRun(body string) string {
	_, part := cutDryRun(body)
	return part
}

// heldFor is what a hold says, for the issue's or comment's reader: its
// first line, a log's path on the job's machine left out.
func heldFor(message string) string {
	l := strings.TrimSpace(firstLine(strings.TrimSpace(message)))
	l = seeLog.ReplaceAllString(l, "")
	return strings.TrimSuffix(l, ":")
}

// seeLog is where a gate's check says its log is: a folder of the job's
// machine, nothing for the reader.
var seeLog = regexp.MustCompile(`; see /\S+`)

func dryRunEndLine(keys []string) string {
	if keys == nil {
		keys = []string{}
	}
	state, _ := json.Marshal(dryRunState{Keys: keys})
	return dryRunEnd + strings.ReplaceAll(string(state), "--", `-\u002d`) + " -->\n"
}

// readDryRunState reads what a text lists, from the end's line; false
// when it holds none.
func readDryRunState(text string) (dryRunState, bool) {
	var s dryRunState
	_, rest, ok := strings.Cut(text, dryRunEnd)
	if !ok {
		return s, false
	}
	data, _, _ := strings.Cut(rest, " -->")
	if json.Unmarshal([]byte(data), &s) != nil {
		return dryRunState{}, false
	}
	return s, true
}
