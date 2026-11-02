package main

import (
	"runtime"
	"strings"
	"testing"
)

// Scenario "the verbose form names the build":
// Given an engine built for linux/amd64 with go1.27.1,
// When a person runs `workline version --verbose`,
// Then the first line is the version, as `workline version` prints it,
// And the second line is `go1.27.1 linux/amd64`.
// The Go release and platform are the ones of the machine running the test.
func TestTheVerboseFormNamesTheBuild(t *testing.T) {
	plain, code := captureStdout(t, func() int { return dispatch([]string{"version"}) })
	if code != 0 {
		t.Fatalf("workline version: exit %d", code)
	}
	version := strings.TrimSuffix(plain, "\n")

	out, code := captureStdout(t, func() int { return dispatch([]string{"version", "--verbose"}) })
	if code != 0 {
		t.Fatalf("workline version --verbose: exit %d, stdout %q", code, out)
	}
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("want two lines, got %d: %q", len(lines), out)
	}
	if lines[0] != version {
		t.Errorf("first line = %q, want the version %q", lines[0], version)
	}
	want := runtime.Version() + " " + runtime.GOOS + "/" + runtime.GOARCH
	if lines[1] != want {
		t.Errorf("second line = %q, want %q", lines[1], want)
	}
}
