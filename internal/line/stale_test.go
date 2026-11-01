package line

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/routing"
)

// TestScopeKind: paths and code names are checked; labels, settings,
// flags and plain words never are.
func TestScopeKind(t *testing.T) {
	for w, want := range map[string]string{
		"internal/line/states.go": "path", "board.go": "path", ".workline/config.yaml": "path", "internal/board": "path",
		"stateOf": "name", "Config.Wakes": "name", "multiply_all()": "name", "add()": "name", "routing.wip": "name",
		"workline:ready": "", "--ai": "", "none": "", "max-hops": "", "../x.go": "", "/etc/x.go": "", "*.md": "",
	} {
		if got := scopeKind(w); got != want {
			t.Errorf("scopeKind(%q) = %q, want %q", w, got, want)
		}
	}
}

// TestScopeGone: a Scope's path, folder and code name still in the
// repository are not gone; a deleted function, a missing file are; a
// name only Markdown keeps is gone.
func TestScopeGone(t *testing.T) {
	repo := t.TempDir()
	write := func(p, s string) {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(repo, p)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repo, p), []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("internal/line/states.go", "package line\nfunc stateOf() {}\n")
	write("src/café.go", "package src\n") // a name git quotes unless asked not to
	write("docs/adr.md", "the old `reconcileAll` is gone\n")
	for _, args := range [][]string{{"init", "-q"}, {"add", "."}} {
		if out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	body := "## Need\n\nx\n\n## Scope\n\n- `internal/line`, `states.go:12`, `src/café.go`, `stateOf()`, `reconcileAll()`, `gone.go`, `Config.Gone`, `workline:ready`.\n"
	got, err := newScopeIndex(repo).gone(body)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"reconcileAll()", "gone.go", "Config.Gone"}; !slices.Equal(got, want) {
		t.Errorf("gone = %q, want %q", got, want)
	}
}

// TestScopeUnread: a folder that is no repository is said, never read as
// a Scope all gone.
func TestScopeUnread(t *testing.T) {
	if got, err := newScopeIndex(t.TempDir()).gone("## Scope\n\n- `lib.sh`\n"); err == nil {
		t.Errorf("gone = %q, want an error", got)
	}
}

// TestLabeledAt: GitHub's and GitLab's times, a day alone, none.
func TestLabeledAt(t *testing.T) {
	for s, want := range map[string]bool{"2026-09-01T09:00:00Z": true, "2026-10-05T05:44:13.850Z": true, "2026-09-01": true, "": false, "soon": false} {
		if _, ok := labeledAt(s); ok != want {
			t.Errorf("labeledAt(%q) = %v, want %v", s, ok, want)
		}
	}
}

// TestDueKeepsStale: a stale move stays known while the issue's state
// is the same — a person put it back —, and goes once it moves on.
func TestDueKeepsStale(t *testing.T) {
	ready := routing.State{Name: "ready", Label: "workline:ready"}
	rec := Record{State: "ready", Body: "0", Labeled: "a", Hops: 1, Stale: "ready", Kept: "a"}
	now, ok := due(ready, forge.Issue{Body: "x"}, "b", "", rec, true)
	if !ok || now.Stale != "ready" || now.Kept != "a" {
		t.Errorf("same state: %+v, %v", now, ok)
	}
	now, _ = due(routing.State{Name: "refine"}, forge.Issue{Body: "x"}, "c", "", rec, true)
	if now.Stale != "" || now.Kept != "" {
		t.Errorf("state moved on: %+v", now)
	}
}
