package line

import (
	"fmt"
	"os"
	"time"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/verdict"
)

// BudgetWaitMax is the longest a run of the line waits in place for the
// forge's API budget to be reset (#804). Short: a CI job waiting holds a
// runner, and on a busy hour twenty runs waiting would hold every runner
// a free plan gives, the pull requests' tests queued behind them. A reset
// further off defers the run to the hourly reconciler, which runs after
// it.
const BudgetWaitMax = 5 * time.Minute

// sleep is how the line waits; a test's own.
var sleep = time.Sleep

// waitBudget reads, before a run of the line reads its issues, what the
// token has left of the forge's API budget, when the forge says it at no
// cost (forge.Budgeted): a busy hour of the line spends a CI job's token
// (#804). Under a tenth, the run waits for its reset when it comes within
// BudgetWaitMax; further off, the run is deferred (defer true): it reads
// and runs nothing, and the issue's step stays due for the reconciler.
// What it did is returned, for the summary, and is a finding; "" when
// the budget is not low. A forge that does not answer is said by the
// run's first read.
func waitBudget(res *Result, f forge.Forge, now time.Time) (said string, deferred bool) {
	// A call refused for the rate half-way, its own or a role's, waits
	// for a reset as close, rather than stopping the step.
	if os.Getenv(forge.RateWaitEnv) == "" {
		_ = os.Setenv(forge.RateWaitEnv, BudgetWaitMax.String()) // a process's own environment: never fails on a valid name
	}
	bf, ok := f.(forge.Budgeted)
	if !ok {
		return "", false
	}
	b, err := bf.Budget()
	if err != nil || !b.Low() {
		return "", false
	}
	left := fmt.Sprintf("API budget low: %d of %d %s calls left this hour, reset at %s",
		b.Remaining, b.Limit, b.Resource, b.Reset.UTC().Format("15:04 UTC"))
	wait := max(b.Reset.Sub(now).Truncate(time.Second)+time.Second, 0)
	rule := "api-budget-low"
	switch {
	case wait > BudgetWaitMax:
		rule, deferred = "api-budget-deferred", true
		said = fmt.Sprintf("%s, %s away, over the %d min a run waits: deferred, nothing run: the hourly reconciler runs what is due after the reset",
			left, minutes(wait), int(BudgetWaitMax.Minutes()))
	default:
		said = fmt.Sprintf("%s: waited %s for it before reading the issues", left, minutes(wait))
		fmt.Fprintf(os.Stderr, "workline: %s: waiting %s for it\n", left, minutes(wait))
		sleep(wait)
	}
	res.Findings = append(res.Findings, verdict.Finding{Rule: rule, Level: "warn", Message: said})
	return said, deferred
}

// minutes says a wait in minutes, or seconds under one.
func minutes(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d s", int(d.Seconds()))
	}
	return fmt.Sprintf("%d min", int(d.Round(time.Minute).Minutes()))
}
