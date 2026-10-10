package backlog

import (
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

func TestDescriptionKeepsTheEngineLines(t *testing.T) {
	body := "Old words.\n\n```\n## not a heading\n```\n\nOpened by the reviewer role.\n\n<!-- workline:opened-by=reviewer -->\n\n## Need\n\nA need."
	if got := DescriptionOf(body); got != "Old words.\n\n```\n## not a heading\n```" {
		t.Fatalf("description = %q", got)
	}
	got := withDescription(body, "New words.")
	want := "New words.\n\nOpened by the reviewer role.\n\n<!-- workline:opened-by=reviewer -->\n\n## Need\n\nA need."
	if got != want {
		t.Fatalf("rewritten = %q, want %q", got, want)
	}
}

// A heading in fenced code is code: the description goes on to the first
// heading outside one — a fence of backticks holding a shorter one, a
// tilde fence; a line of three backticks holding another backtick is
// inline code, no fence.
func TestDescriptionReadsFences(t *testing.T) {
	for _, c := range []struct{ name, body, top string }{
		{"a longer fence", "Old.\n\n````\n```\n## in the code\n```\n````\n\nAfter.\n\n## Need\n\nA need.", "After.\n"},
		{"a tilde fence", "Old.\n\n~~~yaml\n## in the code\n```\n~~~\n\nAfter.\n\n## Need\n\nA need.", "After.\n"},
		{"a tilde fence closed by a longer one", "Old.\n\n~~~\n## in the code\n~~~~\n\nAfter.\n\n## Need\n\nA need.", "After.\n"},
		{"inline code, no fence", "Old: ```go``` here.\n\n## Need\n\nA need.", "here.\n"},
	} {
		top, rest := splitTop(c.body)
		if rest != "## Need\n\nA need." || !strings.HasSuffix(top, c.top) {
			t.Errorf("%s: top = %q, rest = %q", c.name, top, rest)
		}
	}
}

func TestRewriteTellsARolesTextFromAPersons(t *testing.T) {
	finding := "The loop ends early.\n\nOpened by the reviewer role.\n\n<!-- workline:opened-by=reviewer -->"
	bot := forge.Issue{Body: finding, Author: "workline-app[bot]"}
	read := &State{Body: BodyDigest(finding)}
	for _, c := range []struct {
		name string
		is   forge.Issue
		st   *State
		want bool
	}{
		{"a person's issue", forge.Issue{Body: "Rows lost.", Author: "ann"}, nil, false},
		{"recorded at opening", bot, &State{Wrote: map[string]string{DescriptionName: DescriptionDigest(finding)}}, true},
		{"edited since opening", bot, &State{Wrote: map[string]string{DescriptionName: "000000000000"}}, false},
		{"a bot's finding unchanged since read, before the record", bot, read, true},
		{"a GitLab project bot's finding, before the record", forge.Issue{Body: finding, Author: "project_42_bot_3f9a"}, read, true},
		{"a GitLab group bot's finding, before the record", forge.Issue{Body: finding, Author: "group_7_bot"}, read, true},
		{"a finding opened with a person's token, before the record", forge.Issue{Body: finding, Author: "ann"}, read, false},
		{"a finding changed since read", bot, &State{Body: "000000000000"}, false},
		{"rewritten, no comment", bot, &State{Plain: DescriptionDigest(finding)}, false},
	} {
		if got, why := Rewrite(c.is, c.st, false); got != c.want {
			t.Errorf("%s: %v (%s), want %v", c.name, got, why, c.want)
		}
	}
	if ok, _ := Rewrite(bot, &State{Plain: DescriptionDigest(finding)}, true); !ok {
		t.Error("rewritten, a person's comment asking: not rewritable")
	}
	if ok, _ := Rewrite(bot, &State{Plain: "000000000000"}, true); ok {
		t.Error("rewritten, then edited by a person: rewritable")
	}
}

func TestWordsBeforeTheFold(t *testing.T) {
	text := "One two three.\n\n<details><summary>The cause</summary>\n\nfour five six seven\n\n</details>\n\n```\neight nine\n```"
	if n := words(text); n != 3 {
		t.Fatalf("words = %d, want 3", n)
	}
	if n := words(strings.Repeat("word ", 101)); n <= DescriptionWordsMax {
		t.Fatalf("words = %d, want past %d", n, DescriptionWordsMax)
	}
}
