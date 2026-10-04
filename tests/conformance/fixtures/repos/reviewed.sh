#!/bin/sh
# A small Go project on main, and a branch `feature` to change it on: the
# reviewer's cases. calc.go holds a defect from before any change (Parse
# drops the error) and, on a line no case changes, a reference internal to
# the project, in the comment over Average.
set -eu
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
git init -q -b main .
git config user.name "Fixture"
git config user.email "fixture@example.invalid"
export GIT_AUTHOR_DATE="2026-01-01T00:00:00Z" GIT_COMMITTER_DATE="2026-01-01T00:00:00Z"
mkdir -p calc
cat > calc/calc.go <<'GO'
package calc

import "strconv"

// Average returns the mean of the values, rounded down (ABC-7).
func Average(values []int) int {
	total := 0
	for _, v := range values {
		total += v
	}
	return total / len(values)
}

// Parse reads a count.
func Parse(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
GO
printf '# Calc\n\nSmall sums.\n' > README.md
git add . && git commit -q -m "feat: start the project"
git checkout -q -b feature
