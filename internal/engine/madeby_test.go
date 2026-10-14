package engine

import "testing"

// The committer's trailer marks a person's commit whose message it
// rewrote (ADR-0046): left out of the roles that made a commit, wherever
// workline tells its own commits — the line, the merge policy, follow.
func TestMadeBy(t *testing.T) {
	for in, want := range map[string]string{
		"documentalist":            "documentalist",
		"committer":                "",
		"Committer":                "",
		" documentalist,committer": "documentalist",
		"committer, reviewer":      "reviewer",
		"":                         "",
	} {
		if got := madeBy(in); got != want {
			t.Errorf("madeBy(%q) = %q, want %q", in, got, want)
		}
	}
}
