package line

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/routing"
)

// drafting is a backlog that counts what the WIP count reads on it.
type drafting struct {
	counted
	branches []string
}

func (d *drafting) OpenMergeRequests(string) ([]string, error) {
	d.calls["drafts"]++
	return d.branches, nil
}

func lineConfig(t *testing.T, config string) *routing.Config {
	t.Helper()
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".workline"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".workline", "config.yaml"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := routing.Load(repo)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

// TestWaitsCountsWhatCanBePulled: Ready's limit counts the issues the
// developer can take (#854). Below the limit, nothing more is read; at
// it, the tester's Drafts once and the comments of an issue with no
// Draft, each once a run; an issue waiting for its tests, or on another
// issue, is counted apart, and said so.
func TestWaitsCountsWhatCanBePulled(t *testing.T) {
	cfg := lineConfig(t, "routing: {events: {issue: [tester, developer]}, wip: {ready: 2}}\n")
	tests, _ := cfg.State("tests")
	scenario := "## Need\n\nx\n\n## Validation\n\n- **One:** Given a, When b, Then c.\n"
	b := &drafting{counted: counted{calls: map[string]int{}}, branches: []string{"workline/tester/issue-1"}}
	open := []forge.Issue{
		{ID: 1, Labels: []string{"workline:ready"}, Body: scenario},                       // its Draft open: counted
		{ID: 2, Labels: []string{"workline:ready"}, Body: scenario},                       // no Draft, no proof: apart
		{ID: 3, Labels: []string{"workline:ready"}, Body: "## Need\n\nx\n\nBlocked by #6\n"}, // waits on #6: apart
		{ID: 6, Labels: []string{"workline:needs-tests"}, Body: scenario},
	}
	tl := newTally()
	why, err := waits(cfg, b, tests, &open, tl)
	if err != nil || why != "" {
		t.Fatalf("Ready holds 1 the developer can take, of 2: waits %q (%v)", why, err)
	}
	if b.calls["drafts"] != 1 || b.calls["comments"] != 1 || b.calls["issues"] != 0 {
		t.Errorf("reads at the limit: %v, want the Drafts once and #2's comments once", b.calls)
	}
	if _, err := waits(cfg, b, tests, &open, tl); err != nil || b.calls["drafts"] != 1 || b.calls["comments"] != 1 {
		t.Errorf("a second count in the run reads again: %v (%v)", b.calls, err)
	}

	open[1].Body = "## Need\n\nx\n" // no scenario: the developer takes it as it is
	why, err = waits(cfg, b, tests, &open, newTally())
	if err != nil || !strings.Contains(why, "ready holds 2 open issues the developer can take, its limit 2 (routing.wip), and 1 waiting for its tests or another issue, not counted") {
		t.Errorf("at the limit: %q (%v)", why, err)
	}

	b.calls = map[string]int{}
	few := open[2:]
	if why, err := waits(cfg, b, tests, &few, newTally()); err != nil || why != "" || len(b.calls) != 0 {
		t.Errorf("below the limit: %q, reads %v (%v), want none", why, b.calls, err)
	}
}

// TestWaitsTesterNotRouted: where the project routes no tester, the
// developer waits for no tests: every ready issue counts.
func TestWaitsTesterNotRouted(t *testing.T) {
	cfg := lineConfig(t, "routing: {events: {issue: [product-owner, developer]}, wip: {ready: 1}}\n")
	tests, _ := cfg.State("tests")
	b := &drafting{counted: counted{calls: map[string]int{}}}
	open := []forge.Issue{{ID: 1, Labels: []string{"workline:ready"}, Body: "## Need\n\nx\n\n## Validation\n\n- **One:** Given a, When b, Then c.\n"}}
	if why, err := waits(cfg, b, tests, &open, newTally()); err != nil || !strings.HasPrefix(why, "ready holds 1 open issue, its limit 1") {
		t.Errorf("no tester routed: %q (%v)", why, err)
	}
	if len(b.calls) != 0 {
		t.Errorf("reads with no tester routed: %v", b.calls)
	}
}
