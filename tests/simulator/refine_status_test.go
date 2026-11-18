package simulator

import (
	"slices"
	"testing"
)

// statusUnder replays trace name under each script, checking only that
// every open issue on the line has a status after each tick, and fails
// the test for each script that leaves one with none.
func statusUnder(t *testing.T, name string, scripts []string) {
	i := slices.IndexFunc(traces, func(tr trace) bool { return tr.name == name })
	if i < 0 {
		t.Fatalf("no trace named %s", name)
	}
	for _, s := range scripts {
		t.Run(s, func(t *testing.T) {
			f, log := runTrace(t, traces[i], parseScript(s), checks{only: []string{pI1}, until: 1 << 30})
			if f != nil {
				t.Errorf("%s\nreplay: SIM_TRACE=%s SIM_SCRIPT=%q\n%s", f, name, s, log)
			}
		})
	}
}

// TestRefineStatusAfterCodeNamedTwice: Given an issue in refine whose
// product owner, run again in the same tick, names code it was already
// given, twice, When the next tick runs, Then the issue has a status
// saying why it waits, never none.
func TestRefineStatusAfterCodeNamedTwice(t *testing.T) {
	statusUnder(t, "trial-19", []string{
		"progress,needs-engine,needs-engine",
		"progress,needs-engine,needs-engine,needs-engine",
	})
}

// TestRefineStatusAfterUnreadableTwice: Given the same issue whose second
// run's answers are unreadable twice, When the next tick runs, Then the
// issue is due again within the error budget, or blocked with the reason.
func TestRefineStatusAfterUnreadableTwice(t *testing.T) {
	statusUnder(t, "trial-19", []string{
		"progress,malformed,malformed",
	})
}
