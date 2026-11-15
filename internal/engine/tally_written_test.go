package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// Pruned run kept
// Given a job holding more than 50 runs,
// When it prunes the oldest,
// Then their tokens are still counted that week: the tally the job
// writes for the weekly sample holds the pruned runs as well as the
// folders still there.
func TestWrittenTallyKeepsPrunedRuns(t *testing.T) {
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
	if _, err := newRunDir(t.TempDir(), "product-owner"); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "workline-tally.jsonl")
	if _, err := WriteTally(base, file); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("no tally written: %v", err)
	}
	runs, fresh := 0, 0
	for _, l := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var r RunTally
		if err := json.Unmarshal([]byte(l), &r); err != nil {
			t.Fatalf("tally line %q: %v", l, err)
		}
		runs++
		fresh += r.Fresh
	}
	if want := keepRuns + 5; runs != want || fresh != want*1100 {
		t.Errorf("%d runs, %d fresh tokens in the written tally, want %d runs, %d", runs, fresh, want, want*1100)
	}
}
