package line

import (
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/routing"
)

// TestPartsInput: the parts closed, in order, and a person's yes.
func TestPartsInput(t *testing.T) {
	for _, c := range []struct {
		closed []int
		labels []string
		want   string
	}{
		{nil, nil, ""},
		{[]int{6}, nil, "6"},
		{[]int{6, 7}, []string{"workline:accepted"}, "6,7+accepted"},
		{nil, []string{"workline::accepted"}, "+accepted"},
	} {
		if got := partsInput(c.closed, forge.Issue{Labels: c.labels}); got != c.want {
			t.Errorf("partsInput(%v, %v) = %q, want %q", c.closed, c.labels, got, c.want)
		}
	}
}

// TestParentStep: a parent is due once for each change of its parts
// closed or a yes; else it waits, said, on its open parts or a person;
// a blocked one with nothing new goes back to revisit; an issue with no
// parts, or a project not routing the product owner, is not one.
func TestParentStep(t *testing.T) {
	cfg := lineConfig(t, "routing: {events: {issue: [product-owner]}}\n")
	blocked, _ := cfg.State(BlockedState)
	var noState routing.State
	parent := forge.Issue{ID: 4, Children: []int{6, 7}}
	listing := func(ids ...int) []forge.Issue {
		out := []forge.Issue{{ID: 4, Children: []int{6, 7}}}
		for _, n := range ids {
			out = append(out, forge.Issue{ID: n})
		}
		return out
	}
	newB := func(rec *Record, open []forge.Issue) *counted {
		b := &counted{calls: map[string]int{}, open: open}
		if rec != nil {
			b.comments = []string{recordComment(t, *rec)}
		}
		return b
	}

	// Read alone, its parts from the listing: one closed, due.
	b := newB(nil, listing(7))
	var open []forge.Issue
	res := &Result{}
	st, handled, err := parentStep(res, cfg, b, forge.Issue{ID: 4}, noState, &open)
	if err != nil || !handled || st == nil || st.write.Record.Parts != "6" || st.ai != "none" || !st.parent {
		t.Fatalf("one part closed: %+v, %v (%v)", st, handled, err)
	}
	if b.calls["issues"] != 1 {
		t.Errorf("the open issues listed %d times, want once", b.calls["issues"])
	}

	// The same seen: it waits on #7, said.
	res = &Result{}
	open = nil
	st, handled, _ = parentStep(res, cfg, newB(&Record{Parts: "6"}, listing(7)), parent, noState, &open)
	if st != nil || !handled || len(res.Findings) != 1 || !strings.Contains(res.Findings[0].Message, "waits on its open parts, #7") {
		t.Errorf("one part open, seen: %+v %v %+v", st, handled, res.Findings)
	}

	// All closed and seen: waits for a person.
	res = &Result{}
	open = nil
	st, handled, _ = parentStep(res, cfg, newB(&Record{Parts: "6,7"}, listing()), parent, noState, &open)
	if st != nil || !handled || len(res.Findings) != 1 || !strings.Contains(res.Findings[0].Message, "waits for a person") {
		t.Errorf("all closed, seen: %+v %v %+v", st, handled, res.Findings)
	}

	// A yes put on: due again.
	res = &Result{}
	open = nil
	yes := parent
	yes.Labels = []string{"workline:accepted"}
	st, _, _ = parentStep(res, cfg, newB(&Record{Parts: "6,7"}, listing()), yes, noState, &open)
	if st == nil || st.write.Record.Parts != "6,7+accepted" {
		t.Errorf("a yes put on: %+v", st)
	}

	// A part reopened: due, its input smaller.
	res = &Result{}
	open = nil
	st, _, _ = parentStep(res, cfg, newB(&Record{Parts: "6,7"}, listing(6)), parent, noState, &open)
	if st == nil || st.write.Record.Parts != "7" {
		t.Errorf("a part reopened: %+v", st)
	}

	// Blocked, nothing new: revisit reads it.
	res = &Result{}
	open = nil
	st, handled, _ = parentStep(res, cfg, newB(&Record{Parts: "6", State: BlockedState}, listing(7)), parent, blocked, &open)
	if st != nil || handled {
		t.Errorf("blocked, nothing new: %+v %v", st, handled)
	}

	// No parts: not a parent.
	res = &Result{}
	open = nil
	if st, handled, _ := parentStep(res, cfg, newB(nil, listing()), forge.Issue{ID: 9}, noState, &open); st != nil || handled {
		t.Errorf("no parts: %+v %v", st, handled)
	}

	// The product owner not routed: nobody to tell it.
	off := lineConfig(t, "routing: {events: {issue: [developer]}}\n")
	open = nil
	if st, handled, _ := parentStep(&Result{}, off, newB(nil, listing(7)), parent, noState, &open); st != nil || handled {
		t.Errorf("not routed: %+v %v", st, handled)
	}
}
