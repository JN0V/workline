package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// Broken config:
// Given a repository whose `.workline/config.yaml` is not valid YAML,
// When I run `workline docs`,
// Then it prints the YAML error and exits 1.
func TestBrokenConfig(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)

	tmp := t.TempDir()
	bin := filepath.Join(tmp, "workline")
	build := exec.Command("go", "build", "-o", bin, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}

	repo := filepath.Join(tmp, "repo")
	if err := os.MkdirAll(filepath.Join(repo, ".workline"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".workline", "config.yaml"), []byte("foo: [unclosed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "a.txt"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git := func(args ...string) {
		t.Helper()
		c := exec.Command("git", append([]string{"-C", repo}, args...)...)
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q")
	git("add", "-A")
	git("-c", "user.name=T", "-c", "user.email=t@example.com", "commit", "-q", "-m", "init")

	cmd := exec.Command(bin, "docs", "--repo", repo, "--ai", "none")
	cmd.Dir = repo
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true} // no /dev/tty
	var buf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &buf, &buf
	err := cmd.Run()
	code := 0
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	if code != 1 {
		t.Errorf("exit code = %d, want 1\n%s", code, out)
	}
	if !strings.Contains(out, "config-invalid: .workline/config.yaml: yaml:") || !strings.Contains(out, "did not find expected") {
		t.Errorf("output lacks the YAML error:\n%s", out)
	}
	if strings.Contains(out, "block — ;") {
		t.Errorf("output has a block with an empty reason:\n%s", out)
	}
	if strings.Contains(out, "block — no doc changed") {
		t.Errorf("output has `block — no doc changed`:\n%s", out)
	}
}
