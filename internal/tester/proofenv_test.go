package tester

import (
	"strings"
	"testing"
	"time"

	"github.com/JN0V/workline/internal/forge"
)

// The proof runs the project's tests with the job's environment, not
// the line's own settings: a run of the line sets its rate wait for its
// forge calls, and a project test reading the variable saw it (#940).
func TestProofLeavesTheLinesWaitBehind(t *testing.T) {
	t.Setenv(forge.RateWaitEnv, "5m0s")
	t.Setenv("WORKLINE_PROOF_KEPT", "yes")
	out, _, err := shell(t.TempDir(), "env", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), forge.RateWaitEnv+"=") {
		t.Errorf("the proof's command sees %s:\n%s", forge.RateWaitEnv, out.String())
	}
	if !strings.Contains(out.String(), "WORKLINE_PROOF_KEPT=yes") {
		t.Errorf("the proof's command lost the job's environment:\n%s", out.String())
	}
}
