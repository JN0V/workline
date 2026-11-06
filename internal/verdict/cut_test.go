package verdict

import (
	"os"
	"path/filepath"
	"testing"
)

// A cut pre says is a warning of the run, once however often it is said,
// and not again when the run already holds it (#1159).
func TestCutsSaidOnce(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "in"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := CutFindings(dir, nil); len(got) != 0 {
		t.Fatalf("no cut said: %v", got)
	}
	for _, c := range []string{"the code cut", "the code  cut", "the tests cut"} {
		if err := SayCut(dir, c); err != nil {
			t.Fatal(err)
		}
	}
	got := CutFindings(dir, nil)
	if len(got) != 2 || got[0].Rule != CutRule || got[0].Level != "warn" || got[0].Blocks() || got[1].Message != "the tests cut" {
		t.Fatalf("cuts: %+v", got)
	}
	if again := CutFindings(dir, got); len(again) != 0 {
		t.Fatalf("said again: %+v", again)
	}
}
