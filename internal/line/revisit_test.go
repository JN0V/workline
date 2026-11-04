package line

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	wlclock "github.com/JN0V/workline/internal/clock"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/routing"
)

// counted is a backlog that counts the calls a revisit makes on it.
type counted struct {
	forge.Backlog
	comments []string
	open     []forge.Issue
	mrs      []forge.MergeRequest
	calls    map[string]int
}

func (c *counted) ListMergeRequests() ([]forge.MergeRequest, error) {
	c.calls["merge-requests"]++
	return c.mrs, nil
}

func (c *counted) Comments(forge.Target) ([]string, error) {
	c.calls["comments"]++
	return c.comments, nil
}

func (c *counted) Issues() ([]forge.Issue, error) {
	c.calls["issues"]++
	return c.open, nil
}

func recordComment(t *testing.T, r Record) string {
	t.Helper()
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	return "**workline's line** blocks this issue.\n\n<!-- workline:" + recordMark + string(data) + " -->\n" + forge.Marker("sticky="+recordKey)
}

// TestRevisitCalls: what a revisit costs and when it wakes the product
// owner — one call, the comments, on the reconciler's listing; one more,
// the listing, on an issue read alone; no agent with nothing new.
func TestRevisitCalls(t *testing.T) {
	days := 7
	cfg := &routing.Config{Events: map[string][]string{IssueEvent: {"product-owner"}}, Revisit: &days}
	now := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
	defer func(c func() time.Time) { wlclock.Now = c }(wlclock.Now)
	wlclock.Now = func() time.Time { return now }
	body := "## Need\n\nI want to multiply.\n\nBlocked by #2."
	base := Record{State: BlockedState, Body: bodyHash(body), Hops: 9, From: "ready", Since: "2026-10-30T09:00:00Z"}
	is := forge.Issue{ID: 3, Body: body, Labels: []string{"workline:blocked"}}
	listing := []forge.Issue{{ID: 2}, is}
	for _, c := range []struct {
		name     string
		rec      Record
		extra    []string // comments beside the line's
		open     []forge.Issue
		listed   bool // the reconciler listed the open issues already
		wakes    bool
		calls    int
		trigger  string
		finding  string
		clockAdd time.Duration
	}{
		{"nothing new, listed: one call", base, nil, listing, true, false, 1, "", "state-blocked-quiet", 0},
		{"nothing new, read alone: the listing too", base, nil, listing, false, false, 2, "", "state-blocked-quiet", 0},
		{"a role's comment is no news", base, []string{"**The tester** wrote.\n\n<!-- workline:sticky=tester/issue -->"}, listing, true, false, 1, "", "state-blocked-quiet", 0},
		{"a person's comment", base, []string{"Fixed upstream."}, listing, true, true, 1, "a comment was written or edited on it", "state-revisit", 0},
		{"its blocker closed", base, nil, []forge.Issue{is}, true, true, 1, "#2, which it waits on, is closed", "state-revisit", 0},
		{"its blocker seen closed before", Record{State: BlockedState, Body: base.Body, Since: base.Since, Closed: []int{2}}, nil, []forge.Issue{is}, true, false, 1, "", "state-blocked-quiet", 0},
		{"a week on", base, nil, listing, true, true, 1, "nothing new since 2026-10-30, 7 days or more", "state-revisit", 7 * 24 * time.Hour},
		{"read again two days ago", Record{State: BlockedState, Body: base.Body, Since: "2026-09-01T09:00:00Z", Revisited: "2026-10-30T09:00:00Z"}, nil, listing, true, false, 1, "", "state-blocked-quiet", 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			wlclock.Now = func() time.Time { return now.Add(c.clockAdd) }
			b := &counted{comments: append([]string{recordComment(t, c.rec)}, c.extra...), open: c.open, calls: map[string]int{}}
			var open []forge.Issue
			if c.listed {
				open = c.open
			}
			res := &Result{}
			step, write, err := revisit(res, cfg, b, is, &open)
			if err != nil {
				t.Fatal(err)
			}
			if write != nil {
				t.Errorf("write = %+v, want none: its record is there", write)
			}
			if got := b.calls["comments"] + b.calls["issues"]; got != c.calls {
				t.Errorf("calls = %v, want %d", b.calls, c.calls)
			}
			if (step != nil) != c.wakes {
				t.Fatalf("step = %+v, want waking: %v", step, c.wakes)
			}
			if f := res.Findings[len(res.Findings)-1]; f.Rule != c.finding {
				t.Errorf("finding = %+v, want %s", f, c.finding)
			}
			if step == nil {
				return
			}
			if got := step.inputs["revisit"]; !strings.Contains(got, c.trigger) {
				t.Errorf("revisit = %q, want %q in it", got, c.trigger)
			}
			if step.inputs["revisit-from"] != "ready" || step.write.Record.Revisited == "" || step.write.Record.Hops != 9 {
				t.Errorf("step = %+v", step)
			}
		})
	}
}

// TestRevisitFirstSeen: an issue a role blocked, the line's record from
// the state before, is written once, its base, with no agent.
func TestRevisitFirstSeen(t *testing.T) {
	days := 7
	cfg := &routing.Config{Events: map[string][]string{IssueEvent: {"product-owner"}}, Revisit: &days}
	is := forge.Issue{ID: 3, Body: "x", Labels: []string{"workline:blocked"}, BlockedBy: []int{}}
	b := &counted{comments: []string{recordComment(t, Record{State: "tests", Body: "0", Hops: 2}), "A person's word."}, calls: map[string]int{}}
	open := []forge.Issue{is}
	step, write, err := revisit(&Result{}, cfg, b, is, &open)
	if err != nil || step != nil || write == nil {
		t.Fatalf("step %+v, write %+v, err %v", step, write, err)
	}
	r := write.Record
	if r.State != BlockedState || r.From != "tests" || r.Hops != 2 || r.Since == "" || r.Talk == "" || r.Body != bodyHash("x") {
		t.Errorf("record = %+v", r)
	}
}

// TestTalkHash: the engine's comments, a role's or the line's, never
// count; a person's, written or edited, does.
func TestTalkHash(t *testing.T) {
	engine := "x <!-- workline:sticky=line/state -->"
	if talkHash([]string{engine}) != "" {
		t.Error("the engine's comment counted")
	}
	a, b := talkHash([]string{engine, "Fixed."}), talkHash([]string{"Fixed, see #4."})
	if a == "" || a == b {
		t.Errorf("a person's comment: %q, edited %q", a, b)
	}
}
