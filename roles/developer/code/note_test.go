package developer

import (
	"fmt"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
)

// notesKept are the comments bounded kept beyond the claim.
func notesKept(kept []intent.Intention, claim int) []map[string]any {
	var out []map[string]any
	for _, in := range kept[claim:] {
		if m, ok := in.Value.(map[string]any); ok && in.Kind == "comment" {
			out = append(out, m)
		}
	}
	return out
}

func noteOn(issue int, body string) intent.Intention {
	return intent.Intention{Kind: "note", Value: map[string]any{"issue": issue, "body": body}}
}

// One note: Given an agent that leaves part of the issue out, When it
// writes a note, Then one comment from the developer appears on its own
// issue, saying what was left out.
func TestOneNote(t *testing.T) {
	b := Bounds{Issue: 7}
	claim := []intent.Intention{{Kind: "label", Value: map[string]any{"issue": 7}}}
	text := "left out: the retry path, it needs its own issue"
	kept, _, _ := bounded(t.TempDir(), b, claim, []intent.Intention{noteOn(7, text)})
	got := notesKept(kept, len(claim))
	if len(got) != 1 {
		t.Fatalf("want one comment, got %d: %v", len(got), kept)
	}
	if fmt.Sprint(got[0]["issue"]) != "7" {
		t.Errorf("comment on issue %v, want 7", got[0]["issue"])
	}
	if body, _ := got[0]["body"].(string); !strings.Contains(body, text) {
		t.Errorf("body %q does not say %q", body, text)
	}
}

// Bounded: Given an agent that writes three notes, or one far too long,
// When the developer applies them, Then only one comment is posted, cut to
// its limit, and the rest is refused and said.
func TestBounded(t *testing.T) {
	b := Bounds{Issue: 7}
	repo := t.TempDir()

	three := []intent.Intention{noteOn(7, "first gap"), noteOn(7, "second gap"), noteOn(7, "third gap")}
	kept, _, said := bounded(repo, b, nil, three)
	if got := notesKept(kept, 0); len(got) != 1 {
		t.Errorf("three notes: want one comment, got %d", len(got))
	}
	if len(said) < 2 {
		t.Errorf("three notes: want the two others said, got %v", said)
	}

	long := strings.Repeat("x", 200000)
	kept, _, _ = bounded(repo, b, nil, []intent.Intention{noteOn(7, long)})
	got := notesKept(kept, 0)
	if len(got) != 1 {
		t.Fatalf("long note: want one comment, got %d", len(got))
	}
	if body, _ := got[0]["body"].(string); len(body) == 0 || len(body) >= len(long) {
		t.Errorf("long note: body of %d bytes is not cut to a limit", len(body))
	}
}

// Not elsewhere: Given a note aimed at another issue, When the developer
// applies it, Then it is refused, as today.
func TestNotElsewhere(t *testing.T) {
	b := Bounds{Issue: 7}
	kept, _, said := bounded(t.TempDir(), b, nil, []intent.Intention{noteOn(99, "elsewhere")})
	if got := notesKept(kept, 0); len(got) != 0 {
		t.Errorf("a note on another issue was kept: %v", got)
	}
	if len(said) == 0 {
		t.Fatal("the refusal was not said")
	}
	if !strings.Contains(strings.Join(said, "\n"), "99") {
		t.Errorf("the refusal does not name the other issue: %v", said)
	}
}

// Plain note (#902): Given the developer on #7, When its agent writes a
// note as plain text, or as a map naming no issue, Then the note is kept
// on #7.
func TestPlainNoteIsOnItsOwnIssue(t *testing.T) {
	b := Bounds{Issue: 7}
	for name, v := range map[string]any{
		"plain text":    "Issue #7 is not done: the bounds do not hold its scope.",
		"map, no issue": map[string]any{"body": "Issue #7 is not done: the bounds do not hold its scope."},
	} {
		kept, _, said := bounded(t.TempDir(), b, nil, []intent.Intention{{Kind: "note", Value: v}})
		got := notesKept(kept, 0)
		if len(got) != 1 || fmt.Sprint(got[0]["issue"]) != "7" {
			t.Errorf("%s: want one comment on #7, got %v (said %v)", name, got, said)
			continue
		}
		if body, _ := got[0]["body"].(string); !strings.Contains(body, "the bounds do not hold its scope") {
			t.Errorf("%s: body %q lost the note", name, body)
		}
	}
}

// Note in the answer: Given a merge round's task, When its agent reads how
// to say what it kept from each side, Then it is told a note is an item of
// its answer, never a file it writes (#1371).
func TestConflictNoteInTheAnswer(t *testing.T) {
	r := round{MR: 4, Branch: "workline/developer/3", Conflicts: []string{"lib.sh"}}
	task := conflictTask(r, "main", forge.Issue{ID: 3, Title: "Cycles"}, Bounds{Issue: 3}, Settings{}, "sh run-tests.sh {report}", nil)
	for _, want := range []string{"In your answer, a `note` a file in conflict", "never a file you write", "Write nothing outside your worktree"} {
		if !strings.Contains(task, want) {
			t.Errorf("the task lacks %q:\n%s", want, task)
		}
	}
	if strings.Contains(task, "Answer a `note` a file") {
		t.Errorf("the task still reads as a file to write:\n%s", task)
	}
}
