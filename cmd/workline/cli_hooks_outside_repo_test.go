package main

import (
	"os"
	"strings"
	"testing"
)

// Outside a repository
//
// Given a folder that is not a git repository,
// When I run `workline hooks install --repo` in it,
// Then it says `not in a git repository` and exits 64.
func TestOutsideARepository(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	dir := t.TempDir()
	t.Chdir(dir)

	run := func(args ...string) (string, int) {
		var code int
		errOut, _ := captureStderr(t, func() int {
			code = dispatch(args)
			return code
		})
		return errOut, code
	}

	const want = "workline: not in a git repository"
	errOut, code := run("hooks", "install", "--repo")
	if code != 64 {
		t.Errorf("hooks install --repo: exit %d, want 64", code)
	}
	if !strings.Contains(errOut, want) {
		t.Errorf("hooks install --repo: stderr %q, want %q", errOut, want)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("the folder holds %d entries after the command, want none (first: %s)", len(entries), entries[0].Name())
	}

	// Parity: init and review say the same, with the same code.
	for _, c := range []string{"init", "review"} {
		o, k := run(c)
		if k != code || strings.TrimSpace(o) != strings.TrimSpace(errOut) {
			t.Errorf("%s: exit %d, stderr %q; hooks gave exit %d, stderr %q", c, k, o, code, errOut)
		}
	}
}
