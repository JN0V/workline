// Package clock is the one seam through which the line reads the time
// (ADR-0063, step 1): the line, the product owner and the backlog read
// Now, never time.Now, so the line simulator runs them on its own clock.
package clock

import (
	"fmt"
	"os"
	"sync"
	"time"
)

var warn sync.Once

// Fixed names the variable that fixes the time, RFC 3339: the line
// simulator's clock, which the role scripts the engine runs inherit.
// Unset, the machine's clock; not a time, the machine's clock too, said
// on stderr once.
const Fixed = "WORKLINE_CLOCK"

// Now is the time; a test sets its own.
var Now = func() time.Time {
	if v := os.Getenv(Fixed); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			return t
		}
		warn.Do(func() {
			fmt.Fprintf(os.Stderr, "workline: %s=%q is not an RFC 3339 time: the machine's clock is read\n", Fixed, v)
		})
	}
	return time.Now()
}

// Since is the time elapsed since t, on Now.
func Since(t time.Time) time.Duration { return Now().Sub(t) }
