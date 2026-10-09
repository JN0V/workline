package backlog

import (
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// What the role writes in an issue is read by a person (AGENTS.md,
// "Writing user docs"): a decision it cites is a link they can follow, an
// issue it cites says when it is closed. Both are mechanical, and done by
// the engine on every section it writes, whatever the agent wrote.

var (
	// A link, a bare address or a code span: text left as it is.
	linkSpan = regexp.MustCompile("!?\\[[^\\]]*\\]\\([^)]*\\)|<https?://[^>]*>|https?://\\S+|`[^`\n]*`")
	adrRef   = regexp.MustCompile(`\bADR[- ]?(\d{1,5})\b`)
	// A decision's file: a numbered page in a folder of decisions.
	decisionFile = regexp.MustCompile(`(?i)(?:^|/)(?:adr|adrs|decisions?)/0*(\d+)-[^/]*\.md$`)
)

// outsideLinks applies fn to the parts of text that are no link, address
// nor code span.
func outsideLinks(text string, fn func(string) string) string {
	var b strings.Builder
	last := 0
	for _, m := range linkSpan.FindAllStringIndex(text, -1) {
		b.WriteString(fn(text[last:m[0]]))
		b.WriteString(text[m[0]:m[1]])
		last = m[1]
	}
	b.WriteString(fn(text[last:]))
	return b.String()
}

// Decisions are the decision records the commit holds, by number: the
// path of each.
func Decisions(repo string) map[int]string {
	out, _ := exec.Command("git", "-C", repo, "ls-files").Output()
	found := map[int]string{}
	for _, f := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if m := decisionFile.FindStringSubmatch(f); m != nil {
			if n, err := strconv.Atoi(m[1]); err == nil {
				if _, twice := found[n]; !twice {
					found[n] = f
				}
			}
		}
	}
	return found
}

// LinkDecisions makes each decision a text names bare — "ADR-0038" — a
// link to its page on the forge (pages, a file's path after it), when the
// commit holds it; one in a link or a code span is left.
func LinkDecisions(text, pages string, decisions map[int]string) string {
	if pages == "" || len(decisions) == 0 {
		return text
	}
	return outsideLinks(text, func(part string) string {
		return adrRef.ReplaceAllStringFunc(part, func(ref string) string {
			n, _ := strconv.Atoi(adrRef.FindStringSubmatch(ref)[1])
			if path, ok := decisions[n]; ok {
				return "[" + ref + "](" + pages + path + ")"
			}
			return ref
		})
	})
}

// cites finds the issues a text cites: `#12`, as a forge links it, or a
// link to its page on this forge (pages, the issue's number after it); a
// link's end is where a mark goes after it.
func cites(pages string) *regexp.Regexp {
	page := `(\b\B\d+)` // matches nothing: no page known
	if pages != "" {
		page = regexp.QuoteMeta(pages) + `(\d+)\b[^)\s]*\)?`
	}
	return regexp.MustCompile(`(?:^|[^\w&/#\[])#(\d+)\b|\[#\d+\]\(` + page + `|` + page)
}

// CitedIssues are the issues a text cites, by number, lowest first.
func CitedIssues(text, pages string) []int {
	seen := map[int]bool{}
	var out []int
	for _, m := range cites(pages).FindAllStringSubmatch(text, -1) {
		n, _ := strconv.Atoi(m[1] + m[2] + m[3])
		if n > 0 && !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	sort.Ints(out)
	return out
}

// MarkClosed says "(closed)" after each issue a text cites that is closed
// (closed, by number), once: a reader never takes it for work to come.
func MarkClosed(text, pages string, closed map[int]bool) string {
	if len(closed) == 0 {
		return text
	}
	var b strings.Builder
	last := 0
	for _, m := range cites(pages).FindAllStringSubmatchIndex(text, -1) {
		n := 0
		for g := 2; g < len(m); g += 2 {
			if m[g] >= 0 {
				n, _ = strconv.Atoi(text[m[g]:m[g+1]])
			}
		}
		end := m[1]
		if !closed[n] || strings.HasPrefix(text[end:], " (closed") || strings.Count(text[:m[1]], "`")%2 == 1 {
			continue
		}
		b.WriteString(text[last:end])
		b.WriteString(" (closed)")
		last = end
	}
	b.WriteString(text[last:])
	return b.String()
}

// IssueStates says, for the issues a text cites, which are open and which
// closed and why: "#81 (open), #206 (closed as not planned)"; an issue the
// forge does not list (a merge request, another project's) is left out.
func IssueStates(ids []int, open map[int]bool, closed map[int]string) string {
	var out []string
	for _, n := range ids {
		switch reason, ok := closed[n]; {
		case open[n]:
			out = append(out, fmt.Sprintf("#%d (open)", n))
		case ok && reason == "not_planned":
			out = append(out, fmt.Sprintf("#%d (closed as not planned)", n))
		case ok && reason == "duplicate":
			out = append(out, fmt.Sprintf("#%d (closed as a duplicate)", n))
		case ok:
			out = append(out, fmt.Sprintf("#%d (closed)", n))
		}
	}
	return strings.Join(out, ", ")
}
