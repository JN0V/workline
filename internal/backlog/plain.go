package backlog

import (
	"regexp"
	"strings"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/work"
)

// An issue a role opened — an import's item, a finding — starts with a
// text no person wrote: its description, above its sections. While no
// person has edited it, the role takes it into the Need it drafts,
// rewritten in plain words, and it leaves the top: each thing said once
// (ADR-0039, amended). The engine tells it is the role's from what it
// recorded, never the agent. A person's text on top, with no Need
// section, is the Need, a person's: read, never touched.

// DescriptionName names an issue's description among the sections a
// refine writes (Proposal.Added: taken into the Need) and those the role
// wrote (State.Wrote: its text at opening).
const DescriptionName = "Description"

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
		var code bool
		if fence, code = inFence(l, fence); code {
			continue
		}
		if _, ok := work.Heading(l); ok {
			return strings.Join(lines[:i], "\n"), strings.Join(lines[i:], "\n")
		}
	}
	return body, ""
}

// inFence reads a line as CommonMark reads fenced code: whether it is the
// code's — its opening line, its text, its closing line — and the fence
// left open after it: one of three backticks or tildes or more, closed by
// a line of the same character at least as long and nothing else.
func inFence(l, open string) (next string, code bool) {
	t := strings.TrimSpace(l)
	run := func(c byte) int {
		n := 0
		for n < len(t) && t[n] == c {
			n++
		}
		return n
	}
	if open != "" {
		if n := run(open[0]); n >= len(open) && strings.TrimSpace(t[n:]) == "" {
			return "", true
		}
		return open, true
	}
	for _, c := range []byte{'`', '~'} {
		if n := run(c); n >= 3 && !(c == '`' && strings.Contains(t[n:], "`")) {
			return t[:n], true
		}
	}
	return "", false
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

// withoutTop is a body with its description taken out, the engine's lines
// there kept, in their order, above its sections.
func withoutTop(body string) string {
	top, rest := splitTop(body)
	var kept []string
	for _, l := range strings.Split(top, "\n") {
		if t := strings.TrimSpace(l); engineLine.MatchString(t) {
			kept = append(kept, t)
		}
	}
	if rest = strings.TrimSpace(rest); rest != "" {
		kept = append(kept, rest)
	}
	return strings.Join(kept, "\n\n")
}

// RolesTop says whether an issue's description is a role's, and why not:
// opened by a role, and as the role wrote it — the digest it recorded when
// opening it (State.Wrote), an import's key, its rewrite in plain words by
// an earlier engine (State.Plain); a finding opened before by a bot, its
// body unchanged since the role last read it. Edited by a person, it is
// theirs.
func RolesTop(is forge.Issue, st *State) (bool, string) {
	if !opener.MatchString(is.Body) {
		return false, "a person opened the issue: its text is theirs"
	}
	if DescriptionOf(is.Body) == "" {
		return false, "it has no text above its sections"
	}
	digest := DescriptionDigest(is.Body)
	switch {
	case st != nil && st.Plain != "":
		if st.Plain == digest {
			return true, ""
		}
	case st != nil && st.Wrote[DescriptionName] != "":
		if st.Wrote[DescriptionName] == digest {
			return true, ""
		}
	case importDigest.MatchString(is.Body):
		if importDigest.FindStringSubmatch(is.Body)[1] == digest {
			return true, ""
		}
	case OpenedBy(is.Body) != "" && Bot(is.Author) && st != nil && st.Body != "" && st.Body == BodyDigest(is.Body):
		// Opened before the role recorded its text: the bot's, and no
		// change since the role last read it.
		return true, ""
	}
	return false, "a person edited its description, or the role cannot tell no one did: theirs"
}

// NeedOnTop says whether an issue's Need is the text above its sections: a
// person's, with no Need section holding text (ADR-0039, amended). The
// engine reads it so, with no agent and nothing recorded; a person's own
// Need section, or a draft of one, is read instead.
func NeedOnTop(is forge.Issue, st *State) bool {
	if strings.TrimSpace(work.ByMeaning(is.Body)[work.Need].Text) != "" || DescriptionOf(is.Body) == "" {
		return false
	}
	if st == nil && opener.MatchString(is.Body) {
		return false // a role's issue, its record not read: never taken for a person's
	}
	roles, _ := RolesTop(is, st)
	return !roles
}
