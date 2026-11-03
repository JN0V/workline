package tester

import (
	"os"
	"strings"
	"testing"
)

// A tester's test that commits in a temporary repository went through
// the machine's global git config: on a machine with workline's global
// hooks, its commit was refused before the test checked anything (#941).
// The tester's instruction tells it to isolate git, as the project's
// own tests do.
func TestInstructionIsolatesGit(t *testing.T) {
	data, err := os.ReadFile("../../roles/tester/instruction.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)`) {
		t.Error("the tester's instruction does not tell it to point git's global config at the null device")
	}
}
