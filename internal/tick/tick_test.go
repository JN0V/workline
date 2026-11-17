package tick

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/routing"
)

var now = time.Date(2026, 11, 16, 2, 20, 0, 0, time.UTC)

// setUp writes the project's config and a fake forge holding ready
// issues and runs; it returns the options a tick reads.
func setUp(t *testing.T, config string, ready int, runs []forge.Run, unread string) Options {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".workline"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".workline", "config.yaml"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	var s forge.FakeState
	for i := range ready {
		s.Issues = append(s.Issues, forge.FakeItem{ID: i + 1, Title: "x", Labels: []string{"workline:ready"}})
	}
	s.Runs, s.RunsUnread = runs, unread
	data, _ := json.Marshal(s)
	path := filepath.Join(dir, "forge.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := routing.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	return Options{Repo: dir, Config: cfg, Forge: &forge.Fake{Path: path, Repo: dir}, ForgeSpec: "fake:" + path, Now: now}
}

func run(name, wf, status, started string) forge.Run {
	return forge.Run{Name: name, Workflow: wf, Status: status, Started: started, Updated: started}
}

// done: every shipped timed job ran since its time.
var done = []forge.Run{
	run("workline merge", "workline-merge.yml", "completed", "2026-11-16T02:00:00Z"),
	run("workline gardening docs", "workline-gardening.yml", "completed", "2026-11-15T18:10:00Z"),
	run("workline sample", "workline-sample.yml", "completed", "2026-11-16T00:00:00Z"),
}

func jobsOf(r Result) []string {
	var names []string
	for _, it := range r.Items {
		names = append(names, it.Job)
	}
	return names
}

// TestDue: what a tick lists, case by case (ADR-0069 §2, #1665).
func TestDue(t *testing.T) {
	po := "routing: {events: {schedule: [product-owner], issue: [tester, developer]}, wip: {ready: 10}%s}\n"
	nightly := ", timed: {nightly: {at: \"17 2 * * *\"}}"
	for _, c := range []struct {
		name, config string
		ready        int
		runs         []forge.Run
		unread       string
		want         []string
		said         string
	}{
		{"nothing due", po, 10, done, "", []string{"reconcile"}, ""},
		{"nothing at all", strings.Replace(po, "%s", ", timed: {reconcile: off}%s", 1), 10, done, "", nil, ""},
		{"backlog due", po, 6, done, "", []string{"reconcile", "backlog"}, ""},
		{"backlog going", po, 6, append(slices.Clone(done), run("workline gardening backlog", "workline-gardening.yml", "queued", "2026-11-16T02:18:00Z")),
			"", []string{"reconcile"}, `backlog: "workline gardening backlog" queued`},
		{"backlog ran, readied none, nothing left Ready", po, 6, append(slices.Clone(done), run("workline gardening", "workline-gardening.yml", "completed", "2026-11-16T01:47:00Z")),
			"", []string{"reconcile"}, "readied none"},
		{"reconcile going", po, 10, append(slices.Clone(done), run("workline develop heartbeat", "workline-develop.yml", "waiting", "2026-11-16T02:16:00Z")),
			"", nil, `reconcile: "workline develop heartbeat" waiting`},
		{"nightly due", po, 10, append(slices.Clone(done), run("workline gardening nightly", "workline-gardening.yml", "completed", "2026-11-15T02:19:00Z")),
			"", []string{"reconcile", "nightly"}, ""},
		{"nightly done", po, 10, append(slices.Clone(done), run("workline gardening nightly", "workline-gardening.yml", "completed", "2026-11-16T02:17:00Z")),
			"", []string{"reconcile"}, "nightly: ran at 2026-11-16T02:17:00Z"},
		{"merge's gap passed", po, 10, []forge.Run{done[1], done[2], run("workline merge", "workline-merge.yml", "completed", "2026-11-16T01:19:00Z")},
			"", []string{"reconcile", "merge"}, ""},
		{"merge within its gap", po, 10, []forge.Run{done[1], done[2], run("workline merge", "workline-merge.yml", "completed", "2026-11-16T01:50:00Z")},
			"", []string{"reconcile"}, ""},
		{"a gap of its own on the reconcile", strings.Replace(po, "%s", ", timed: {reconcile: {gap: 15}}%s", 1), 10,
			append(slices.Clone(done), run("workline develop reconcile", "workline-develop.yml", "completed", "2026-11-16T02:10:00Z")), "", nil, "reconcile: ran at"},
		{"a time of its own on the reconcile, past", strings.Replace(po, "%s", ", timed: {reconcile: {at: \"0 2 * * *\"}}%s", 1), 10,
			append(slices.Clone(done), run("workline develop reconcile", "workline-develop.yml", "completed", "2026-11-16T02:05:00Z")), "", nil, "reconcile: ran at"},
		{"runs not read", po, 6, nil, "HTTP 502", []string{"reconcile"}, ""},
	} {
		config := c.config
		if c.name == "nightly due" || c.name == "nightly done" {
			config = strings.Replace(config, "%s", nightly+"%s", 1)
		}
		o := setUp(t, strings.Replace(config, "%s", "", 1), c.ready, c.runs, c.unread)
		r := Due(o)
		if got := jobsOf(r); !slices.Equal(got, c.want) {
			t.Errorf("%s: jobs %v, want %v; said %v, unread %v", c.name, got, c.want, r.Said, r.Unread)
		}
		if c.said != "" && !slices.ContainsFunc(r.Said, func(s string) bool { return strings.Contains(s, c.said) }) {
			t.Errorf("%s: said %v, want %q", c.name, r.Said, c.said)
		}
		if c.unread != "" && !slices.ContainsFunc(r.Unread, func(s string) bool { return strings.Contains(s, c.unread) }) {
			t.Errorf("%s: unread %v, want %q", c.name, r.Unread, c.unread)
		}
	}
}

// TestDueNoForge: with no forge, the reconcile alone, and why the rest is
// not listed.
func TestDueNoForge(t *testing.T) {
	o := setUp(t, "routing: {events: {schedule: [product-owner]}, wip: {ready: 10}}\n", 0, nil, "")
	o.Forge = nil
	r := Due(o)
	if got := jobsOf(r); !slices.Equal(got, []string{"reconcile"}) {
		t.Errorf("jobs %v, want the reconcile alone", got)
	}
	if !slices.ContainsFunc(r.Unread, func(s string) bool { return strings.Contains(s, "lists no CI runs") }) {
		t.Errorf("unread %v: want the runs said not listed", r.Unread)
	}
}

// TestKeysStable: two ticks on the same state list the same keys.
func TestKeysStable(t *testing.T) {
	o := setUp(t, "routing: {events: {schedule: [product-owner]}, wip: {ready: 10}}\n", 3, nil, "")
	a, b := Due(o), Due(o)
	if len(a.Items) == 0 || !slices.Equal(a.Items, b.Items) {
		t.Errorf("first %v, then %v", a.Items, b.Items)
	}
}
