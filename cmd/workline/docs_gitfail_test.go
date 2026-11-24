//go:build linux

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeGitOnPath puts a `git` first on PATH that is the real git, except for
// the listing of the changed docs (`git diff --name-only -- *.md`). With
// listed set, the first listing succeeds and prints it; every other listing
// fails with the exit code and, when msg is not empty, msg on stderr.
func fakeGitOnPath(t *testing.T, listed, msg, code string) {
	t.Helper()
	real, err := execLookGit()
	if err != nil {
		t.Skipf("no git: %v", err)
	}
	dir := t.TempDir()
	script := `#!/bin/sh
d=0; n=0; m=0
for a in "$@"; do
  case "$a" in
    diff) d=1;;
    --name-only) n=1;;
    '*.md') m=1;;
  esac
done
if [ $d = 1 ] && [ $n = 1 ] && [ $m = 1 ]; then
  echo x >> "$FAKE_GIT_COUNT"
  calls=$(wc -l < "$FAKE_GIT_COUNT" | tr -d ' ')
  if [ "$calls" -le "$FAKE_GIT_OK" ]; then
    printf '%s\n' "$FAKE_GIT_LISTED"
    exit 0
  fi
  if [ -n "$FAKE_GIT_MSG" ]; then
    echo "$FAKE_GIT_MSG" >&2
  fi
  exit "$FAKE_GIT_CODE"
fi
exec "$FAKE_GIT_REAL" "$@"
`
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	ok := "0"
	if listed != "" {
		ok = "1"
	}
	t.Setenv("FAKE_GIT_REAL", real)
	t.Setenv("FAKE_GIT_COUNT", filepath.Join(t.TempDir(), "count"))
	t.Setenv("FAKE_GIT_OK", ok)
	t.Setenv("FAKE_GIT_LISTED", listed)
	t.Setenv("FAKE_GIT_MSG", msg)
	t.Setenv("FAKE_GIT_CODE", code)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// gitFailFinding returns the message of the failure that stdout reports in
// JSON: the status must be block; rule is the rule of the finding sought
// ("" : the first finding).
func gitFailFinding(t *testing.T, stdout, stderr, rule string) string {
	t.Helper()
	var res struct {
		Status   string `json:"status"`
		Summary  string `json:"summary"`
		Findings []struct {
			Rule    string `json:"rule"`
			Message string `json:"message"`
		} `json:"findings"`
	}
	if err := json.Unmarshal([]byte(stdout), &res); err != nil {
		t.Fatalf("stdout is not the JSON result: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if res.Status != "block" {
		t.Errorf("status = %q, want block\n%s", res.Status, stdout)
	}
	for _, f := range res.Findings {
		if rule == "" || f.Rule == rule {
			return f.Message
		}
	}
	t.Errorf("no finding of rule %q\n%s", rule, stdout)
	return ""
}

// Listing fails, review-only
// Given a repository where the `git diff` that lists the changed docs fails,
// When `workline docs --review --json` runs,
// Then it exits 1 and reports a `git-failed` finding whose message holds
// git's error text, not "no doc changed".
func TestDocsReviewListingFails(t *testing.T) {
	sinceEnv(t)
	bin := sinceBuild(t)
	repo := docsJSONRepo(t)
	const said = "fatal: simulated failure of the listing"
	fakeGitOnPath(t, "", said, "128")

	stdout, stderr, code := sweepRun(t, bin, repo, "docs", "--repo", repo, "--review", "--json")
	if code != 1 {
		t.Errorf("exit code = %d, want 1\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	msg := gitFailFinding(t, stdout, stderr, "git-failed")
	if !strings.Contains(msg, said) {
		t.Errorf("message = %q, want git's error %q", msg, said)
	}
	if !strings.Contains(stderr, said) {
		t.Errorf("stderr lacks git's error %q:\n%s", said, stderr)
	}
	if strings.Contains(stderr, "no doc changed") {
		t.Errorf("stderr says no doc changed:\n%s", stderr)
	}
}

// Listing fails, before review
// Given the documentalist run ended and the `git diff` that lists the changed
// docs fails,
// When `workline docs --json` runs,
// Then it exits 1 and reports a failure with git's error, not a pass.
func TestDocsListingFailsBeforeReview(t *testing.T) {
	sinceEnv(t)
	bin := sinceBuild(t)
	repo := docsJSONRepo(t)
	const said = "fatal: simulated failure of the listing"
	fakeGitOnPath(t, "", said, "128")

	stdout, stderr, code := sweepRun(t, bin, repo, "docs", "--repo", repo, "--json", "--ai", "none")
	if code != 1 {
		t.Errorf("exit code = %d, want 1\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	msg := gitFailFinding(t, stdout, stderr, "")
	if !strings.Contains(msg, said) {
		t.Errorf("message = %q, want git's error %q", msg, said)
	}
}

// Listing fails, after review
// Given a person reviewed the changed docs and the `git diff` that lists the
// docs left fails,
// When `workline docs` runs,
// Then it exits 1 and does not say the docs are judged.
func TestDocsListingFailsAfterReview(t *testing.T) {
	sinceEnv(t)
	bin := sinceBuild(t)
	repo := docsJSONRepo(t)
	const said = "fatal: simulated failure of the listing"
	// The first listing names a doc, the person drops it; the second fails.
	fakeGitOnPath(t, "docs/a.md", said, "128")

	stdout, stderr, code := docsOnTerminal(t, bin, repo, "n\n", "docs", "--repo", repo, "--ai", "none")
	if code != 1 {
		t.Errorf("exit code = %d, want 1\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	if strings.Contains(stdout+stderr, "docs judged") {
		t.Errorf("it says the docs are judged:\n%s\n%s", stdout, stderr)
	}
	if ref := revParse(repo, docsJudgedRef); ref != "" {
		t.Errorf("%s was moved to %s although the docs left were not listed", docsJudgedRef, ref)
	}
}

// Git says nothing
// Given a failing `git diff` with an empty stderr,
// When `workline docs --json` runs,
// Then the message holds the error's own text, not an empty string.
func TestDocsListingFailsWithEmptyStderr(t *testing.T) {
	sinceEnv(t)
	bin := sinceBuild(t)
	repo := docsJSONRepo(t)
	fakeGitOnPath(t, "", "", "1")

	stdout, stderr, code := sweepRun(t, bin, repo, "docs", "--repo", repo, "--json", "--ai", "none")
	if code != 1 {
		t.Errorf("exit code = %d, want 1\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	msg := gitFailFinding(t, stdout, stderr, "")
	if strings.TrimSpace(msg) == "" || !strings.Contains(msg, "exit status 1") {
		t.Errorf("message = %q, want the error's own text (exit status 1)", msg)
	}
}

// execLookGit is the path of the real git.
func execLookGit() (string, error) { return lookPath("git") }
