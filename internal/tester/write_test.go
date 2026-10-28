package tester

import (
	"slices"
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

func TestNamesAScenarioWhole(t *testing.T) {
	for _, code := range []string{"func TestOldDebt(t *testing.T)", "def test_old_debt():", "// Scenario: Old debt",
		`t.Run("Old debt", func(t *testing.T) {`, "func TestGate_OldDebt(t *testing.T)"} {
		if !names(code, "Old debt") {
			t.Errorf("%q does not name the scenario", code)
		}
	}
	for _, code := range []string{"func TestOldDebtGone(t *testing.T)", "def test_old_debt_gone():", "// Scenario: Old debt gone",
		"// the old debt is kept", `msg := "old debt and more"`, "func TestNewOldDebt(t *testing.T)"} {
		if names(code, "Old debt") {
			t.Errorf("%q names the scenario, though only a longer name or prose holds it", code)
		}
	}
	if names("// Scenario: !!", "!!") {
		t.Error("a name of no word is named")
	}
}

func TestClaimTakesTheLongestName(t *testing.T) {
	sc := []Scenario{{Name: "Minus one"}, {Name: "Minus one hundred"}}
	if i := claim(words("Minus one hundred: a person reads it"), sc, hasPrefix); i != 1 {
		t.Errorf("note claimed by %d, want 1", i)
	}
	if i := claim(words("test_minus_one"), sc, hasSuffix); i != 0 {
		t.Errorf("test claimed by %d, want 0", i)
	}
}

// A one-letter word in a camel-cased name: `TestSpaceInAName` names
// "Space in a name", `TestHTTPServer` "HTTP server".
func TestWordsOfCapitals(t *testing.T) {
	if !names("func TestSpaceInAName(t *testing.T)", "Space in a name") {
		t.Error("TestSpaceInAName does not name \"Space in a name\"")
	}
	if got := words("TestHTTPServer"); !slices.Equal(got, []string{"test", "http", "server"}) {
		t.Errorf("words(TestHTTPServer) = %q", got)
	}
}
