package clock

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// The packages the line simulator runs on its own clock (ADR-0063, step
// 1): a time read there that skips Now would run on the machine's time,
// and a trace would no longer replay.
var seamed = []string{"../line", "../builtin/productowner", "../backlog"}

var timeRead = regexp.MustCompile(`\btime\.(Now|Since|Until)\b`)

func TestFixedSetsNow(t *testing.T) {
	t.Setenv(Fixed, "2026-11-04T20:00:00Z")
	if got := Now(); !got.Equal(time.Date(2026, 11, 4, 20, 0, 0, 0, time.UTC)) {
		t.Errorf("Now with %s set: %v", Fixed, got)
	}
	if got := Since(time.Date(2026, 11, 4, 19, 0, 0, 0, time.UTC)); got != time.Hour {
		t.Errorf("Since: %v, want 1h", got)
	}
	t.Setenv(Fixed, "tomorrow")
	if got := time.Since(Now()); got < 0 || got > time.Minute {
		t.Errorf("Now with %s not a time: %v, want the machine's", Fixed, got)
	}
}

func TestEveryTimeReadGoesThroughTheSeam(t *testing.T) {
	for _, dir := range seamed {
		files, err := filepath.Glob(filepath.Join(dir, "*.go"))
		if err != nil || len(files) == 0 {
			t.Fatalf("%s: no Go file read (%v)", dir, err)
		}
		for _, f := range files {
			if strings.HasSuffix(f, "_test.go") {
				continue
			}
			data, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			for i, line := range strings.Split(string(data), "\n") {
				// A real wait or a unique name says so: "// real time".
				if timeRead.MatchString(line) && !strings.Contains(line, "// real time") {
					t.Errorf("%s:%d reads the time past the seam, use clock.Now: %s", f, i+1, strings.TrimSpace(line))
				}
			}
		}
	}
}
