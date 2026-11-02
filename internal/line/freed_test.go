package line

import (
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// trailed is a backlog whose claim label was put on at claimed, per
// issue, as the forge's timeline says it.
type trailed struct {
	counted
	claimed map[int]string
}

func (t *trailed) Trail(id int, _ string) (forge.Trail, error) {
	t.calls["trail"]++
	return forge.Trail{Labeled: t.claimed[id]}, nil
}

// TestFreed: a ready issue woken once is free for the developer again
// when its claim was put on since and let go, a hop more, or an issue
// it was held back by is held no more (#894); never twice for one end,
// never on an issue the developer holds or a person's.
func TestFreed(t *testing.T) {
	cfg := lineConfig(t, "routing: {events: {issue: [developer]}}\n")
	ready, _ := cfg.State("ready")
	woke := "2026-11-01T10:48:40Z"
	b := &trailed{counted: counted{calls: map[string]int{}, open: []forge.Issue{
		{ID: 5, Labels: []string{"workline:developing"}},
	}}, claimed: map[int]string{3: "2026-11-01T10:49:33Z", 4: "2026-10-30T08:00:00Z"}}
	is := func(id int) forge.Issue { return forge.Issue{ID: id, Labels: []string{"workline:ready"}} }

	var open []forge.Issue
	rec, why, err := freed(cfg, b, ready, is(3), Record{Hops: 1, Woke: woke}, nil, &open)
	if err != nil || !strings.Contains(why, "its claim, workline:developing put on 2026-11-01, was let go") || rec.Hops != 2 {
		t.Errorf("a claim let go since the wake: %q, hops %d (%v)", why, rec.Hops, err)
	}
	if _, why, _ := freed(cfg, b, ready, is(4), Record{Hops: 1, Woke: woke}, nil, &open); why != "" {
		t.Errorf("a claim before the wake: freed, %q", why)
	}
	if b.calls["issues"] != 0 {
		t.Errorf("the open issues listed with no issue held at the wake: %v", b.calls)
	}

	// The wake's own record: the issue it is on is never one it held.
	open = nil
	rec, err = woken(cfg, b, ready, 7, Record{}, &open)
	if err != nil || rec.Woke == "" || len(rec.Held) != 1 || rec.Held[0] != 5 {
		t.Errorf("the wake's record: %+v (%v), want held [5]", rec, err)
	}
	if _, why, _ := freed(cfg, b, ready, is(7), rec, nil, &open); why != "" {
		t.Errorf("#5 still held: freed, %q", why)
	}
	open[0].Labels = []string{"workline:ready"} // #5 let go
	if _, why, _ := freed(cfg, b, ready, is(7), rec, nil, &open); !strings.HasPrefix(why, "#5, held when the line woke the developer on it, is held no more") {
		t.Errorf("#5 let go: %q", why)
	}

	mine := is(3)
	mine.Assignees = []string{"alice"}
	if _, why, _ := freed(cfg, b, ready, mine, Record{Hops: 1, Woke: woke}, nil, &open); why != "" {
		t.Errorf("a person's issue: freed, %q", why)
	}

	// A record from before Woke: once, when the developer said a word.
	said := []string{"**The developer** waits on this issue.\n\n" + developerSaid}
	if _, why, _ := freed(cfg, b, ready, is(8), Record{Hops: 1}, said, &open); why == "" {
		t.Error("an old record the developer left free: not freed")
	}
	if _, why, _ := freed(cfg, b, ready, is(8), Record{Hops: 1}, nil, &open); why != "" {
		t.Errorf("an old record, nothing said by the developer: freed, %q", why)
	}
}
