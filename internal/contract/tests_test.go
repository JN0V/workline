package contract

import (
	"testing"
	"time"

	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/kit/verdict"
)

// wholeTests is a Tests with every function set, none called.
func wholeTests() *Tests {
	return &Tests{
		CommandRuns: func(string, string, map[string]any) (string, *verdict.Finding, error) { return "", nil, nil },
		Settings:    func(map[string]any) (TestSettings, error) { return TestSettings{}, nil },
		OnChange: func(string, string, []intent.Intention, TestSettings, time.Time, TestScope) (TestOutcome, *verdict.Finding) {
			return TestOutcome{}, nil
		},
	}
}

// The project's tests are asked of the role offering them, one role at
// most; none offered, none found.
func TestTheTestsAreAskedOfTheRoleOfferingThem(t *testing.T) {
	r := NewRegistry()
	if tt, n := r.Tests(); tt != nil || n != "" {
		t.Fatalf("Tests = %v, %q with no role offering them", tt, n)
	}
	r.Join("runner", Role{Tests: wholeTests()})
	if tt, n := r.Tests(); tt == nil || n != "runner" {
		t.Errorf("Tests = %v, %q", tt, n)
	}
	defer func() {
		if recover() == nil {
			t.Error("a second role offering the tests is taken")
		}
	}()
	r.Join("other", Role{Tests: wholeTests()})
}

// Tests missing a function are refused at Join.
func TestTestsMissingAFunctionAreRefused(t *testing.T) {
	for what, tt := range map[string]func(*Tests){
		"CommandRuns": func(t *Tests) { t.CommandRuns = nil },
		"Settings":    func(t *Tests) { t.Settings = nil },
		"OnChange":    func(t *Tests) { t.OnChange = nil },
	} {
		missing := wholeTests()
		tt(missing)
		if !refused(func() { NewRegistry().Join("x", Role{Tests: missing}) }) {
			t.Errorf("tests missing %s are taken", what)
		}
	}
}
