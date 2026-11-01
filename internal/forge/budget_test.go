package forge

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// GitHub's budget is read from rate_limit, which counts against no limit:
// of the REST and GraphQL budgets, the one with the least share left.
func TestGitHubBudget(t *testing.T) {
	bin := t.TempDir()
	script := "#!/bin/sh\n" + `cat <<'EOF'
{"resource":"core","remaining":90,"limit":1000,"reset":1793539440}
{"resource":"graphql","remaining":300,"limit":1000,"reset":1793537866}
EOF
`
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	b, err := (&github{repo: t.TempDir()}).Budget()
	if err != nil {
		t.Fatal(err)
	}
	if b.Resource != "core" || b.Remaining != 90 || b.Limit != 1000 || !b.Reset.Equal(time.Unix(1793539440, 0)) {
		t.Errorf("budget %+v, want core's", b)
	}
	if !b.Low() {
		t.Errorf("90 of 1000 left: low")
	}
	if (Budget{Remaining: 100, Limit: 1000}).Low() || (Budget{}).Low() {
		t.Errorf("a tenth left, or no limit said: not low")
	}
}

// A call refused for the rate, its reset half an hour away, is not
// waited for on a person's push; in a run of the line (RateWaitEnv), it
// is, up to the tries' bound (#804).
func TestRateWaitInALineRun(t *testing.T) {
	f := failure{code: 403, limited: true, header: http.Header{"Retry-After": {"1800"}}}
	if wait, note := again("GitHub", true, 1, f); wait >= 0 || !strings.Contains(note, "longer than workline waits") {
		t.Errorf("no line run: waits %s, %q; want not tried again, said", wait, note)
	}
	t.Setenv(RateWaitEnv, "61m")
	if wait, note := again("GitHub", true, 1, f); wait != 30*time.Minute || note != "" {
		t.Errorf("a line run: waits %s, %q; want 30 min", wait, note)
	}
	if wait, _ := again("GitHub", true, retry.tries, f); wait >= 0 {
		t.Errorf("the last try: waits %s; want not tried again", wait)
	}
	f.header = http.Header{"Retry-After": {"7200"}}
	if wait, _ := again("GitHub", true, 1, f); wait >= 0 {
		t.Errorf("a reset past the line's wait: waits %s; want not tried again", wait)
	}
}
