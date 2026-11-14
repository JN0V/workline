package forge

import (
	"testing"
	"time"
)

// A command that answered and exited 0, leaving a child on its output, is
// reachable: its answer is kept.
func TestCheckCommandFinishedWithChildIsReachable(t *testing.T) {
	start := time.Now()
	state, detail := CheckCommand(t.TempDir(), `sleep 5 & echo '{}'`)
	if state != ProbeReachable {
		t.Fatalf("state = %s (%s), want reachable", state, detail)
	}
	if time.Since(start) > 4*time.Second {
		t.Fatalf("took %s", time.Since(start))
	}
}
