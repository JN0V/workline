package simulator

import (
	"slices"
	"testing"
)

// TestUnreadTwiceNotAskedAgainInTick: Given a product owner's run whose
// answer could not be read twice (trace trial-35-po-unread), When the
// line runs again in the same tick, Then it starts no step on that issue:
// the trace passes the same-tick-twice property, whatever pending.txt
// says of it.
func TestUnreadTwiceNotAskedAgainInTick(t *testing.T) {
	i := slices.IndexFunc(traces, func(tr trace) bool { return tr.name == "trial-35-po-unread" })
	if i < 0 {
		t.Fatal("no trace trial-35-po-unread")
	}
	f, log, _ := measured(t, traces[i], nil, checks{only: []string{pTwice}, until: 1 << 30})
	if f != nil {
		t.Errorf("%s\n%s", f, log)
	}
}
