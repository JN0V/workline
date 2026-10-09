package backlog

import (
	"fmt"
	"os/exec"
	"regexp"
	"slices"
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
// nor code span, nor fenced block.
func outsideLinks(text string, fn func(string) string) string {
	var spans [][2]int
	for _, m := range linkSpan.FindAllStringIndex(text, -1) {
		spans = append(spans, [2]int{m[0], m[1]})
	}
	spans = append(spans, codeRanges(text)...)
	slices.SortFunc(spans, func(a, b [2]int) int { return a[0] - b[0] })
	var b strings.Builder
	last := 0
	for _, m := range spans {
		if m[0] < last { // within one already left as it is
			if m[1] > last {
				b.WriteString(text[last:m[1]])
				last = m[1]
			}
			continue
		}
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
	bare := `(?:^|[^\w&/#\[])#(\d+)\b`
	if pages == "" {
		return regexp.MustCompile(bare)
	}
	page := regexp.QuoteMeta(pages) + `(\d+)\b[^)\s]*\)?`
	return regexp.MustCompile(bare + `|\[#\d+\]\(` + page + `|` + page)
}

// citation is one issue a text cites: its number, and where the cite ends.
type citation struct{ n, start, end int }

// citations are the issues a text cites, outside code — a code span, a
// fenced block —, in their order.
func citations(text, pages string) []citation {
	code := codeRanges(text)
	var out []citation
	for _, m := range cites(pages).FindAllStringSubmatchIndex(text, -1) {
		if slices.ContainsFunc(code, func(r [2]int) bool { return m[0] < r[1] && m[1] > r[0] }) {
			continue
		}
		for g := 2; g < len(m); g += 2 {
			if m[g] >= 0 {
				n, _ := strconv.Atoi(text[m[g]:m[g+1]])
				out = append(out, citation{n, m[0], m[1]})
				break
			}
		}
	}
	return out
}

var (
	fenceLine  = regexp.MustCompile("(?m)^[ \t]*(`{3,}|~{3,})")
	inlineCode = regexp.MustCompile("`[^`\n]+`")
)

// codeRanges are where a text holds code: each fenced block, to its
// closing fence or the text's end, and each code span outside them.
func codeRanges(text string) [][2]int {
	var out [][2]int
	fences := fenceLine.FindAllStringSubmatchIndex(text, -1)
	for i := 0; i < len(fences); i++ {
		open, mark := fences[i], text[fences[i][2]:fences[i][3]]
		end := len(text)
		for j := i + 1; j < len(fences); j++ {
			if c := text[fences[j][2]:fences[j][3]]; c[0] == mark[0] && len(c) >= len(mark) {
				end, i = fences[j][1], j
				break
			}
		}
		if end == len(text) {
			i = len(fences)
		}
		out = append(out, [2]int{open[0], end})
	}
	inFence := func(at int) bool {
		return slices.ContainsFunc(out, func(r [2]int) bool { return at >= r[0] && at < r[1] })
	}
	for _, s := range inlineCode.FindAllStringIndex(text, -1) {
		if !inFence(s[0]) {
			out = append(out, [2]int{s[0], s[1]})
		}
	}
	return out
}

// CitedIssues are the issues a text cites, by number, lowest first.
func CitedIssues(text, pages string) []int {
	seen := map[int]bool{}
	var out []int
	for _, c := range citations(text, pages) {
		if c.n > 0 && !seen[c.n] {
			seen[c.n] = true
			out = append(out, c.n)
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
	for _, c := range citations(text, pages) {
		end := c.end
		if !closed[c.n] || strings.HasPrefix(text[end:], " (closed") {
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
