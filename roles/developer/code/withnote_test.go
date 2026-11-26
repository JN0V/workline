package developer

import (
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/kit/devmark"
)

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

// The note is inert on the issue too, and the pull request's Left for a
// person holds the same text: made inert twice, a text does not change.
func TestNoteInertOnTheIssueAndTheSameInLeftOut(t *testing.T) {
	raw := "## Need\n<b>x</b> ping @someone"
	kept, _, _ := bounded(t.TempDir(), Bounds{Issue: 7}, nil, []intent.Intention{{Kind: "note", Value: map[string]any{"issue": 7, "body": raw}}})
	note := noteOf(kept, 7)
	if note != devmark.InertText(raw) {
		t.Fatalf("issue comment = %q, want it inert: %q", note, devmark.InertText(raw))
	}
	if devmark.InertText(note) != note {
		t.Errorf("inert twice changes the text: %q -> %q", note, devmark.InertText(note))
	}
	if got := body(devmark.Act{Issue: 7}, true, note); !strings.Contains(got, note+"\n") || !strings.HasPrefix(got, "Part of #7\n") {
		t.Errorf("Left for a person does not hold the issue's text: %q", got)
	}
}

// A note naming an issue with a closing keyword closes nothing at the merge.
func TestInertTextBreaksClosingKeywords(t *testing.T) {
	got := devmark.InertText("Closes #12, fixes #13 and resolved #14; a fixture stays.")
	for _, w := range []string{"Closes #", "fixes #", "resolved #"} {
		if strings.Contains(got, w) {
			t.Errorf("%q left live: %q", w, got)
		}
	}
	if !strings.Contains(got, "a fixture stays") || devmark.InertText(got) != got {
		t.Errorf("InertText = %q", got)
	}
}

// An escaped backslash before < leaves the tag live: it is escaped again.
func TestInertTextEscapesATagAfterAnEscapedBackslash(t *testing.T) {
	for in, want := range map[string]string{`\<b>`: `\<b>`, `\\<b>`: `\\\<b>`, `<b>`: `\<b>`} {
		if got := devmark.InertText(in); got != want || devmark.InertText(got) != got {
			t.Errorf("InertText(%q) = %q, want %q, the same made inert twice", in, got, want)
		}
	}
}
