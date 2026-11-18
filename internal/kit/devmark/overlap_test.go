package devmark

import (
	"testing"

	"github.com/JN0V/workline/internal/backlog"
)

// The line reads the developer's comment for an overlap it passed an
// issue over for (#998): that reason alone, as OverlapWhy says it.
func TestPassedForOverlap(t *testing.T) {
	for c, want := range map[string]bool{
		backlog.Said("**The developer**", "waits on this issue", OverlapWhy(5, "lib.sh"), "#5 merged or let go"):                       true,
		backlog.Said("**The developer**", "waits on this issue", "its acceptance tests are not written yet", "the tester writes them"): false,
		"**The developer** took this issue. Passed over: #2: " + OverlapWhy(5, "lib.sh"):                                               false,
	} {
		if got := PassedForOverlap(c); got != want {
			t.Errorf("PassedForOverlap(%q) = %v, want %v", c, got, want)
		}
	}
}
