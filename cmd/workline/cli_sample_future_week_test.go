package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// Future week
//
// Given today is in 2026-W45,
// When I run `workline sample --week 2030-W01`,
// Then it blocks, says the week has not started, and exits 1.
func TestFutureWeek(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	repo := t.TempDir()
	if out, err := exec.Command("git", "-C", repo, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	var out string
	var code int
	errOut, _ := captureStderr(t, func() int {
		out, code = captureStdout(t, func() int {
			return dispatch([]string{"sample", "--repo", repo, "--week", "2030-W01"})
		})
		return code
	})
	if code != 1 {
		t.Errorf("workline sample --week 2030-W01: exit %d, want 1", code)
	}
	if !strings.Contains(out+errOut, "has not started") {
		t.Errorf("no line saying the week has not started, got stdout %q, stderr %q", out, errOut)
	}
}
