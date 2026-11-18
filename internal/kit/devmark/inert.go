package devmark

import (
	"regexp"
	"strings"
)

// closingWord is a word the forges read, in a pull request's body, as
// closing the issue it names.
var closingWord = regexp.MustCompile(`(?i)\b(close[sd]?|fix(e[sd])?|resolve[sd]?)\b`)

// InertText is the agent's text made safe in a pull request body or an
// issue comment: a line that would open a heading, a fence or a rule is
// escaped, so it cannot forge a section, a < cannot open a tag, and an @ is
// broken so it notifies no one; a closing keyword (Closes, Fixes,
// Resolves…) is broken too, so a merge closes no issue the note names.
// Made inert twice, a text stays the same.
func InertText(s string) string {
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	for i, l := range lines {
		t := strings.TrimLeft(l, " \t")
		if strings.HasPrefix(t, "#") || strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~") || strings.HasPrefix(t, "---") || strings.HasPrefix(t, "===") {
			lines[i] = "\\" + t
		}
	}
	joined := closingWord.ReplaceAllStringFunc(strings.Join(lines, "\n"), func(w string) string { return w[:1] + "\u200b" + w[1:] })
	rs := []rune(joined)
	var out strings.Builder
	for i, r := range rs {
		switch {
		case r == '<' && !escapedAt(rs, i):
			out.WriteString("\\<")
		case r == '@' && (i+1 == len(rs) || rs[i+1] != '\u200b'):
			out.WriteString("@\u200b")
		default:
			out.WriteRune(r)
		}
	}
	return out.String()
}

// escapedAt tells whether the rune at i is escaped in Markdown: preceded
// by an odd run of backslashes (an even run escapes only itself).
func escapedAt(rs []rune, i int) bool {
	n := 0
	for j := i - 1; j >= 0 && rs[j] == '\\'; j-- {
		n++
	}
	return n%2 == 1
}
