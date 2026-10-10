package backlog

import (
	"regexp"
	"strings"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/work"
)

// An issue a role opened — an import's item, a finding — starts with a
// text no person wrote: its description, above its sections. The role
// rewrites it in plain words while no person has edited it, the engine
// telling so from what it recorded, never the agent; a person's text is
// never rewritten (ADR-0038, amended).

// DescriptionName names an issue's description among the sections a
// refine writes (Proposal.Added) and those the role wrote (State.Wrote).
const DescriptionName = "Description"

// DescriptionWordsMax bounds a description rewritten in plain words, its
// words outside a fold and fenced code: past it, it is not written.
const DescriptionWordsMax = 100

// engineLine is a line of the engine's in a description: a hidden marker,
// or the line saying where the issue comes from ("Opened from … by the
// … role.", "Found by the … role …").
var engineLine = regexp.MustCompile(`^(<!-- workline:[^\n]*-->|Opened( from .+)? by the [\w -]+ role\.|Found by the [\w -]+ role\b.*\.)$`)

// opener is the line of a body that says a role opened it: an import's
// key, or a finding's role.
var opener = regexp.MustCompile(`(?m)^<!-- workline:(import|opened-by)=[^\n]*-->[ \t\r]*$`)

// importDigest is the digest an import's key holds of the text imported.
var importDigest = regexp.MustCompile(`(?m)^<!-- workline:import=[^\n]*:([0-9a-f]{12}) -->[ \t\r]*$`)

// splitTop splits a body at its first section heading, outside fenced
// code: the text above, the rest from the heading.
func splitTop(body string) (top, rest string) {
	lines := strings.Split(body, "\n")
	fence := ""
	for i, l := range lines {
		t := strings.TrimSpace(l)
		switch {
		case fence != "":
			if strings.HasPrefix(t, fence) {
				fence = ""
			}
			continue
		case strings.HasPrefix(t, "```"):
			fence = "```"
			continue
		case strings.HasPrefix(t, "~~~"):
			fence = "~~~"
			continue
		}
		if _, ok := work.Heading(l); ok {
			return strings.Join(lines[:i], "\n"), strings.Join(lines[i:], "\n")
		}
	}
	return body, ""
}

// DescriptionOf is an issue's description as a person reads it: the text
// above its sections, the engine's lines left out.
func DescriptionOf(body string) string {
	top, _ := splitTop(body)
	var kept []string
	for _, l := range strings.Split(top, "\n") {
		if !engineLine.MatchString(strings.TrimSpace(l)) {
			kept = append(kept, l)
		}
	}
	return strings.TrimSpace(strings.Join(kept, "\n"))
}

// DescriptionDigest is what the state keeps of a description, spaces
// aside: the digest an import's key holds of the text it opened.
func DescriptionDigest(body string) string {
	return BodyDigest(squeeze(DescriptionOf(body)))
}

// withDescription is a body with its description replaced by text, the
// engine's lines there kept after it, in their order.
func withDescription(body, text string) string {
	top, rest := splitTop(body)
	out := strings.TrimSpace(text)
	for _, l := range strings.Split(top, "\n") {
		if t := strings.TrimSpace(l); engineLine.MatchString(t) {
			out += "\n\n" + t
		}
	}
	if rest = strings.TrimSpace(rest); rest != "" {
		out += "\n\n" + rest
	}
	return out
}

// Rewrite says whether the role may rewrite an issue's description in
// plain words, and why not: opened by a role, and as the role wrote it —
// the digest it recorded when opening it (State.Wrote), else an import's
// key; a finding opened before by a bot and its body unchanged since the
// role last read it. Rewritten once (State.Plain), it is again only to
// answer a person's comment (revising). Edited by a person, it is theirs.
func Rewrite(is forge.Issue, st *State, revising bool) (bool, string) {
	if !opener.MatchString(is.Body) {
		return false, "a person opened the issue: its text is theirs, never rewritten"
	}
	if DescriptionOf(is.Body) == "" {
		return false, "it has no text above its sections"
	}
	digest := DescriptionDigest(is.Body)
	switch {
	case st != nil && st.Plain != "" && st.Plain != digest:
		return false, "a person edited its description since the role rewrote it: theirs, never rewritten"
	case st != nil && st.Plain != "" && !revising:
		return false, "its description is rewritten already: again only when a person's comment asks"
	case st != nil && st.Plain != "":
		return true, ""
	case st != nil && st.Wrote[DescriptionName] != "":
		if st.Wrote[DescriptionName] == digest {
			return true, ""
		}
	case importDigest.MatchString(is.Body):
		if importDigest.FindStringSubmatch(is.Body)[1] == digest {
			return true, ""
		}
	case OpenedBy(is.Body) != "" && bot(is.Author) && st != nil && st.Body != "" && st.Body == BodyDigest(is.Body):
		// Opened before the role recorded its text: the bot's, and no
		// change since the role last read it.
		return true, ""
	}
	return false, "a person edited its description, or the role cannot tell no one did: theirs, never rewritten"
}

// bot says whether an author is a bot: a GitHub app's `[bot]` login, a
// GitLab project or group bot.
func bot(author string) bool {
	return strings.HasSuffix(author, "[bot]") || gitlabBot.MatchString(author)
}

// words counts a description's words a reader reads first: those outside
// a fold and fenced code.
func words(text string) int {
	n, fence, folded := 0, false, 0
	for _, l := range strings.Split(text, "\n") {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~") {
			fence = !fence
			continue
		}
		if fence {
			continue
		}
		folded += strings.Count(t, "<details")
		if folded == 0 {
			n += len(strings.Fields(t))
		}
		folded = max(0, folded-strings.Count(t, "</details>"))
	}
	return n
}
