package simulator

import (
	"slices"
	"testing"
)

// TestCodeNamedStartsNoSecondStep: Given a product owner's answer naming
// the code of an issue in refine (trial 37 replayed), When the line runs
// again in the same tick, Then it starts no step on that issue: the
// same-tick-twice property holds on every tick of the trace.
func TestCodeNamedStartsNoSecondStep(t *testing.T) {
	i := slices.IndexFunc(traces, func(tr trace) bool { return tr.name == "trial-37-replay" })
	if i < 0 {
		t.Fatal("no trace named trial-37-replay")
	}
	f, log := runTrace(t, traces[i], nil, checks{only: []string{pTwice}, until: 1 << 30})
	if f != nil {
		t.Errorf("%s\n%s", f, log)
	}
}
