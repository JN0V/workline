package gate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// A check's log, which a finding names and a role quotes, holds ***
// in place of a token's value (#1215).
func TestCheckLogMasked(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "fake-value-for-test-1215")
	dir, out := t.TempDir(), t.TempDir()
	code, f := execute(dir, out, Check{ID: "lint", Run: `echo "error: refused $GITLAB_TOKEN"; echo "error: x.go:1 unused"; exit 1`}, time.Minute, nil)
	if f != nil || code != 1 {
		t.Fatalf("execute = %d, %+v", code, f)
	}
	data, err := os.ReadFile(filepath.Join(out, "lint.log"))
	if err != nil {
		t.Fatal(err)
	}
	if got := string(data); got != "error: refused ***\nerror: x.go:1 unused\n" {
		t.Errorf("log = %q", got)
	}
	if strings.Contains(string(data), "fake-value") {
		t.Error("token in the log")
	}
}
