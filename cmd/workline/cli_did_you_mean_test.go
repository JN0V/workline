package main

import (
	"os"
	"strings"
	"testing"
)

// Typo
//
// Given workline installed,
// When a person runs `workline revew`,
// Then the error says `did you mean "review"?` and the exit code stays 64.
func TestTypo(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	errOut, code := captureStderr(t, func() int {
		return dispatch([]string{"revew"})
	})
	if code != 64 {
		t.Errorf("workline revew: exit %d, want 64", code)
	}
	if !strings.Contains(errOut, `did you mean "review"?`) {
		t.Errorf("error does not suggest review, got %q", errOut)
	}
}

// Help typo
//
// Given workline installed,
// When a person runs `workline help docter`,
// Then the error says `did you mean "doctor"?`.
func TestHelpTypo(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	errOut, _ := captureStderr(t, func() int {
		return dispatch([]string{"help", "docter"})
	})
	if !strings.Contains(errOut, `did you mean "doctor"?`) {
		t.Errorf("error does not suggest doctor, got %q", errOut)
	}
}
