package routing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestCronLast: a cron's last time at or before now, in UTC.
func TestCronLast(t *testing.T) {
	now := time.Date(2026, 11, 16, 2, 20, 30, 0, time.UTC) // a Monday
	for _, c := range []struct{ cron, want string }{
		{"17 2 * * *", "2026-11-16T02:17:00Z"},
		{"20 2 * * *", "2026-11-16T02:20:00Z"},
		{"21 2 * * *", "2026-11-15T02:21:00Z"},
		{"7 18 * * 0", "2026-11-15T18:07:00Z"},
		{"43 5 * * 1", "2026-11-09T05:43:00Z"},
		{"*/5 * * * *", "2026-11-16T02:20:00Z"},
		{"3-59/10 * * * *", "2026-11-16T02:13:00Z"},
		{"0 0 1 * *", "2026-11-01T00:00:00Z"},
		{"0 12 * * 7", "2026-11-15T12:00:00Z"},
	} {
		cr, err := ParseCron(c.cron)
		if err != nil {
			t.Fatalf("%s: %v", c.cron, err)
		}
		got, ok := cr.Last(now)
		if !ok || got.Format(time.RFC3339) != c.want {
			t.Errorf("%s: Last = %v %v, want %s", c.cron, got, ok, c.want)
		}
	}
	for _, bad := range []string{"", "* * * *", "60 * * * *", "*/0 * * * *", "a * * * *", "5-1 * * * *", "@daily"} {
		if _, err := ParseCron(bad); err == nil {
			t.Errorf("%q: read, want refused", bad)
		}
	}
}

// TestTimed: a job's time, the project's over the shipped one; refused
// when it names no job, sets both at and gap, or gives the backlog a time.
func TestTimed(t *testing.T) {
	load := func(config string) (*Config, error) {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, ".workline"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, ".workline", "config.yaml"), []byte(config), 0o644); err != nil {
			t.Fatal(err)
		}
		return Load(dir)
	}
	c, err := load("routing: {timed: {nightly: {at: \"17 2 * * *\"}, merge: {gap: 15}, docs: off}}\n")
	if err != nil {
		t.Fatal(err)
	}
	if got := c.Timing("nightly"); got.At != "17 2 * * *" || got.Off {
		t.Errorf("nightly = %+v", got)
	}
	if got := c.Timing("merge"); got.Gap != 15 {
		t.Errorf("merge = %+v", got)
	}
	if !c.Timing("docs").Off || c.Timing("sample").At != "43 5 * * 1" || !c.Timing("seeds").Off || c.Timing("reconcile").Off {
		t.Errorf("docs %+v, sample %+v, seeds %+v, reconcile %+v", c.Timing("docs"), c.Timing("sample"), c.Timing("seeds"), c.Timing("reconcile"))
	}
	for config, want := range map[string]string{
		"routing: {timed: {nightly: {at: \"17 2 * * *\", gap: 60}}}\n": "routing.timed.nightly sets both at and gap",
		"routing: {timed: {weekly: {gap: 60}}}\n":                      "routing.timed.weekly: no such job",
		"routing: {timed: {backlog: {gap: 60}}}\n":                     "routing.timed.backlog: due when Ready is below its target",
		"routing: {timed: {nightly: {at: \"61 2 * * *\"}}}\n":          "routing.timed.nightly.at",
		"routing: {timed: {merge: {gap: -1}}}\n":                       "routing.timed.merge.gap is -1",
		"routing: {timed: {nightly: {}}}\n":                            "routing.timed.nightly sets neither at nor gap",
		"routing: {timed: {merge: sometimes}}\n":                       "a job's time is",
	} {
		if _, err := load(config); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: err = %v, want %q", config, err, want)
		}
	}
}
