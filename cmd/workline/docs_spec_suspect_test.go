package main

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"testing"
)

// Specification docs are verified:
// Given the repository as it is on main once the first release is out,
// When `workline docs --ai none` judges the docs made suspect since the
// first commit,
// Then no doc under `docs/spec/` is listed as suspect.
//
// The repository is cloned first, so that the run's ref of the docs
// judged is never written into the one under test.
func TestSpecDocsHaveNoneSuspect(t *testing.T) {
	sinceEnv(t)
	bin := sinceBuild(t)
	src, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	clone := filepath.Join(t.TempDir(), "repo")
	gitIn(t, filepath.Dir(clone), "clone", "-q", "--no-hardlinks", src, clone)

	roots, err := exec.Command("git", "-C", clone, "rev-list", "--max-parents=0", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Fields(string(roots))
	if len(lines) == 0 {
		t.Fatal("the repository has no first commit")
	}
	first := lines[len(lines)-1]

	cmd := exec.Command(bin, "docs", "--repo", clone, "--since", first, "--ai", "none")
	cmd.Dir = clone
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true} // no /dev/tty
	var buf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &buf, &buf
	_ = cmd.Run()
	out := buf.String()

	if !strings.Contains(out, "judging the docs made suspect by the commits since") {
		t.Fatalf("the docs were not judged:\n%s", out)
	}
	suspect := regexp.MustCompile(`(?m)^\s*suspect docs/spec/\S+:.*$`)
	if found := suspect.FindAllString(out, -1); len(found) > 0 {
		t.Errorf("%d doc(s) under docs/spec/ are suspect:\n%s", len(found), strings.Join(found, "\n"))
	}
}
