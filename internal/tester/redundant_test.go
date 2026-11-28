package tester

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// Redundant tests: Given test X covers a set of lines and test Y covers
// only lines that X also covers, When the tester gardens the corpus, Then
// it proposes merging them. It only proposes: the coverage it read is left
// as it was, so the lines stay covered by what remains.
func TestMergeCandidatesSubset(t *testing.T) {
	cov := map[string]map[string]bool{
		"TestX": {"a.go:1": true, "a.go:2": true, "a.go:3": true},
		"TestY": {"a.go:1": true, "a.go:2": true},
		"TestZ": {"b.go:1": true},
	}
	before := map[string]map[string]bool{
		"TestX": {"a.go:1": true, "a.go:2": true, "a.go:3": true},
		"TestY": {"a.go:1": true, "a.go:2": true},
		"TestZ": {"b.go:1": true},
	}
	got := MergeCandidates(cov)
	if len(got) != 1 {
		t.Fatalf("got %d proposals %+v, want one: Y into X", len(got), got)
	}
	p := fmt.Sprintf("%+v", got[0])
	if !strings.Contains(p, "TestX") || !strings.Contains(p, "TestY") || strings.Contains(p, "TestZ") {
		t.Errorf("the proposal %s must name TestX and TestY, not TestZ", p)
	}
	if !reflect.DeepEqual(cov, before) {
		t.Errorf("the coverage was changed: %v", cov)
	}
	// Two tests that each cover a line the other misses are not proposed.
	both := map[string]map[string]bool{
		"TestP": {"a.go:1": true, "a.go:2": true},
		"TestQ": {"a.go:2": true, "a.go:3": true},
	}
	if got := MergeCandidates(both); len(got) != 0 {
		t.Errorf("tests each covering a line the other misses were proposed: %+v", got)
	}
}
