package tester

import (
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/kit/verdict"
)

// Guard written: Given a scenario kept as today that no existing test
// checks, When the tester writes the issue's tests, Then it is no longer
// said that a test of it "is not written": its Draft holds a guard test
// of it, green on the base. A scenario kept as today that an existing
// test checks keeps its reason, naming that test.
func TestKeptScenarioWithNoGuardIsGivenOne(t *testing.T) {
	repo, run := gitRepo(t, map[string]string{
		"add.go":      "package calc\n\nfunc Add(a, b int) int { return a + b }\n",
		"add_test.go": "package calc\n\nimport \"testing\"\n\nfunc TestAddTwoNumbers(t *testing.T) {\n\tif Add(1, 2) != 3 {\n\t\tt.Fatal(\"sum\")\n\t}\n}\n",
	})
	base := run(repo, "rev-parse", "HEAD")
	s := Settings{Paths: []string{"**/*_test.go"}}
	said := map[string]string{
		"Late fee":  "kept as today: no existing test checks it",
		"Add twice": "kept as today: guarded by `TestAddTwoNumbers`",
	}
	guardedSaid(repo, base, s, said)
	for _, bad := range []string{"is not written", "nothing guards it"} {
		if strings.Contains(said["Late fee"], bad) {
			t.Errorf("a scenario kept as today with no guard is said %q, when the tester writes its guard test: %q", bad, said["Late fee"])
		}
	}
	if got := said["Add twice"]; got != "kept as today: guarded by `TestAddTwoNumbers`" {
		t.Errorf("a scenario guarded by an existing test keeps its reason, naming it: %q", got)
	}
}

// Guard written, a test shared: Given one test declared both as the guard
// of a scenario kept as today and as the test of a scenario that is not
// a guard, green on the base, When the guards are read for the red proof,
// Then that test is not one of them, so the proof still refuses it as
// green on the base; a test declared only as a guard is one.
func TestSharedTestOfGuardAndNonGuardScenarioIsNotAGuard(t *testing.T) {
	covers := []declared{
		{Scenario: "Late fee", Tests: []string{"TestShared", "TestOnlyGuard"}, Guard: true},
		{Scenario: "Early fee", Tests: []string{"TestShared"}},
	}
	newTests := []string{"TestShared", "TestOnlyGuard"}
	got := guardsOf(covers, newTests, []string{"fee_test.go"})
	if joined := strings.Join(got, ","); joined != "TestOnlyGuard" {
		t.Errorf("guards = %q, want only %q: a test also declared for a scenario that is not a guard must stay refused when green on the base", joined, "TestOnlyGuard")
	}
}

// Guard written, a proof refused for another reason: Given guards
// declared that match no finding of the proof, and a proof refused for a
// reason other than a guard's green run (no blocking finding left, only a
// warning), When the guards are allowed for the red proof, Then the proof
// stays refused, its findings untouched.
func TestGuardsMatchingNoFindingLeaveRefusedProofRefused(t *testing.T) {
	proof := &verdict.Verdict{
		Status: verdict.Block,
		Findings: []verdict.Finding{
			{Rule: "no-new-test", Level: "warn", Where: "fee_test.go", Message: "no test of the files ran"},
		},
	}
	allowGuards(proof, []string{"TestOnlyGuard"})
	if proof.Status != verdict.Block {
		t.Errorf("status = %q, want %q: guards that match no finding must not let a proof refused for another reason pass", proof.Status, verdict.Block)
	}
	if len(proof.Findings) != 1 || proof.Findings[0].Rule != "no-new-test" {
		t.Errorf("findings = %+v, want the one finding left as it was", proof.Findings)
	}
}
