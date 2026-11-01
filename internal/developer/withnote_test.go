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
