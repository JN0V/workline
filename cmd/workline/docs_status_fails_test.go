//go:build linux

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// failStatusOnPath puts a `git` first on PATH that is the real git, except for
// the check of the docs for uncommitted changes
// (`git status --porcelain -- *.md`): that one prints msg on stderr and exits
// with status 128, as git does on a broken index or a lock.
func failStatusOnPath(t *testing.T, msg string) {
	t.Helper()
	real, err := lookPath("git")
	if err != nil {
		t.Skipf("no git: %v", err)
	}
	dir := t.TempDir()
	script := `#!/bin/sh
s=0; p=0; m=0
for a in "$@"; do
  case "$a" in
    status) s=1;;
    --porcelain) p=1;;
    '*.md') m=1;;
  esac
done
if [ $s = 1 ] && [ $p = 1 ] && [ $m = 1 ]; then
  echo "$FAKE_GIT_MSG" >&2
  exit 128
fi
exec "$FAKE_GIT_REAL" "$@"
`
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAKE_GIT_REAL", real)
	t.Setenv("FAKE_GIT_MSG", msg)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// Git status fails
// Given a repository where `git status` fails, for example a broken index or
// a lock,
// When `workline docs` runs,
// Then it stops with a git-failed error and the agent does not run.
func TestDocsStatusFailsStopsBeforeTheAgent(t *testing.T) {
	sinceEnv(t)
	bin := sinceBuild(t)
	repo := docsJSONRepo(t)
	const said = "fatal: simulated failure of the status"
	failStatusOnPath(t, said)

	stdout, stderr, code := sweepRun(t, bin, repo, "docs", "--repo", repo, "--json", "--ai", "none")
	if code != 1 {
		t.Errorf("exit code = %d, want 1\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	msg := gitFailFinding(t, stdout, stderr, "git-failed")
	if !strings.Contains(msg, said) {
		t.Errorf("message = %q, want git's error %q", msg, said)
	}
	if strings.Contains(stderr, "the documentalist judges the docs") {
		t.Errorf("the documentalist ran although git status failed:\n%s", stderr)
	}
}
