package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// isolate points git and the home at nothing of the person's.
func isolate(t *testing.T) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
}

func countOf(r *Report, level string) int {
	n := 0
	for _, c := range r.Checks {
		if c.Level == level {
			n++
		}
	}
	return n
}

// One warning: Given a repository where doctor finds one warning and no
// problem, When I run `workline doctor`, Then its last line reads
// `1 warning: some checks do not run`.
func TestOneWarning(t *testing.T) {
	isolate(t)
	r := Run(Options{})
	if countOf(r, Warn) != 1 || countOf(r, Error) != 0 {
		t.Fatalf("setup: want 1 warning and no problem, got %+v", r.Checks)
	}
	if want := "1 warning: some checks do not run"; r.Summary != want {
		t.Fatalf("summary = %q, want %q", r.Summary, want)
	}
}

// Several warnings: Given a repository where doctor finds two warnings,
// When I run `workline doctor`, Then its last line reads
// `2 warnings: some checks do not run`.
func TestSeveralWarnings(t *testing.T) {
	isolate(t)
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".workline"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := "forge: cmd:workline-no-such-forge\n"
	if err := os.WriteFile(filepath.Join(repo, ".workline", "config.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	r := Run(Options{Repo: repo})
	if countOf(r, Error) != 0 {
		t.Fatalf("setup: want no problem, got %+v", r.Checks)
	}
	n := countOf(r, Warn)
	if n < 2 {
		t.Fatalf("setup: want at least 2 warnings, got %+v", r.Checks)
	}
	if !strings.HasSuffix(r.Summary, ": some checks do not run") || !strings.HasPrefix(r.Summary, "2 warnings") {
		t.Fatalf("summary = %q, want %q (warnings found: %d)", r.Summary, "2 warnings: some checks do not run", n)
	}
}
