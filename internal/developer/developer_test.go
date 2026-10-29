package developer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/JN0V/workline/internal/verdict"
)

// TestPreBlocksWithoutMergeRequest: a line that opens no merge request
// takes nothing, the option absent or set empty.
func TestPreBlocksWithoutMergeRequest(t *testing.T) {
	for name, set := range map[string]bool{"absent": false, "empty": true} {
		t.Run(name, func(t *testing.T) {
			runDir := t.TempDir()
			for _, d := range []string{"in", "out"} {
				if err := os.MkdirAll(filepath.Join(runDir, d), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(filepath.Join(runDir, "in", "settings.json"), []byte("{}"), 0o644); err != nil {
				t.Fatal(err)
			}
			t.Setenv("WORKLINE_OPEN_MERGE_REQUESTS", "")
			if !set {
				if err := os.Unsetenv("WORKLINE_OPEN_MERGE_REQUESTS"); err != nil {
					t.Fatal(err)
				}
			}
			if code := Pre(runDir, t.TempDir()); code != exitNothing {
				t.Fatalf("exit %d, want %d", code, exitNothing)
			}
			v, err := verdict.Read(filepath.Join(runDir, "out", "verdict.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			if v.Status != verdict.Block || len(v.Findings) != 1 || v.Findings[0].Rule != "no-merge-request" {
				t.Fatalf("verdict %+v, want a block on no-merge-request", v)
			}
		})
	}
}
