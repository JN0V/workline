package main

import (
	"strings"
	"testing"
)

// Typo: Given workline installed, When a person runs `workline revew`, Then the error says `did you mean "review"?` and the exit code stays 64.
func TestTypo(t *testing.T) {
	var code int
	errOut, _ := captureStderr(t, func() int {
		code = dispatch([]string{"revew"})
		return code
	})
	if !strings.Contains(errOut, `did you mean "review"?`) {
		t.Errorf("stderr = %q, want it to say did you mean \"review\"?", errOut)
	}
	if code != 64 {
		t.Errorf("exit code = %d, want 64", code)
	}
}

// Help typo: Given workline installed, When a person runs `workline help docter`, Then the error says `did you mean "doctor"?`.
func TestHelpTypo(t *testing.T) {
	errOut, _ := captureStderr(t, func() int {
		return dispatch([]string{"help", "docter"})
	})
	if !strings.Contains(errOut, `did you mean "doctor"?`) {
		t.Errorf("stderr = %q, want it to say did you mean \"doctor\"?", errOut)
	}
}
