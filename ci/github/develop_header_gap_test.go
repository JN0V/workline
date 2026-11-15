package github

import (
	"os"
	"strings"
	"testing"
)

// heartbeatHeader returns the header's entry for the heartbeat of a
// workflow file: its comment lines from `#   heartbeat` up to
// `# Why each part`, the comment marks dropped, the spaces collapsed.
func heartbeatHeader(t *testing.T, file string) string {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var words []string
	in := false
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "#   heartbeat") {
			in = true
		} else if strings.HasPrefix(line, "# Why each part") {
			break
		}
		if in {
			words = append(words, strings.Fields(strings.TrimPrefix(line, "#"))...)
		}
	}
	if len(words) == 0 {
		t.Fatalf("%s: no heartbeat entry found in the header", file)
	}
	return strings.ToLower(strings.Join(words, " "))
}

// Header true
// Given the header of the develop workflow (the template and the
// project's own copy),
// When a maintainer reads how to set the heartbeat's gap,
// Then it names the environment's wait timer, and says the variable only
// gates a reconcile started by the schedule or an event.
func TestDevelopHeaderSaysTheWaitTimerSetsTheGap(t *testing.T) {
	for _, file := range []string{"workline-develop.yml", "../../.github/workflows/workline-develop.yml"} {
		h := heartbeatHeader(t, file)
		if !strings.Contains(h, "wait timer") {
			t.Errorf("%s: the header's heartbeat entry does not name the environment's wait timer: %q", file, h)
		}
		if !strings.Contains(h, "workline_reconcile_gap_minutes") {
			t.Errorf("%s: the header's heartbeat entry does not name the variable: %q", file, h)
		}
		if strings.Contains(h, "55 min, or the variable") {
			t.Errorf("%s: the header still offers the variable as another way to set the heartbeat's gap: %q", file, h)
		}
		for _, want := range []string{"only", "schedule", "event"} {
			if !strings.Contains(h, want) {
				t.Errorf("%s: the header's heartbeat entry does not say the variable only gates a reconcile started by the schedule or an event (no %q): %q", file, want, h)
			}
		}
	}
}
