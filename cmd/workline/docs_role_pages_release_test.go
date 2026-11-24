package main_test

import (
	"os/exec"
	"strings"
	"testing"

	documentalist "github.com/JN0V/workline/roles/documentalist/code"
)

// The release 0.89.0 (e2a0bc95) only bumped `WORKLINE_VERSION` in the CI
// templates; pages of #2027 kept at `checked: 53a5d2c6`, saying no version,
// must not count it as a change of their sources.
func TestRolePagesCheckedMovedRelease0890(t *testing.T) {
	const release, checked = "e2a0bc95", "53a5d2c6"
	top, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		t.Fatal(err)
	}
	root := strings.TrimSpace(string(top))
	git := func(args ...string) (string, error) {
		out, err := exec.Command("git", append([]string{"-C", root}, args...)...).Output()
		return strings.TrimSpace(string(out)), err
	}
	for _, c := range []string{release, checked} {
		if _, err := git("rev-parse", "--verify", "-q", c+"^{commit}"); err != nil {
			t.Skipf("commit %s is not in this clone's history", c)
		}
	}
	files, _ := git("diff-tree", "--no-commit-id", "--name-only", "-r", release, "--", "defaults/ci")
	if files == "" {
		t.Fatalf("%s changed no CI template", release)
	}
	for _, path := range strings.Split(files, "\n") {
		out, _ := git("log", "--format=%h %s", checked+".."+release, "--", path)
		out = documentalist.DropReleaseBumps(root, out, path, "a page that states no version")
		if strings.Contains(out, release[:7]) {
			t.Errorf("%s: release %s counts as a change after %s", path, release, checked)
		}
	}
}
