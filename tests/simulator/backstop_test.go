package simulator

import (
	"encoding/json"
	"maps"
	"slices"
	"testing"
)

// TestBackstopTrace holds ADR-0064's step 5 on its trace, every run
// crossing its target: the issue ends Waiting(person), blocked at its
// backstop, within the hard caps — its hops within max-hops, no machine
// error — its runs past the backstop by the one run that reached it at
// most.
func TestBackstopTrace(t *testing.T) {
	i := slices.IndexFunc(traces, func(tr trace) bool { return tr.name == "trial-35-every-run-crosses" })
	if i < 0 {
		t.Fatal("no trace trial-35-every-run-crosses")
	}
	tr := traces[i]
	f, log, m := measured(t, tr, nil, checks{all: true})
	if f != nil {
		t.Fatalf("%s\n%s", f, log)
	}
	const backstop, maxHops = 1600000, 8 // the seed's, 4× 400k; the shipped max-hops
	for _, is := range m.end.Issues {
		var r struct {
			State    string           `json:"state"`
			Hops     int              `json:"hops"`
			Backstop int              `json:"backstop"`
			Errors   any              `json:"errors"`
			Spent    map[string][]int `json:"spent"`
		}
		read := false
		for _, c := range is.Comments {
			if s := recordRe.FindStringSubmatch(c.Body); s != nil {
				read = json.Unmarshal([]byte(s[1]), &r) == nil
			}
		}
		if !read {
			t.Fatalf("#%d holds no line record\n%s", is.ID, log)
		}
		total := 0
		for _, runs := range r.Spent {
			for _, n := range runs {
				total += n
			}
		}
		switch {
		case is.Closed || r.State != "blocked" || r.Backstop != backstop:
			t.Errorf("#%d not blocked at its backstop: closed %v, %s\n%s", is.ID, is.Closed, trail(is), log)
		case total < backstop || total >= backstop+slices.Max(slices.Collect(maps.Values(tr.tokens))):
			t.Errorf("#%d spent %d fresh tokens: not past its backstop of %d by one run at most", is.ID, total, backstop)
		case r.Hops > maxHops || r.Errors != nil:
			t.Errorf("#%d past the hard caps: %d hops, errors %v", is.ID, r.Hops, r.Errors)
		}
		t.Logf("#%d: %d fresh tokens, %v, blocked at its backstop of %d, %d hops", is.ID, total, r.Spent, r.Backstop, r.Hops)
	}
}
