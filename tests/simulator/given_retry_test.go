package simulator

import "testing"

// TestGivenCodeKeptAtRetry: Given a tester that named the code it lacked,
// run again at once given it, When that run fails and is retried at the
// next tick, Then the retry is given that code still: never run again on
// the input it lacked it on (#1875).
func TestGivenCodeKeptAtRetry(t *testing.T) {
	holdsUnder(t, "trial-16-fresh", pI2, []string{"progress,machine-error,needs-engine,machine-error"})
}
