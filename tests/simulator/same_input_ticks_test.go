package simulator

import "testing"

// TestJudgeAskedOncePerInputAcrossTicks: Given a tester whose tests do
// not build, judged once, When the tester runs again on the same broken
// build at the next tick, after a machine error, Then the judge is not
// asked again on that input (#1875).
func TestJudgeAskedOncePerInputAcrossTicks(t *testing.T) {
	holdsUnder(t, "trial-4-fresh", pI2, []string{"progress,broken,machine-error,broken"})
	holdsUnder(t, "workline-1004", pI2, []string{"machine-error,progress,broken,waiting,broken"})
}
