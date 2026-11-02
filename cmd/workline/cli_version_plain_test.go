package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

func captureStderr(t *testing.T, f func() int) (string, int) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	read := make(chan []byte)
	go func() { out, _ := io.ReadAll(r); read <- out }()
	old := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = old }()
	code := f()
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return string(<-read), code
}

// The release check (docs/release-holds.md) reads one line from the plain form.
func TestVersionPlainOneLine(t *testing.T) {
	out, code := captureStdout(t, func() int { return dispatch([]string{"version"}) })
	if code != 0 {
		t.Fatalf("workline version: exit %d", code)
	}
	if !strings.HasSuffix(out, "\n") || strings.Count(out, "\n") != 1 {
		t.Fatalf("want exactly one line, got %q", out)
	}
	if strings.TrimSuffix(out, "\n") != engineVersion() {
		t.Errorf("line = %q, want the version %q", out, engineVersion())
	}
}

func TestVersionVerboseFalse(t *testing.T) {
	plain, _ := captureStdout(t, func() int { return dispatch([]string{"version"}) })
	out, code := captureStdout(t, func() int { return dispatch([]string{"version", "--verbose=false"}) })
	if code != 0 {
		t.Fatalf("workline version --verbose=false: exit %d", code)
	}
	if out != plain || strings.Count(out, "\n") != 1 {
		t.Errorf("got %q, want the plain form %q", out, plain)
	}
}

func TestVersionRefusesArgument(t *testing.T) {
	for _, args := range [][]string{{"version", "extra"}, {"version", "--verbose", "extra"}} {
		var code int
		errOut, _ := captureStderr(t, func() int {
			code = dispatch(args)
			return code
		})
		if code != 64 {
			t.Errorf("workline %v: exit %d, want 64", args, code)
		}
		if !strings.Contains(errOut, "takes no argument") {
			t.Errorf("workline %v: stderr %q lacks \"takes no argument\"", args, errOut)
		}
	}
}
