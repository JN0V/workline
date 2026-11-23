//go:build linux

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"unsafe"
)

// docsPassResult checks that stdout is the JSON result of a pass.
func docsPassResult(t *testing.T, stdout, stderr string) {
	t.Helper()
	var res struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(stdout), &res); err != nil {
		t.Fatalf("stdout is not the JSON result: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if res.Status != "pass" {
		t.Errorf("status = %q, want pass\n%s", res.Status, stdout)
	}
}

// docsOnTerminal runs the built binary in dir with a pseudo-terminal as its
// controlling terminal (what /dev/tty is for it), the lines in answers typed
// on it; it returns stdout, stderr and the exit code.
func docsOnTerminal(t *testing.T, bin, dir, answers string, args ...string) (string, string, int) {
	t.Helper()
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		t.Skipf("no pseudo-terminal here: %v", err)
	}
	defer func() { _ = master.Close() }()
	ioctl := func(req uintptr, arg unsafe.Pointer) error {
		if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, master.Fd(), req, uintptr(arg)); e != 0 {
			return e
		}
		return nil
	}
	var unlock int32
	var n uint32
	if err := ioctl(syscall.TIOCSPTLCK, unsafe.Pointer(&unlock)); err != nil {
		t.Skipf("no pseudo-terminal here: %v", err)
	}
	if err := ioctl(syscall.TIOCGPTN, unsafe.Pointer(&n)); err != nil {
		t.Skipf("no pseudo-terminal here: %v", err)
	}
	slave, err := os.OpenFile(fmt.Sprintf("/dev/pts/%d", n), os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		t.Skipf("no pseudo-terminal here: %v", err)
	}
	defer func() { _ = slave.Close() }()

	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	cmd.Stdin = slave
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 0}
	var o, e bytes.Buffer
	cmd.Stdout, cmd.Stderr = &o, &e
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	_ = slave.Close()
	go func() { _, _ = io.Copy(io.Discard, master) }()
	if _, err := master.WriteString(answers); err != nil {
		t.Fatal(err)
	}
	code := 0
	var ee *exec.ExitError
	if err := cmd.Wait(); errors.As(err, &ee) {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return o.String(), e.String(), code
}

// No doc changed
// Given `--review --json` and no changed doc,
// When it runs,
// Then a pass result in JSON is printed and the exit is 0.
func TestDocsReviewJSONNoDocChanged(t *testing.T) {
	sinceEnv(t)
	bin := sinceBuild(t)
	repo := docsJSONRepo(t)

	stdout, stderr, code := sweepRun(t, bin, repo, "docs", "--repo", repo, "--review", "--json")
	if code != 0 {
		t.Errorf("exit code = %d, want 0\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	docsPassResult(t, stdout, stderr)
}

// No new commit
// Given no commit since the docs were last judged,
// When `workline docs --json` runs,
// Then a pass result in JSON is printed and the exit is 0.
func TestDocsJSONNoNewCommit(t *testing.T) {
	sinceEnv(t)
	bin := sinceBuild(t)
	repo := docsJSONRepo(t)
	gitIn(t, repo, "update-ref", docsJudgedRef, "HEAD")

	stdout, stderr, code := sweepRun(t, bin, repo, "docs", "--repo", repo, "--json", "--ai", "none")
	if code != 0 {
		t.Errorf("exit code = %d, want 0\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	docsPassResult(t, stdout, stderr)
}

// Review done
// Given a successful `--review --json`,
// When it ends,
// Then a pass result in JSON is printed.
func TestDocsReviewJSONDone(t *testing.T) {
	sinceEnv(t)
	bin := sinceBuild(t)
	repo := docsJSONRepo(t)
	if err := os.WriteFile(filepath.Join(repo, "docs", "a.md"), []byte("# A\n\nSome other words.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, code := docsOnTerminal(t, bin, repo, "y\n", "docs", "--repo", repo, "--review", "--json")
	if code != 0 {
		t.Errorf("exit code = %d, want 0\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	docsPassResult(t, stdout, stderr)
}
