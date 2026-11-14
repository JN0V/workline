package forge

import (
	"testing"
	"time"
)

// Timeout kept: given a forge command that starts a child sleeping 30 s and
// a timeout of 200 ms, when it runs, then it returns a failure in under 2 s.
func TestCheckCommandTimeoutKeptWithChild(t *testing.T) {
	old := probeTimeout
	probeTimeout = 200 * time.Millisecond
	t.Cleanup(func() { probeTimeout = old })
	repo := probeRepo(t, "forge.sh", "sleep 30\ntrue")
	start := time.Now()
	state, detail := CheckCommand(repo, "./forge.sh")
	elapsed := time.Since(start)
	if state != ProbeFailing {
		t.Fatalf("got %s %q, want failing", state, detail)
	}
	if elapsed >= 2*time.Second {
		t.Fatalf("returned after %s, want under 2s: the child held the command", elapsed)
	}
}
