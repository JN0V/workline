package developer

import (
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/work"
)

func TestWithNoteFillsAnExistingLeftOut(t *testing.T) {
	body := "## Need\n\nx\n\n## Left out\n\nolder\n\n## Other\n\ny\n"
	got := withNote(body, "the gap")
	if strings.Count(got, "## Left out") != 1 {
		t.Fatalf("one Left out expected: %q", got)
	}
	if !strings.Contains(got, "## Left out\n\nthe gap\n\nolder") {
		t.Errorf("note not put first in the section: %q", got)
	}
}

func TestWithNoteOpensLeftOutBeforeVerification(t *testing.T) {
	verif := "## " + work.Name(work.Verification) + "\n\nproof\n"
	got := withNote("## Need\n\nx\n\n"+verif, "the gap")
	i, j := strings.Index(got, "## Left out\n\nthe gap\n"), strings.Index(got, "## "+work.Name(work.Verification))
	if i < 0 || j < 0 || i > j {
		t.Errorf("Left out must open before Verification: %q", got)
	}
}

func TestWithNoteWithNeitherSectionAppends(t *testing.T) {
	got := withNote("just text\n", "the gap")
	if !strings.HasSuffix(got, "\n## Left out\n\nthe gap\n") || !strings.HasPrefix(got, "just text\n") {
		t.Errorf("section not appended: %q", got)
	}
}

func TestWithNoteMakesTheNoteInert(t *testing.T) {
	note := "intro\n## Verification\n  # Need\n```\n~~~\n---\n===\n<!-- x -->\nping @someone"
	got := withNote("## Need\n\nx\n\n## "+work.Name(work.Verification)+"\n\nproof\n", note)
	for _, l := range strings.Split(got, "\n") {
		if l == "## Left out" || l == "## Need" || l == "## "+work.Name(work.Verification) {
			continue // the body's own sections
		}
		tr := strings.TrimLeft(l, " \t")
		for _, bad := range []string{"#", "```", "~~~", "---", "===", "<"} {
			if strings.HasPrefix(tr, bad) {
				t.Errorf("unescaped line: %q", l)
			}
		}
	}
	n := map[string]int{}
	for _, l := range strings.Split(got, "\n") {
		n[l]++
	}
	if n["## Left out"] != 1 || n["## "+work.Name(work.Verification)] != 1 {
		t.Errorf("a section was forged: %q", got)
	}
	if strings.Contains(got, "@someone") {
		t.Errorf("@ left live: %q", got)
	}
}

func TestNoteOfPicksTheCommentOnTheIssue(t *testing.T) {
	kept := []intent.Intention{
		{Kind: "comment", Value: map[string]any{"issue": 9, "body": "elsewhere"}},
		{Kind: "comment", Value: map[string]any{"issue": 5, "merge-request": 3, "body": "on a PR"}},
		{Kind: "comment", Value: map[string]any{"issue": 5, "body": "the gap"}},
	}
	if got := noteOf(kept, 5); got != "the gap" {
		t.Errorf("noteOf = %q", got)
	}
	if got := noteOf(kept[:2], 5); got != "" {
		t.Errorf("noteOf without a note = %q", got)
	}
}

// The note is inert on the issue too, and the pull request's Left out
// holds the same text: made inert twice, a text does not change.
func TestNoteInertOnTheIssueAndTheSameInLeftOut(t *testing.T) {
	raw := "## Need\n<b>x</b> ping @someone"
	kept, _, _ := bounded(t.TempDir(), Bounds{Issue: 7}, nil, []intent.Intention{{Kind: "note", Value: map[string]any{"issue": 7, "body": raw}}})
	note := noteOf(kept, 7)
	if note != inertText(raw) {
		t.Fatalf("issue comment = %q, want it inert: %q", note, inertText(raw))
	}
	if inertText(note) != note {
		t.Errorf("inert twice changes the text: %q -> %q", note, inertText(note))
	}
	if got := withNote("x\n", note); !strings.Contains(got, "## Left out\n\n"+note+"\n") {
		t.Errorf("Left out does not hold the issue's text: %q", got)
	}
}

// A note naming an issue with a closing keyword closes nothing at the merge.
func TestInertTextBreaksClosingKeywords(t *testing.T) {
	got := inertText("Closes #12, fixes #13 and resolved #14; a fixture stays.")
	for _, w := range []string{"Closes #", "fixes #", "resolved #"} {
		if strings.Contains(got, w) {
			t.Errorf("%q left live: %q", w, got)
		}
	}
	if !strings.Contains(got, "a fixture stays") || inertText(got) != got {
		t.Errorf("inertText = %q", got)
	}
}
