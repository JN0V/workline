package developer

import (
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/kit/devmark"
)

// whatItDid is the "What it did" part of a round note: from its heading
// to the closing line.
func whatItDid(t *testing.T, body string) string {
	t.Helper()
	_, after, ok := strings.Cut(body, "What it did:")
	if !ok {
		t.Fatalf("no \"What it did\" part in the round note:\n%s", body)
	}
	before, _, _ := strings.Cut(after, "The reviewer decides a finding is fixed.")
	return before
}

func refusedLine(file, rule string) string {
	return "`" + file + "`: refused at apply (" + rule + "); a person must write this change."
}

// Given a review round whose patch to a CODEOWNERS file is refused at
// apply, When its round note is posted, Then the note names that file,
// says it was refused and why, and that a person must write it.
func TestRoundNoteNamesARefusedChange(t *testing.T) {
	refused := []devmark.Refusal{{File: ".github/CODEOWNERS", Rule: "steering-file"}}
	notes := []string{"Added the new owner to `.github/CODEOWNERS`."}
	x := devmark.Round{N: 1, Result: "no-code", Asked: []string{"a finding"}}
	got := roundBody(x, Settings{Rounds: 3}, notes, "", nil, refused)
	did := whatItDid(t, got)
	if want := refusedLine(".github/CODEOWNERS", "steering-file"); !strings.Contains(did, want) {
		t.Errorf("What it did part lacks %q:\n%s", want, did)
	}
}

// Given the same round, its agent's note saying the refused file's
// change is done, When the round note is posted, Then the note does not
// say that change landed. A note naming both a refused and a landed
// file is replaced whole, a refused file's line is written once, and a
// note naming no refused file stays as it is.
func TestRoundNoteDoesNotClaimARefusedChange(t *testing.T) {
	refused := []devmark.Refusal{
		{File: "cmd/workline/wake.go", Rule: "steering-file"},
		{File: "internal/wake/issue.go", Rule: "steering-file"},
		{File: "internal/wake/wake.go", Rule: "steering-file"},
	}
	notes := []string{
		"The change to `.github/CODEOWNERS` is done.",
		"Added the flag in `cmd/workline/wake.go` and `internal/wake/issue.go`, and updated `internal/wake/render.go`.",
		"`internal/wake/wake.go` now wakes the issue; `cmd/workline/wake.go` is done too.",
		"Ran the project's tests.",
	}
	refused = append(refused, devmark.Refusal{File: ".github/CODEOWNERS", Rule: "steering-file"})
	x := devmark.Round{N: 1, Result: "no-code", Asked: []string{"a finding"}}
	got := roundBody(x, Settings{Rounds: 3}, notes, "", nil, refused)
	did := whatItDid(t, got)
	for _, claim := range []string{"is done", "internal/wake/render.go", "Added the flag", "now wakes the issue"} {
		if strings.Contains(did, claim) {
			t.Errorf("What it did part still holds %q:\n%s", claim, did)
		}
	}
	for _, r := range refused {
		line := refusedLine(r.File, r.Rule)
		if n := strings.Count(did, line); n != 1 {
			t.Errorf("line %q written %d times, want once:\n%s", line, n, did)
		}
	}
	if !strings.Contains(did, "- Ran the project's tests.\n") {
		t.Errorf("a note naming no refused file did not stay as it is:\n%s", did)
	}
}
