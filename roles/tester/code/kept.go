package tester

import (
	"strings"

	"github.com/JN0V/workline/internal/kit/testmark"
)

// keptPrefix opens the reason of a scenario kept as it is today. Beside
// its tests, it makes them guards: green on the base, as they must be.
const keptPrefix = "kept as today"

// guardsOf are the tests the covers entries declare as guards: those of a
// scenario kept as today, its reason saying so.
func guardsOf(covers []declared) (ids []string) {
	for _, c := range covers {
		if strings.HasPrefix(strings.ToLower(c.Reason), keptPrefix) {
			ids = append(ids, c.Tests...)
		}
	}
	return ids
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
