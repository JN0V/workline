package forge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// probeRepo writes an executable script in a new repository and returns it.
func probeRepo(t *testing.T, name, body string) string {
	t.Helper()
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, name), []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return repo
}

func TestCheckCommandMissing(t *testing.T) {
	repo := t.TempDir()
	state, detail := CheckCommand(repo, "workline-no-such-forge --flag")
	if state != ProbeMissing || !strings.Contains(detail, "workline-no-such-forge") {
		t.Fatalf("got %s %q, want missing naming the command", state, detail)
	}
	state, detail = CheckCommand(repo, "./nope.sh")
	if state != ProbeMissing || !strings.Contains(detail, "does not exist") {
		t.Fatalf("got %s %q, want missing: does not exist", state, detail)
	}
	if state, _ := CheckCommand(repo, "  "); state != ProbeMissing {
		t.Fatalf("an empty command is missing, got %s", state)
	}
}

func TestCheckCommandNotExecutable(t *testing.T) {
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "forge.sh"), []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	state, detail := CheckCommand(repo, "./forge.sh")
	if state != ProbeMissing || !strings.Contains(detail, "not executable") {
		t.Fatalf("got %s %q, want missing: not executable", state, detail)
	}
}

func TestCheckCommandReachable(t *testing.T) {
	repo := probeRepo(t, "forge.sh", `cat >/dev/null; echo '{}'`)
	if state, detail := CheckCommand(repo, "./forge.sh"); state != ProbeReachable {
		t.Fatalf("got %s %q, want reachable", state, detail)
	}
	// A command that does not know ping, but answers, still runs.
	repo = probeRepo(t, "forge.sh", `cat >/dev/null; echo '{"error":"unknown operation ping"}'`)
	if state, detail := CheckCommand(repo, "./forge.sh"); state != ProbeReachable {
		t.Fatalf("got %s %q, want reachable", state, detail)
	}
}

func TestCheckCommandRefused(t *testing.T) {
	repo := probeRepo(t, "forge.sh", `cat >/dev/null; echo '{"error":"FORGEJO_URL is not set"}'`)
	state, detail := CheckCommand(repo, "./forge.sh")
	if state != ProbeRefused || !strings.Contains(detail, "FORGEJO_URL") {
		t.Fatalf("got %s %q, want refused with the reason", state, detail)
	}
}

func TestCheckCommandFailing(t *testing.T) {
	repo := probeRepo(t, "forge.sh", `echo 'no token' >&2; exit 3`)
	state, detail := CheckCommand(repo, "./forge.sh")
	if state != ProbeFailing || detail != "no token" {
		t.Fatalf("got %s %q, want failing: no token", state, detail)
	}
	repo = probeRepo(t, "forge.sh", `cat >/dev/null; echo not-json`)
	if state, _ := CheckCommand(repo, "./forge.sh"); state != ProbeFailing {
		t.Fatalf("an answer with no JSON is failing, got %s", state)
	}
}

// A command that never answers is failing once the deadline passes.
func TestCheckCommandTimeoutIsFailing(t *testing.T) {
	old := probeTimeout
	probeTimeout = 200 * time.Millisecond
	t.Cleanup(func() { probeTimeout = old })
	repo := probeRepo(t, "forge.sh", `exec sleep 30`)
	state, detail := CheckCommand(repo, "./forge.sh")
	if state != ProbeFailing || !strings.Contains(detail, "did not answer within") {
		t.Fatalf("got %s %q, want failing: did not answer within", state, detail)
	}
}

// A shell expression as first word cannot be looked up: the shell's own
// 127 says it is missing.
func TestCheckCommandShellExpressionNotFoundIsMissing(t *testing.T) {
	repo := t.TempDir()
	state, _ := CheckCommand(repo, "FOO=1 workline-no-such-forge")
	if state != ProbeMissing {
		t.Fatalf("got %s, want missing", state)
	}
}

// A command found that exits 126 or 127 for its own reasons has run: it is
// failing, not missing.
func TestCheckCommandOwnExit127IsFailing(t *testing.T) {
	repo := probeRepo(t, "forge.sh", `echo 'curl: not found' >&2; exit 127`)
	state, detail := CheckCommand(repo, "./forge.sh")
	if state != ProbeFailing || !strings.Contains(detail, "curl") {
		t.Fatalf("got %s %q, want failing", state, detail)
	}
}
