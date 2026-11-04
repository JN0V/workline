package main

import (
	"os"
	"testing"
)

// TestMain lets the test binary stand for `workline` when the engine runs a
// role's script with WORKLINE_BIN set to its own executable (`workline init`
// under test): "builtin ..." is dispatched, not run as test flags.
func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == "builtin" {
		os.Exit(dispatch(os.Args[1:]))
	}
	os.Exit(m.Run())
}
