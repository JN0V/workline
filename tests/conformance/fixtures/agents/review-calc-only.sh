#!/bin/sh
# A cmd: lens of the reviewer, on the `reviewed` fixture once Average divides
# by one less than the count: it finds that defect only when its task gives
# it calc/calc.go, nothing otherwise. A run that reviews again commits a run
# before it reviewed asks the judge again; one that does not, never (#375).
if grep -q '^### calc/calc.go' ; then
	cat <<'YAML'
- finding:
    severity: important
    title: Average divides by one less than the count
    why: the mean of [2, 4] comes out 6, and one value divides by zero
    cause: {path: calc/calc.go, quote: "return total / (len(values) - 1)"}
    fix: divide by len(values)
YAML
else
	echo '[]'
fi
