package forge

import "testing"

// A command that answered and exited 0 but left a child on its output is
// reachable, and its answer is kept, not failed after the wait delay.
func TestCheckCommandLeftoverChildIsNotFailing(t *testing.T) {
	repo := probeRepo(t, "forge.sh", "sleep 30 &\necho '{}'")
	state, detail := CheckCommand(repo, "./forge.sh")
	if state != ProbeReachable {
		t.Fatalf("got %s %q, want reachable", state, detail)
	}
}
