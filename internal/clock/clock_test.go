package clock

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The packages the line simulator runs on its own clock (ADR-0063, step
// 1): a time read there that skips Now would run on the machine's time,
// and a trace would no longer replay.
var seamed = []string{"../line", "../builtin/productowner", "../backlog"}

var timeRead = regexp.MustCompile(`\btime\.(Now|Since|Until)\b`)

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
				if timeRead.MatchString(line) {
					t.Errorf("%s:%d reads the time past the seam, use clock.Now: %s", f, i+1, strings.TrimSpace(line))
				}
			}
		}
	}
}
