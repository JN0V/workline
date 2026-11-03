package hooks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The wrapper is gone:
// Given the hooks package,
// When its code is searched for the old local wrapper's name
// (the word "range" followed by "Head", written in pieces here),
// Then nothing is found.
func TestTheWrapperIsGone(t *testing.T) {
	name := "range" + "Head"
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(src), "\n") {
			if strings.Contains(line, name) {
				t.Errorf("%s:%d still names %s: %s", f, i+1, name, strings.TrimSpace(line))
			}
		}
	}
}
