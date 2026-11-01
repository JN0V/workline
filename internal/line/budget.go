package line

import (
	"fmt"
	"os"
	"time"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/verdict"
)

// BudgetWaitMax is the longest a run of the line waits for the forge's
// API budget to be reset: GitHub resets it within the hour (#804).
const BudgetWaitMax = time.Hour + time.Minute

// sleep is how the line waits; a test's own.
var sleep = time.Sleep

// waitBudget reads, before a run of the line reads its issues, what the
// token has left of the forge's API budget, when the forge says it at no
// cost (forge.Budgeted). Under a tenth, the run waits for its reset, up
// to BudgetWaitMax, rather than stopping half-way on the rate: a busy
// hour of the line spends a CI job's token (#804). What it waited is
// returned, for the summary, and is a finding; "" when the budget is not
// low. A forge that does not answer is said by the run's first read.
func waitBudget(res *Result, f forge.Forge, now time.Time) string {
	// A call refused for the rate half-way, its own or a role's, waits
	// for the reset too, rather than stopping the step.
	if os.Getenv(forge.RateWaitEnv) == "" {
		_ = os.Setenv(forge.RateWaitEnv, BudgetWaitMax.String()) // a process's own environment: never fails on a valid name
	}
	bf, ok := f.(forge.Budgeted)
	if !ok {
		return ""
	}
	b, err := bf.Budget()
	if err != nil || !b.Low() {
		return ""
	}
	left := fmt.Sprintf("API budget low: %d of %d %s calls left this hour, reset at %s",
		b.Remaining, b.Limit, b.Resource, b.Reset.UTC().Format("15:04 UTC"))
	wait := max(b.Reset.Sub(now).Truncate(time.Second)+time.Second, 0)
	var said string
	switch {
	case wait > BudgetWaitMax:
		said = fmt.Sprintf("%s, over %d min away: not waited for, the run may stop on the rate", left, int(BudgetWaitMax.Minutes()))
	default:
		said = fmt.Sprintf("%s: waited %s for it before reading the issues", left, minutes(wait))
		fmt.Fprintf(os.Stderr, "workline: %s: waiting %s for it\n", left, minutes(wait))
		sleep(wait)
	}
	res.Findings = append(res.Findings, verdict.Finding{Rule: "api-budget-low", Level: "warn", Message: said})
	return said
}

// minutes says a wait in minutes, or seconds under one.
func minutes(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d s", int(d.Seconds()))
	}
	return fmt.Sprintf("%d min", int(d.Round(time.Minute).Minutes()))
}
