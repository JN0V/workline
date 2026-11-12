package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// Past keepRuns, the runs dropped leave what they spent in the tally: the
// whole line's fresh tokens are still summed (#1360).
func TestPrunedRunsKeepTheirTokens(t *testing.T) {
	base := t.TempDir()
	t.Setenv(forge.RunsDirEnv, base)
	for i := range keepRuns + 5 {
		dir := filepath.Join(base, fmt.Sprintf("20260105T0900%02d.000000000-reviewer", i))
		if err := os.MkdirAll(filepath.Join(dir, "out"), 0o755); err != nil {
			t.Fatal(err)
		}
		_ = os.WriteFile(filepath.Join(dir, "out", "run.yaml"), []byte("role: reviewer\ntarget: {kind: merge-request, id: 7}\n"), 0o644)
		_ = os.WriteFile(filepath.Join(dir, "out", "calls.jsonl"), []byte(`{"agent":"claude","tokens-in":1500,"tokens-cached":500,"tokens-out":100,"seconds":2}`+"\n"), 0o644)
	}
	quiet := filepath.Join(base, "20260105T085959.000000000-committer") // no agent called: nothing to keep
	_ = os.MkdirAll(filepath.Join(quiet, "out"), 0o755)
	if _, err := newRunDir(t.TempDir(), "product-owner"); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(base)
	dirs := 0
	for _, e := range entries {
		if e.IsDir() {
			dirs++
		}
	}
	if dirs != keepRuns {
		t.Fatalf("%d run folders kept, want %d", dirs, keepRuns)
	}
	runs, err := Tallies(base)
	if err != nil {
		t.Fatal(err)
	}
	fresh, calls := 0, 0
	for _, r := range runs {
		fresh += r.Fresh
		calls += r.Calls
		if r.Calls > 0 && (r.Target == nil || r.Target.ID != 7 || r.Role != "reviewer" || r.Started().IsZero()) {
			t.Errorf("run %s kept as %+v", r.Run, r)
		}
	}
	if want := (keepRuns + 5) * 1100; fresh != want || calls != keepRuns+5 {
		t.Errorf("fresh %d over %d calls, want %d over %d", fresh, calls, want, keepRuns+5)
	}
	// Pruned again, nothing is counted twice.
	if _, err := newRunDir(t.TempDir(), "product-owner"); err != nil {
		t.Fatal(err)
	}
	runs, _ = Tallies(base)
	again := 0
	for _, r := range runs {
		again += r.Fresh
	}
	if again != fresh {
		t.Errorf("fresh %d after a second prune, want %d", again, fresh)
	}
}

// A run whose calls cannot all be read is never dropped: its tokens
// would be lost with it.
func TestDamagedCallsKeepTheirRun(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "20260105T090000.000000000-reviewer")
	if err := os.MkdirAll(filepath.Join(dir, "out"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "out", "calls.jsonl"), []byte("{\"tokens-in\":10}\n{\"tokens-in\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := keepTally(base, dir); err == nil {
		t.Fatal("a damaged calls file kept nothing and said nothing")
	}
	if _, err := os.Stat(filepath.Join(base, TallyFile)); err == nil {
		t.Error("a damaged run was put in the tally")
	}
}
