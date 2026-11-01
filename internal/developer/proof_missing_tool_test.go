package developer

import (
	"strings"
	"testing"
)

// Not run names the tool.
// Given a proof whose report names no test and a tool not found where it ran,
// When the developer says its proof,
// Then the finding `proof-not-run` names that tool.
func TestNotRunNamesTheTool(t *testing.T) {
	p := Proof{Result: Unproven, Why: "the report names no test", Missing: []string{"jq"}}
	f := proofFindings("#3", p, nil)
	if len(f) != 1 || f[0].Rule != "proof-not-run" {
		t.Fatalf("findings = %+v, want one proof-not-run", f)
	}
	if !strings.Contains(f[0].Message, "not found where it ran: jq") {
		t.Errorf("message %q does not name the tool jq (not found where it ran: jq)", f[0].Message)
	}
}

// Verification names the tool.
// Given the same proof (report names no test, a tool not found where it ran),
// When the developer writes its pull request's Verification,
// Then it lists the tool not found.
func TestVerificationNamesTheTool(t *testing.T) {
	p := Proof{Result: Unproven, Why: "the report names no test", Missing: []string{"jq"}}
	lines := proofLines(p, nil)
	if !strings.Contains(lines, "`jq`") {
		t.Errorf("Verification lines %q do not list the tool not found, jq", lines)
	}
}
