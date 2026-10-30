package developer

import "testing"

// A test is a file whose name holds "test" or "spec" as a word, or in a
// test folder; a name that only holds the letters is code.
func TestIsTest(t *testing.T) {
	for p, want := range map[string]bool{
		"mul_test.go":           true,
		"test_mul.py":           true,
		"conftest.py":           true,
		"web/mul.spec.ts":       true,
		"src/MulTest.java":      true,
		"src/TestMul.java":      true,
		"tests/helpers.rs":      true,
		"a/__tests__/mul.js":    true,
		"latest.go":             false,
		"contest.go":            false,
		"attestation.go":        false,
		"inspect.go":            false,
		"src/Testimony.java":    false,
		"internal/spectrum.go":  false,
		"docs/specification.md": false,
	} {
		if got := isTest(p); got != want {
			t.Errorf("isTest(%q) = %v, want %v", p, got, want)
		}
	}
}

// In CI, the notice names the job's read-only token, not a person's
// account (ADR-0058); outside it, the person's rights.
func TestRightsInCI(t *testing.T) {
	for _, v := range []string{"GITHUB_ACTIONS", "GITLAB_CI", "CI"} {
		t.Setenv(v, "")
	}
	if got := rights(); got != "the agent runs with your account's rights; the forge's guards decide what merges" {
		t.Errorf("outside CI: %q", got)
	}
	t.Setenv("GITLAB_CI", "true")
	if got := rights(); got != "the agent runs with the CI job's read-only token; the forge's guards decide what merges" {
		t.Errorf("in CI: %q", got)
	}
}
