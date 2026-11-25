package tester

import (
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/kit/testmark"
)

// keptPrefix opens the reason of a scenario kept as it is today. Beside
// its tests, it makes them guards: green on the base, as they must be.
const keptPrefix = "kept as today"

// guardsOf are the tests the covers entries declare as guards, those of
// a scenario kept as today, its reason saying so, and those scenarios.
// A test another scenario's entry declares too is no guard: it must be
// red for that one.
func guardsOf(covers []declared) (ids, scenarios []string) {
	kept := func(c declared) bool { return strings.HasPrefix(strings.ToLower(c.Reason), keptPrefix) }
	for _, c := range covers {
		if kept(c) && len(c.Tests) > 0 {
			scenarios = append(scenarios, c.Scenario)
			ids = append(ids, slices.DeleteFunc(slices.Clone(c.Tests), func(id string) bool {
				return slices.ContainsFunc(covers, func(o declared) bool { return !kept(o) && slices.Contains(o.Tests, id) })
			})...)
		}
	}
	return ids, scenarios
}

// unprovenScenarios are the issue's scenarios, in its order, that no test
// proves, each with its reason: said rather than tested.
func unprovenScenarios(t task, said map[string]string) []testmark.Unproven {
	var out []testmark.Unproven
	for _, sc := range t.Scenarios {
		if n, ok := said[sc.Name]; ok {
			out = append(out, testmark.Unproven{Scenario: sc.Name, Why: strings.Join(strings.Fields(n), " ")})
		}
	}
	return out
}
