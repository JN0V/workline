package sample

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// Past week: with today in 2026-W45, the past week and the current one
// are not refused as weeks to come.
func TestPastAndCurrentWeekNotFuture(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	repo := t.TempDir()
	if out, err := exec.Command("git", "-C", repo, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	for _, now := range []time.Time{
		time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC),  // 2026-W45, a Wednesday
		time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC),   // the Monday the week starts
		time.Date(2026, 11, 8, 23, 59, 0, 0, time.UTC), // the Sunday it ends
	} {
		for _, week := range []string{"2026-W44", "2026-W45"} {
			res := Draw(Options{Repo: repo, Week: week, Now: now})
			if strings.Contains(res.Summary, "has not started") {
				t.Errorf("--week %s on %s: refused as a week to come: %q", week, now.Format(time.DateOnly), res.Summary)
			}
		}
	}
}
