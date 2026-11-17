package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/JN0V/workline/internal/developer"
)

// The apply finds the bundle the merge round writes, where it writes it;
// a run with no bundle (an older engine's) applies as before (#1704).
func TestOwnMergeFindsTheRoundsBundle(t *testing.T) {
	run := t.TempDir()
	if err := os.MkdirAll(filepath.Join(run, "out"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(run, "out", "merge-request.yaml"), []byte("merge: abc\ntree: def\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if m := ownMerge(run); m == nil || m.Bundle != "" {
		t.Fatalf("no bundle: want none named, got %+v", m)
	}
	b := filepath.Join(run, developer.MergeBundle)
	if err := os.WriteFile(b, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if m := ownMerge(run); m == nil || m.Bundle != b {
		t.Fatalf("want the bundle %s, got %+v", b, m)
	}
}
