package main

import (
	"os"
	"strings"
	"testing"
)

// Missing folder
//
// Given no folder `/nonexistent`,
// When I run `workline doctor --repo /nonexistent`,
// Then it says that folder does not exist, and exits 64.
func TestMissingFolder(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	if _, err := os.Stat("/nonexistent"); err == nil {
		t.Skip("/nonexistent exists on this machine")
	}
	var code int
	var out string
	errOut, _ := captureStderr(t, func() int {
		out, code = captureStdout(t, func() int {
			return dispatch([]string{"doctor", "--repo", "/nonexistent"})
		})
		return code
	})
	if code != 64 {
		t.Errorf("workline doctor --repo /nonexistent: exit %d, want 64", code)
	}
	all := strings.ToLower(out + errOut)
	if !strings.Contains(all, "does not exist") {
		t.Errorf("no line saying the folder does not exist, got stdout %q, stderr %q", out, errOut)
	}
}
