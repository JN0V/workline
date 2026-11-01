package line

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/JN0V/workline/internal/forge"
)

// budgeted is a forge saying a budget, nothing else asked of it.
type budgeted struct {
	forge.Forge
	b forge.Budget
}

func (f budgeted) Budget() (forge.Budget, error) { return f.b, nil }

func TestWaitBudget(t *testing.T) {
	t.Setenv(forge.RateWaitEnv, "") // set by the run, put back after
	now := time.Date(2026, 11, 1, 9, 3, 0, 0, time.UTC)
	for _, c := range []struct {
		name   string
		b      forge.Budget
		waited time.Duration
		said   string
	}{
		{"plenty", forge.Budget{Resource: "core", Remaining: 400, Limit: 1000, Reset: now.Add(time.Hour)}, 0, ""},
		{"low: waits for the reset", forge.Budget{Resource: "core", Remaining: 12, Limit: 1000, Reset: now.Add(37 * time.Minute)},
			37*time.Minute + time.Second, "API budget low: 12 of 1000 core calls left this hour, reset at 09:40 UTC: waited 37 min for it"},
		{"spent, reset passed: no wait", forge.Budget{Resource: "graphql", Remaining: 0, Limit: 1000, Reset: now.Add(-time.Minute)},
			0, "0 of 1000 graphql calls left"},
		{"reset too far: said, not waited", forge.Budget{Resource: "core", Remaining: 0, Limit: 5000, Reset: now.Add(3 * time.Hour)},
			0, "not waited for"},
	} {
		t.Run(c.name, func(t *testing.T) {
			var waited time.Duration
			sleep = func(d time.Duration) { waited += d }
			defer func() { sleep = time.Sleep }()
			res := &Result{}
			said := waitBudget(res, budgeted{b: c.b}, now)
			if os.Getenv(forge.RateWaitEnv) != BudgetWaitMax.String() {
				t.Errorf("%s = %q: a call refused for the rate half-way would not wait", forge.RateWaitEnv, os.Getenv(forge.RateWaitEnv))
			}
			if waited != c.waited {
				t.Errorf("waited %s, want %s", waited, c.waited)
			}
			if c.said == "" {
				if said != "" || len(res.Findings) != 0 {
					t.Errorf("said %q, %d findings; want nothing", said, len(res.Findings))
				}
				return
			}
			if !strings.Contains(said, c.said) {
				t.Errorf("said %q, want it to hold %q", said, c.said)
			}
			if len(res.Findings) != 1 || res.Findings[0].Rule != "api-budget-low" {
				t.Errorf("findings %+v, want one api-budget-low", res.Findings)
			}
		})
	}
}
