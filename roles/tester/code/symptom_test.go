package tester

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A test file the Draft and main both changed since the record's base is
// merged three ways: lines apart merge, the same line changed conflicts
// (#1212).
func TestSymptomMergesATestFileBothChanged(t *testing.T) {
	repo, run := gitRepo(t, map[string]string{"t.sh": "a\nb\nc\nd\ne\n"})
	commit := func(text, msg string) string {
		if err := os.WriteFile(filepath.Join(repo, "t.sh"), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		run(repo, "-c", "user.name=T", "-c", "user.email=t@example.invalid", "commit", "-qam", msg)
		return run(repo, "rev-parse", "HEAD:t.sh")
	}
	base := run(repo, "rev-parse", "HEAD")
	ours := commit("a\nb\nc\nd\ne\nrepro\n", "draft")
	run(repo, "checkout", "-q", base)
	theirs := commit("main\na\nb\nc\nd\ne\n", "main")
	text, err := merge3(repo, base, "t.sh", theirs, ours)
	if err != nil || text != "main\na\nb\nc\nd\ne\nrepro\n" {
		t.Errorf("merged = %q, %v; want both lines", text, err)
	}
	run(repo, "checkout", "-q", base)
	clash := commit("a\nb\nc\nd\ne\nother\n", "main again")
	if _, err := merge3(repo, base, "t.sh", clash, ours); err == nil {
		t.Error("the same line changed on both sides merged; want a conflict")
	}
}

// With an origin, the default branch is read from it: one that cannot be
// fetched is an error, never the clone's own copy, maybe behind.
func TestSymptomTipNeverFallsBackOnAStaleBranch(t *testing.T) {
	repo, run := gitRepo(t, map[string]string{"t.sh": "a\n"})
	head := run(repo, "rev-parse", "HEAD")
	if got, err := tip(repo, "main"); err != nil || got != head {
		t.Fatalf("no origin: tip = %q, %v; want the local %s", got, err, head)
	}
	run(repo, "remote", "add", "origin", filepath.Join(t.TempDir(), "gone.git"))
	if got, err := tip(repo, "main"); err == nil || !strings.Contains(err.Error(), "origin could not be read") {
		t.Errorf("origin unreachable: tip = %q, %v; want an error", got, err)
	}
}
