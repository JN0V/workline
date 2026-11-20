package tester

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// Redundant tests: Given test X covers a set of lines and test Y covers
// only lines that X also covers, When the tester gardens the corpus,
// Then it proposes merging them. It only proposes; it changes nothing
// itself, and coverage stays the same after the merge.
func TestMergeCandidatesSubset(t *testing.T) {
	cov := map[string]map[string]bool{
		"TestWide":   {"a.go:1": true, "a.go:2": true, "a.go:3": true},
		"TestNarrow": {"a.go:1": true, "a.go:3": true},
		"TestApart":  {"b.go:9": true},
	}
	before := map[string]map[string]bool{}
	for name, lines := range cov {
		before[name] = map[string]bool{}
		for l := range lines {
			before[name][l] = true
		}
	}
	got := MergeCandidates(cov)
	if len(got) != 1 {
		t.Fatalf("got %d proposals %v, want one", len(got), got)
	}
	text := fmt.Sprintf("%v", got[0])
	if !strings.Contains(text, "TestWide") || !strings.Contains(text, "TestNarrow") {
		t.Errorf("the proposal %q does not name TestWide and TestNarrow", text)
	}
	if strings.Contains(text, "TestApart") {
		t.Errorf("the proposal %q names TestApart, which covers a line no other test does", text)
	}
	if !reflect.DeepEqual(cov, before) {
		t.Errorf("MergeCandidates changed the coverage it read: %v, was %v", cov, before)
	}
	// Two tests that each cover a line the other misses are never merged.
	each := map[string]map[string]bool{
		"TestLeft":  {"a.go:1": true, "a.go:2": true},
		"TestRight": {"a.go:2": true, "a.go:3": true},
	}
	if got := MergeCandidates(each); len(got) != 0 {
		t.Errorf("tests that each cover a line the other misses: got %v, want no proposal", got)
	}
}
