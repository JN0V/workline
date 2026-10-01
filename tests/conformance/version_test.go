package conformance

import (
	"os/exec"
	"strings"
	"testing"
)

// `workline version` says which engine runs: the release, set when it was
// built, else what Go recorded — so a CI log names the engine it used.
func TestVersion(t *testing.T) {
	out, err := exec.Command(engineBin, "version").CombinedOutput()
	if err != nil || strings.TrimSpace(string(out)) == "" || strings.Contains(string(out), "usage") {
		t.Fatalf("%v: %q", err, out)
	}
}
