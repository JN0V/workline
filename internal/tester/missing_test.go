package tester

import (
	"slices"
	"testing"
)

func TestMissingTools(t *testing.T) {
	for _, c := range []struct {
		name, output string
		want         []string
	}{
		{"workline's committer", "findings: [{secrets-not-checked  warn gitleaks is not installed, so neither secrets nor forbidden terms were looked for", []string{"gitleaks"}},
		{"Go's exec", `run: exec: "lychee": executable file not found in $PATH`, []string{"lychee"}},
		{"bash", "bash: line 1: shellcheck: command not found\n", []string{"shellcheck"}},
		{"dash", "tests/test_scan.sh: 1: no-such-scanner: not found\n", []string{"no-such-scanner"}},
		{"each once, sorted", "sh: 1: zz: not found\nsh: 1: aa: not found\nsh: 1: zz: not found\n", []string{"aa", "zz"}},
		{"a plain red", "--- FAIL: TestAdd (0.00s)\n    add_test.go:9: got 4, want 5\n", nil},
	} {
		if got := MissingTools(c.output); !slices.Equal(got, c.want) {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}
