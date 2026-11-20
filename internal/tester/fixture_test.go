package tester

import (
	"os"
	"testing"

	"github.com/JN0V/workline/internal/contract"
	"github.com/JN0V/workline/internal/kit/judgemark"
	"github.com/JN0V/workline/internal/kit/verdict"
)

// The tester's tests run beside a fixture reviewer (docs/spec/layout.md,
// Tests): what the tester asks of the review (contract.Review) answers
// nothing found, nothing served; the real reviewer is tried in its own
// package and, linked whole, by the conformance cases.
func TestMain(m *testing.M) {
	contract.Join("fixture-reviewer", contract.Role{Review: &contract.Review{
		ReviewedAll: func(string, string, string, string) (bool, error) { return false, nil },
		RulesOn:     func(string, string, string) ([]verdict.Finding, error) { return nil, nil },
		ServeReads: func(string, string, string, []judgemark.Read, int, int) (string, int, []string) {
			return "", 0, nil
		},
	}})
	os.Exit(m.Run())
}
