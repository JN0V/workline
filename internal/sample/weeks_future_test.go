package sample

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/JN0V/workline/internal/verdict"
)

// Future week
//
// Given today is in 2026-W45,
// When I run `workline sample --week 2030-W01`,
// Then it blocks, says the week has not started, and exits 1.
func TestFutureWeek(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	repo := t.TempDir()
	if out, err := exec.Command("git", "-C", repo, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	now := time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC) // 2026-W45
	for _, week := range []string{"2030-W01", "2026-W46"} {
		res := Draw(Options{Repo: repo, Week: week, Now: now})
		if res.Status != verdict.Block {
			t.Errorf("--week %s: status %q, want %q", week, res.Status, verdict.Block)
		}
		if !strings.Contains(res.Summary, "has not started") {
			t.Errorf("--week %s: summary %q does not say the week has not started", week, res.Summary)
		}
	}
}
