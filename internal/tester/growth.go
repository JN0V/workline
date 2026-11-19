package tester

// GrowthDue is true when current is above baseline by more than
// limitPercent percent.
func GrowthDue(baseline, current, limitPercent int) bool {
	return (current-baseline)*100 > baseline*limitPercent
}
