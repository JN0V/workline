package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

// testHelp runs `workline test --help` and returns what it prints on stdout.
func testHelp(t *testing.T) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	code := dispatch([]string{"test", "--help"})
	os.Stdout = old
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if code != 0 {
		t.Fatalf("workline test --help exited %d, want 0", code)
	}
	return string(out)
}

// Usage names five
//
// Given workline built from `main`,
// When I run `workline test --help`,
// Then its usage line names `check`, `write`, `answer`, `measure` and `curate`.
func TestUsageNamesFive(t *testing.T) {
	out := testHelp(t)
	usage := ""
	for _, l := range strings.Split(out, "\n") {
		if strings.HasPrefix(l, "usage:") {
			usage = l
			break
		}
	}
	if usage == "" {
		t.Fatalf("no usage line in:\n%s", out)
	}
	for _, name := range []string{"check", "write", "answer", "measure", "curate"} {
		if !strings.Contains(usage, name) {
			t.Errorf("usage line %q does not name %q", usage, name)
		}
	}
}

// About says what each does
//
// Given the same,
// When I read the line under the usage,
// Then it says in a few words what `measure` and `curate` do, as it does for the other three.
func TestAboutSaysWhatEachDoes(t *testing.T) {
	out := testHelp(t)
	_, rest, ok := strings.Cut(out, "usage:")
	if !ok {
		t.Fatalf("no usage line in:\n%s", out)
	}
	_, rest, _ = strings.Cut(rest, "\n")
	about, _, _ := strings.Cut(rest, "\nOptions:")
	for _, name := range []string{"check", "write", "answer", "measure", "curate"} {
		if !strings.Contains(about, name) {
			t.Errorf("the about %q does not say what %q does", strings.TrimSpace(about), name)
		}
	}
}
