package tester

import (
	"slices"
	"testing"
)

const (
	conformanceFile = "tests/conformance/cases/merge/local-fix-merges-alone.yaml"
	conformanceCase = "merge/local-fix-merges-alone"
	conformanceFrozen = conformanceFile + ":" + conformanceCase
	conformanceID = "github.com/JN0V/workline/tests/conformance.TestConformance/" + conformanceCase
)

// Given a case frozen as `<file>.yaml:<folder>/<case>`, When the head's
// report names it `<package>.TestConformance/<folder>/<case>` and it
// passes, Then the frozen test is found in the head's reports.
func TestFrozenInConformanceSubtest(t *testing.T) {
	r := record{Files: map[string]string{conformanceFile: "a1"}}
	ids := map[string]bool{
		conformanceID: true,
		"github.com/JN0V/workline/tests/conformance.TestConformance": true,
	}
	if got, want := r.frozenIn(conformanceFrozen, ids), []string{conformanceID}; !slices.Equal(got, want) {
		t.Errorf("frozenIn(%q) = %q, want %q", conformanceFrozen, got, want)
	}
	if !namedAs(conformanceID, conformanceCase, conformanceFile) {
		t.Errorf("namedAs(%q, %q, %q) = false, want true", conformanceID, conformanceCase, conformanceFile)
	}
}

// Given the same frozen case, When the head's reports do not name it,
// Then it is still missing, and the judge's finding is frozen-test-missing:
// a neighbouring case that a report names is found, this one is not.
func TestFrozenInConformanceSubtestMissing(t *testing.T) {
	other := "tests/conformance/cases/merge/local-red-line-holds.yaml"
	r := record{Files: map[string]string{conformanceFile: "a1", other: "b1"}}
	otherID := "github.com/JN0V/workline/tests/conformance.TestConformance/merge/local-red-line-holds"
	ids := map[string]bool{
		otherID: true,
		"github.com/JN0V/workline/tests/conformance.TestConformance/merge/local-fix-merges-alone-not": true,
		"github.com/JN0V/workline/tests/conformance.TestConformance":                                   true,
	}
	if got, want := r.frozenIn(other+":merge/local-red-line-holds", ids), []string{otherID}; !slices.Equal(got, want) {
		t.Errorf("frozenIn of the case a report names = %q, want %q", got, want)
	}
	if got := r.frozenIn(conformanceFrozen, ids); len(got) != 0 {
		t.Errorf("frozenIn(%q) = %q, want none: no report names it", conformanceFrozen, got)
	}
	f := missingFinding(conformanceFrozen, greenRuns, "")
	if f.Rule != "frozen-test-missing" || f.Where != conformanceFrozen {
		t.Errorf("missingFinding = %s at %s, want frozen-test-missing at %s", f.Rule, f.Where, conformanceFrozen)
	}
}

// Given the same frozen case, When a report names `<folder>/<case>` under
// another test or package, Then it is not taken for the frozen one.
func TestFrozenInConformanceSubtestElsewhere(t *testing.T) {
	r := record{Files: map[string]string{conformanceFile: "a1"}}
	ids := map[string]bool{
		conformanceID: true,
		"github.com/JN0V/workline/tests/conformance.TestOther/" + conformanceCase: true,
		"github.com/JN0V/workline/internal/line.TestConformance/" + conformanceCase:  true,
		"github.com/JN0V/workline/cmd/conformance.TestConformance/" + conformanceCase: true,
		conformanceCase: true,
	}
	if got, want := r.frozenIn(conformanceFrozen, ids), []string{conformanceID}; !slices.Equal(got, want) {
		t.Errorf("frozenIn(%q) = %q, want only %q", conformanceFrozen, got, want)
	}
	for _, id := range []string{
		"github.com/JN0V/workline/tests/conformance.TestOther/" + conformanceCase,
		"github.com/JN0V/workline/internal/line.TestConformance/" + conformanceCase,
	} {
		if namedAs(id, conformanceCase, conformanceFile) {
			t.Errorf("namedAs(%q, %q, %q) = true, want false", id, conformanceCase, conformanceFile)
		}
	}
}
