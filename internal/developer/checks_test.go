package developer

import (
	"fmt"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

func TestLogEndBounded(t *testing.T) {
	var b strings.Builder
	b.WriteString("\ufeff2026-11-02T22:28:54.3390482Z ##[group]Run go test ./...\n")
	for i := range 200 {
		fmt.Fprintf(&b, "2026-11-02T22:29:16.0557039Z ok  \tpkg%d\t0.016s\n", i)
	}
	b.WriteString("2026-11-02T22:34:56.3602416Z \x1b[31m--- FAIL: TestConformance (326.42s)\x1b[0m\n")
	b.WriteString("2026-11-02T22:34:56.4395746Z ##[error]Process completed with exit code 1.\n\n")
	got := logEnd(b.String())
	lines := strings.Split(got, "\n")
	if lines[0] != "…" || len(lines) != logLines+1 {
		t.Fatalf("want … then %d lines, got %d:\n%s", logLines, len(lines), got)
	}
	if !strings.HasSuffix(got, "--- FAIL: TestConformance (326.42s)\n##[error]Process completed with exit code 1.") {
		t.Errorf("the end lost, or stamps and colours kept:\n%s", got[len(got)-200:])
	}
	if strings.Contains(got, "##[group]") || strings.Contains(got, "2026-11-02T") {
		t.Errorf("stamps or group marks kept")
	}
	long := logEnd(strings.Repeat(strings.Repeat("x", 199)+"\n", 60))
	if len(long) > logBytes+4 || !strings.HasPrefix(long, "…\n") {
		t.Errorf("not bounded to %d bytes: %d", logBytes, len(long))
	}
}

func TestOwnCheck(t *testing.T) {
	for _, c := range []struct {
		check forge.HeadCheck
		own   bool
	}{
		{forge.HeadCheck{Name: "test", Workflow: "test"}, false},
		{forge.HeadCheck{Name: "build", Workflow: "CI"}, false},
		{forge.HeadCheck{Name: "judge", Workflow: "workline"}, true},
		{forge.HeadCheck{Name: "bounds-apply", Workflow: "workline bounds"}, true},
		{forge.HeadCheck{Name: "workline findings"}, true},
		{forge.HeadCheck{Name: "workline/verdict"}, true},
	} {
		if got := forge.OwnCheck(c.check); got != c.own {
			t.Errorf("%+v: own %v, want %v", c.check, got, c.own)
		}
	}
}
