package sample

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

// The weekly sample shows the tester's Scope code at the base: a file
// named with a line, or by its bare name, is the file there. A path
// holding a space is one file, never two words a name could match.
func TestScopeAt(t *testing.T) {
	repo := t.TempDir()
	for _, f := range []string{"internal/b/parts.go", "my notes/ops.go", "x/ops.go"} {
		p := filepath.Join(repo, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("package b\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "."}, {"commit", "-qm", "base"}} {
		cmd := exec.Command("git", append([]string{"-C", repo, "-c", "user.name=t", "-c", "user.email=t@example.invalid", "-c", "core.hooksPath=/dev/null"}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	got := scopeAt(repo, "HEAD", "- `uncovered` (`parts.go:943`), `internal/b/parts.go:10-20`, `notes/ops.go`, `internal/b`.")
	want := []string{"uncovered", "internal/b/parts.go", "notes/ops.go", "internal/b"}
	if !slices.Equal(got, want) {
		t.Errorf("scopeAt = %q, want %q", got, want)
	}
	items := codeAt(repo, "HEAD", got, 100)
	if len(items) == 0 || items[0].name != "internal/b/parts.go" {
		t.Errorf("codeAt gives %d items, want internal/b/parts.go first", len(items))
	}
}
