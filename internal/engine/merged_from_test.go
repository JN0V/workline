package engine

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The developer's branch starts from main merged into the tester's Draft,
// which was behind it (#1941): the engine's own merge, two parents, the
// Draft first, its trailer the developer's, as the merge policy reads
// workline's own; a Draft main conflicts with makes none.
func TestMergedFromIsTheEnginesMerge(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	repo := t.TempDir()
	g := func(args ...string) string {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", repo, "-c", "user.name=T", "-c", "user.email=t@example.invalid"}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
		return strings.TrimSpace(string(out))
	}
	put := func(name, text, msg string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(filepath.Join(repo, name)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repo, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		g("add", "-A")
		g("commit", "-q", "-m", msg)
	}
	g("init", "-q", "-b", "main")
	g("config", "user.name", "T")
	g("config", "user.email", "t@example.invalid")
	put("internal/tester/proof.go", "package tester\n", "first")
	g("checkout", "-q", "-b", "workline/tester/issue-3")
	put("tests/proof_test.go", "package tests\n", "test: the acceptance tests")
	draft := g("rev-parse", "HEAD")
	g("checkout", "-q", "main")
	g("mv", "internal/tester", "roles")
	g("commit", "-q", "-m", "refactor: move")
	main := g("rev-parse", "HEAD")

	from, err := mergedFrom(repo, draft, main, "main")
	if err != nil {
		t.Fatal(err)
	}
	if parents := g("rev-list", "--parents", "-n", "1", from); parents != from+" "+draft+" "+main {
		t.Errorf("parents %q, want the Draft then main", parents)
	}
	if by := ownTrailer(repo, from); by != "developer" {
		t.Errorf("its trailer says %q, want developer", by)
	}
	if got := g("ls-tree", "-r", "--name-only", from); got != "roles/proof.go\ntests/proof_test.go" {
		t.Errorf("its tree holds %q, want main's paths and the Draft's tests", got)
	}

	put("tests/proof_test.go", "package other\n", "test: another")
	if _, err := mergedFrom(repo, draft, g("rev-parse", "HEAD"), "main"); err == nil || !strings.Contains(err.Error(), "tests/proof_test.go") {
		t.Errorf("a conflict: got %v, want it named", err)
	}
}
