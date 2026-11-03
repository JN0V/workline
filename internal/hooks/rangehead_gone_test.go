package hooks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The wrapper is gone:
// Given the hooks package,
// When its code is searched for `rangeHead`,
// Then nothing is found.
func TestTheWrapperIsGone(t *testing.T) {
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
			if strings.Contains(line, "rangeHead") {
				t.Errorf("%s:%d still names rangeHead: %s", f, i+1, strings.TrimSpace(line))
			}
		}
	}
}
