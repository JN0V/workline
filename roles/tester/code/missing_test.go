package tester

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestMissingTools(t *testing.T) {
	bin := t.TempDir() // the PATH: one tool there, present-tool
	if err := os.WriteFile(filepath.Join(bin, "present-tool"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	for _, c := range []struct {
		name, output string
		want         []string
	}{
		{"workline's committer", "findings: [{secrets-not-checked  warn gitleaks is not installed, so neither secrets nor forbidden terms were looked for", []string{"gitleaks"}},
		{"workline's committer, in a Go struct", "(got [{Rule:secrets-not-checked Where: Level:warn Message:gitleaks is not installed, so", []string{"gitleaks"}},
		{"Go's exec", `run: exec: "lychee": executable file not found in $PATH`, []string{"lychee"}},
		{"bash", "bash: line 1: shellcheck: command not found\n", []string{"shellcheck"}},
		{"dash", "tests/test_scan.sh: 1: no-such-scanner: not found\n", []string{"no-such-scanner"}},
		{"each once, sorted", "sh: 1: zz: not found\nsh: 1: aa: not found\nsh: 1: zz: not found\n", []string{"aa", "zz"}},
		{"a plain red", "--- FAIL: TestAdd (0.00s)\n    add_test.go:9: got 4, want 5\n", nil},
		{"a tool the PATH finds: a test printing the message", "want \"present-tool: command not found\", got \"\"\n", nil},
		{"a package, not a tool", "error: package foo is not installed\n", nil},
	} {
		if got := MissingTools(c.output); !slices.Equal(got, c.want) {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}
