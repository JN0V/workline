package tester

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// A run's ids are kept by its tree, command and report: read back for
// the same three only, from a worktree too; a run that failed, or wrote no
// report, is never kept (#1338).
func TestKeptIDs(t *testing.T) {
	repo, git := gitRepo(t, map[string]string{"lib.sh": "add() { :; }\n"})
	ok := ran{results: map[string]result{"a": passed, "b": failed}, reports: []string{"r.xml"}}
	keepIDs(repo, "HEAD", "sh t.sh {report}", "r.xml", ok)
	ids, _, found := readIDs(repo, "HEAD", "sh t.sh {report}", "r.xml")
	if !found || len(ids) != 2 || !ids["a"] || !ids["b"] {
		t.Fatalf("kept ids = %v, %v; want a and b", ids, found)
	}
	if _, _, found := readIDs(repo, "HEAD", "sh other.sh {report}", "r.xml"); found {
		t.Error("another command read the ids")
	}
	if _, _, found := readIDs(repo, "HEAD", "sh t.sh {report}", "other.xml"); found {
		t.Error("another report read the ids")
	}
	// Another commit of the same tree is the same tree.
	commit := func(args ...string) {
		git(repo, append([]string{"-c", "user.name=T", "-c", "user.email=t@example.invalid", "commit", "-q"}, args...)...)
	}
	commit("--allow-empty", "-m", "empty")
	if _, _, found := readIDs(repo, "HEAD", "sh t.sh {report}", "r.xml"); !found {
		t.Error("a commit of the same tree did not read the ids")
	}
	tree := filepath.Join(t.TempDir(), "wt")
	git(repo, "worktree", "add", "-q", "--detach", tree, "HEAD")
	if _, _, found := readIDs(tree, "HEAD", "sh t.sh {report}", "r.xml"); !found {
		t.Error("a worktree did not read the main clone's ids")
	}
	if err := os.WriteFile(filepath.Join(repo, "lib.sh"), []byte("add() { echo; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	commit("-am", "change")
	if _, _, found := readIDs(repo, "HEAD", "sh t.sh {report}", "r.xml"); found {
		t.Error("another tree read the ids")
	}
	failedRun := ok
	failedRun.failure = "exit status 1"
	keepIDs(repo, "HEAD", "sh t.sh {report}", "r.xml", failedRun)
	noReport := ok
	noReport.reports = nil
	keepIDs(repo, "HEAD", "sh u.sh {report}", "r.xml", noReport)
	if _, _, found := readIDs(repo, "HEAD", "sh t.sh {report}", "r.xml"); found {
		t.Error("a failed run was kept")
	}
	if _, _, found := readIDs(repo, "HEAD", "sh u.sh {report}", "r.xml"); found {
		t.Error("a run with no report was kept")
	}
}

// Past keptIDsMax files, the oldest go.
func TestKeptIDsPruned(t *testing.T) {
	dir := t.TempDir()
	old := time.Now().Add(-time.Hour)
	for i := range keptIDsMax + 3 {
		p := filepath.Join(dir, fmt.Sprintf("%02d.json", i))
		if err := os.WriteFile(p, []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
		at := old.Add(time.Duration(i) * time.Second)
		if err := os.Chtimes(p, at, at); err != nil {
			t.Fatal(err)
		}
	}
	pruneIDs(dir)
	entries, _ := os.ReadDir(dir)
	if len(entries) != keptIDsMax {
		t.Fatalf("%d files left, want %d", len(entries), keptIDsMax)
	}
	for i := range 3 {
		if _, err := os.Stat(filepath.Join(dir, fmt.Sprintf("%02d.json", i))); err == nil {
			t.Errorf("%02d.json, among the oldest, kept", i)
		}
	}
}
