package backlog

import (
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/work"
)

func TestTopTakenTheEngineLinesKept(t *testing.T) {
	body := "Old words.\n\n```\n## not a heading\n```\n\nOpened by the reviewer role.\n\n<!-- workline:opened-by=reviewer -->\n\n## Verification\n\nA test."
	if got := DescriptionOf(body); got != "Old words.\n\n```\n## not a heading\n```" {
		t.Fatalf("description = %q", got)
	}
	got := withoutTop(body)
	want := "Opened by the reviewer role.\n\n<!-- workline:opened-by=reviewer -->\n\n## Verification\n\nA test."
	if got != want {
		t.Fatalf("taken = %q, want %q", got, want)
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

func TestRolesTopToldFromAPersons(t *testing.T) {
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
		{"rewritten in plain words by an earlier engine", bot, &State{Plain: DescriptionDigest(finding)}, true},
		{"rewritten, then edited by a person", bot, &State{Plain: "000000000000"}, false},
	} {
		if got, why := RolesTop(c.is, c.st); got != c.want {
			t.Errorf("%s: %v (%s), want %v", c.name, got, why, c.want)
		}
	}
}

// A person's text on top, with no Need section holding text, is the Need;
// a role's text, a person's Need section, or a role's issue whose record
// was not read, is not.
func TestNeedOnTop(t *testing.T) {
	need := work.Name(work.Need)
	imported := "Exports lose a row.\n\nOpened from `docs/BACKLOG.md`, lines 3 to 3 by the product-owner role.\n\n<!-- workline:import=docs/BACKLOG.md:000000000000 -->"
	for _, c := range []struct {
		name string
		is   forge.Issue
		st   *State
		want bool
	}{
		{"a person's text on top", forge.Issue{Body: "Exports lose a row: 3 sales, 2 rows."}, &State{}, true},
		{"its form's empty field", forge.Issue{Body: "Exports lose a row.\n\n### " + need + "\n\n_No response_"}, &State{}, true},
		{"a person's own section", forge.Issue{Body: "Exports lose a row.\n\n## " + need + "\n\nAnn needs every row."}, &State{}, false},
		{"no text on top", forge.Issue{Body: "## " + need + "\n\n_No response_"}, &State{}, false},
		{"an import a person edited", forge.Issue{Body: imported}, &State{}, true},
		{"an import as opened", forge.Issue{Body: imported}, &State{Wrote: map[string]string{DescriptionName: DescriptionDigest(imported)}}, false},
		{"a role's issue, its record not read", forge.Issue{Body: imported}, nil, false},
	} {
		if got := NeedOnTop(c.is, c.st); got != c.want {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}
