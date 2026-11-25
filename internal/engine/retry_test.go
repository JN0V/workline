package engine

import (
	"strings"
	"testing"
	"time"
)

// WORKLINE_RETRY_PAUSE reads a duration or a number of seconds; anything
// else takes 60 s, said.
func TestRetryPause(t *testing.T) {
	for v, want := range map[string]time.Duration{"": time.Minute, "5": 5 * time.Second, "0": 0, "1.5": 1500 * time.Millisecond, "30s": 30 * time.Second, "soon": time.Minute, "-3": time.Minute} {
		t.Setenv("WORKLINE_RETRY_PAUSE", v)
		got, how := retryPause()
		if got != want {
			t.Errorf("%q: %s, want %s", v, got, want)
		}
		if bad := v == "soon" || v == "-3"; bad != strings.Contains(how, "60s taken") {
			t.Errorf("%q: said %q", v, how)
		}
	}
}
