package committer

import (
	"fmt"
	"testing"
)

// A line git could read as workline's own trailer is found wherever it
// stands, however its key is cased or spaced (ADR-0046).
func TestOwnLines(t *testing.T) {
	msg := "fix: a thing\n\nWorkline-Role:committer in the body\n\nRefs: AC-3\n  workline-model : claude:x\nWorkline-Was: fix: AC-3"
	want := "[workline-role: committer in the body workline-model: claude:x workline-was: fix: AC-3]"
	if got := fmt.Sprint(ownLines(msg)); got != want {
		t.Errorf("ownLines = %s, want %s", got, want)
	}
	if got := ownLines("fix: a thing\n\nRefs: AC-3"); len(got) != 0 {
		t.Errorf("no workline trailer: %v", got)
	}
}
