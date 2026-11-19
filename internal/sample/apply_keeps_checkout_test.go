package sample

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/kit/verdict"
)

// docBeforePutBack is a doc whose `checked` the sample found false, as it
// stands in the person's clone.
const docBeforePutBack = "---\nsources: [a.go]\nchecked: 111\n---\n# A\n\nOne hour.\n"

// applyWatched runs `Apply` of a sample with one false `checked`, in a
// clone on main, with a forge kept in the clone. A pre-commit hook of the
// clone records, each time a commit is made, the branch the clone's own
// checkout is on, and what its working tree shows changed. It returns the
// clone and those records: "head <branch>" and "status <porcelain line>".
func applyWatched(t *testing.T) (repo string, seen []string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the watch is a shell hook")
	}
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	repo = t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-q", "-b", "main")
	git("config", "user.name", "Tester")
	git("config", "user.email", "tester@example.invalid")
	if err := os.MkdirAll(filepath.Join(repo, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, text := range map[string]string{"docs/a.md": docBeforePutBack, "a.go": "package a\n"} {
		if err := os.WriteFile(filepath.Join(repo, filepath.FromSlash(name)), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git("add", ".")
	git("commit", "-q", "-m", "docs: start")

	logFile := filepath.Join(t.TempDir(), "seen.log")
	hook := fmt.Sprintf(`#!/bin/sh
unset GIT_DIR GIT_INDEX_FILE GIT_WORK_TREE GIT_PREFIX GIT_COMMON_DIR
{
  echo "head $(git --no-optional-locks -C '%[1]s' symbolic-ref -q --short HEAD || echo detached)"
  git --no-optional-locks -C '%[1]s' status --porcelain | sed 's/^/status /'
} >> '%[2]s'
exit 0
`, repo, logFile)
	hooks := filepath.Join(repo, ".git", "hooks")
	if err := os.MkdirAll(hooks, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hooks, "pre-commit"), []byte(hook), 0o755); err != nil {
		t.Fatal(err)
	}

	res := Result{Status: verdict.Pass, Week: "2026-W40", Window: "the week", Vouched: 1, Reads: []Read{{
		Doc: "docs/a.md", Commit: "abc1234", Checked: map[string]string{"": "111"}, Previous: map[string]string{"": "000"}, Verdict: "false",
	}}}
	data, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "sample.json")
	if err := os.WriteFile(file, data, 0o644); err != nil {
		t.Fatal(err)
	}

	got := Apply(file, repo, &forge.Local{Repo: repo})

	if now := git("symbolic-ref", "--short", "HEAD"); now != "main" {
		t.Errorf("after Apply, HEAD is on %q, want main (%s)", now, got.Summary)
	}
	if branches := git("for-each-ref", "--format=%(refname:short)", "refs/heads/workline/"); branches == "" {
		t.Fatalf("Apply made no branch for the put-back: %s", got.Summary)
	} else if shown := git("show", strings.Fields(branches)[0]+":docs/a.md"); !strings.Contains(shown, "checked: 000") || !strings.Contains(shown, "judged: 111") {
		t.Errorf("the branch's doc does not hold the put-back `checked`:\n%s", shown)
	}
	raw, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("no commit was made while Apply ran (%s): %v", got.Summary, err)
	}
	return repo, strings.Split(strings.TrimSpace(string(raw)), "\n")
}

// Branch kept: Given a clone on `main` and a sample with a false `checked`,
// When I run `workline sample --apply`, Then `HEAD` stays on `main` the
// whole time: the clone's checkout is on main when the commit is made.
func TestApplyKeepsTheCloneOnItsBranch(t *testing.T) {
	_, seen := applyWatched(t)
	heads := 0
	for _, l := range seen {
		if !strings.HasPrefix(l, "head ") {
			continue
		}
		heads++
		if l != "head main" {
			t.Errorf("while the sample committed, the clone's checkout was %q, want %q", l, "head main")
		}
	}
	if heads == 0 {
		t.Errorf("the hook recorded no HEAD: %q", seen)
	}
}

// Files kept out: Given the same, When the sample writes the put-back docs,
// Then none changes in the person's working tree: the clone shows nothing
// changed when the commit is made, and the doc is as it was after.
func TestApplyKeepsTheWorkingTreeClean(t *testing.T) {
	repo, seen := applyWatched(t)
	for _, l := range seen {
		if strings.HasPrefix(l, "status ") {
			t.Errorf("while the sample committed, the person's working tree showed %q, want no change", l)
		}
	}
	got, err := os.ReadFile(filepath.Join(repo, "docs", "a.md"))
	if err != nil || string(got) != docBeforePutBack {
		t.Errorf("docs/a.md in the working tree is %q (%v), want it as it was", got, err)
	}
}
