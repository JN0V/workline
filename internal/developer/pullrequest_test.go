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
