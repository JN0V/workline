package tester

import (
	"strings"
	"testing"
)

func TestScenarios(t *testing.T) {
	got := Scenarios("Intro line.\n\n- **Old debt:** Given `main` with 140 findings, When a merge request\n  adds one, Then only that one blocks.\n- **Moved code**: Given a move, When the gate runs, Then nothing new.\n- Given no name, When read, Then dropped.\n")
	want := []Scenario{
		{Name: "Old debt", Text: "Given `main` with 140 findings, When a merge request adds one, Then only that one blocks."},
		{Name: "Moved code", Text: "Given a move, When the gate runs, Then nothing new."},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("scenario %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestNormNamesAScenarioInCode(t *testing.T) {
	for _, code := range []string{"func TestOldDebt(t *testing.T)", "def test_old_debt():", "// Scenario: Old debt"} {
		if !strings.Contains(norm(code), norm("Old debt")) {
			t.Errorf("%q does not name the scenario", code)
		}
	}
}
