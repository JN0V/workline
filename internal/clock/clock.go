// Package clock is the one seam through which the line reads the time
// (ADR-0063, step 1): the line, the product owner and the backlog read
// Now, never time.Now, so the line simulator runs them on its own clock.
package clock

import (
	"os"
	"time"
)

// Fixed names the variable that fixes the time, RFC 3339: the line
// simulator's clock, which the role scripts the engine runs inherit.
// Unset, or not a time, the machine's clock.
const Fixed = "WORKLINE_CLOCK"

// Now is the time; a test sets its own.
var Now = func() time.Time {
	if v := os.Getenv(Fixed); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t
		}
	}
	return time.Now()
}

// Since is the time elapsed since t, on Now.
func Since(t time.Time) time.Duration { return Now().Sub(t) }
