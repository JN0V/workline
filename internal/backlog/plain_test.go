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
	got := withoutTop(body, false)
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

// What a role folded on top — a finding's cause and its fix — is the
// builder's: as the top is taken into the Need, it goes to the end of the
// last builder's section written, whole, a fence holding `</details>` and
// a block in the block read as theirs; with no builder's section written,
// it stays on top. A plain text on top is taken whole, as before (#841).
func TestRefineKeepsWhatTheTopFolded(t *testing.T) {
	cause := "<details><summary>The cause in the code</summary>\n\nAt `backlog.go:1459`:\n\n```\nreturn \"</details>\"\n```\n\n<details><summary>More</summary>\n\ninner\n\n</details>\n\nWhat would fix it: say the drafts were left out whenever `c.Repeated` is not empty\n\n</details>"
	top := "A refine says nothing was to refine.\n\n" + cause + "\n\nFound by the reviewer role while reviewing abc1234, outside that change.\n\n<!-- workline:opened-by=reviewer -->"
	need := "A person reads a false message."
	for _, c := range []struct {
		name, body      string
		c               Proposal
		contains, lacks []string
	}{
		{"scope written", top, Proposal{Need: need, Scope: "`Refine` in internal/backlog.", Lift: true},
			[]string{"<!-- workline:opened-by=reviewer -->\n\n## Need\n\n<!-- workline:draft -->\n\n" + need + "\n\n---\n\n## Scope\n\n`Refine` in internal/backlog.\n\n" + cause},
			[]string{"A refine says nothing"}},
		{"verification and scope written", top, Proposal{Need: need, Verification: "A test.", Scope: "`Refine`.", Lift: true},
			[]string{"## Verification\n\nA test.\n\n## Scope\n\n`Refine`.\n\n" + cause},
			nil},
		{"verification written alone", top, Proposal{Need: need, Verification: "A test.", Lift: true},
			[]string{"## Verification\n\nA test.\n\n" + cause},
			nil},
		{"no builder's section written", top, Proposal{Need: need, Lift: true},
			[]string{cause + "\n\nFound by the reviewer role while reviewing abc1234, outside that change.\n\n<!-- workline:opened-by=reviewer -->\n\n## Need"},
			[]string{"A refine says nothing"}},
		{"plain text on top", "Exports lose their last row.\n\nOpened by the reviewer role.\n\n<!-- workline:opened-by=reviewer -->", Proposal{Need: need, Scope: "`WriteRows`.", Lift: true},
			[]string{"<!-- workline:opened-by=reviewer -->\n\n## Need\n\n<!-- workline:draft -->\n\n" + need + "\n\n---\n\n## Scope\n\n`WriteRows`."},
			[]string{"Exports lose", "<details"}},
	} {
		got, added, _ := Refine(c.body, c.c, "product-owner")
		for _, s := range c.contains {
			if !strings.Contains(got, s) {
				t.Errorf("%s: body lacks %q:\n%s", c.name, s, got)
			}
		}
		for _, s := range c.lacks {
			if strings.Contains(got, s) {
				t.Errorf("%s: body holds %q:\n%s", c.name, s, got)
			}
		}
		if strings.Count(got, "The cause in the code") > 1 {
			t.Errorf("%s: the folded block written twice:\n%s", c.name, got)
		}
		if len(added) == 0 || added[0] != DescriptionName {
			t.Errorf("%s: added = %v", c.name, added)
		}
	}
}

// A block never closed is not folded; one in fenced code is code.
func TestFoldedNeedsItsClose(t *testing.T) {
	if got := Folded("Text.\n\n<details><summary>Open</summary>\n\nnever closed\n\n## Scope\n\nx"); got != "" {
		t.Errorf("folded = %q", got)
	}
	if got := Folded("```\n<details>\n```\n\n<details>one line</details>"); got != "<details>one line</details>" {
		t.Errorf("folded = %q", got)
	}
}
