#!/bin/sh
# A cmd: agent answering a reviewer's lens as Sonnet did on PR #157: Go
# code quoted under a block scalar with its tabs, the first line starting
# with one and a line pasted as it is in the file, at the start of the
# line. No YAML reader takes that; the engine indents it, the code kept.
cat >/dev/null
cat <<'YAML'
- finding:
    severity: nit
    title: The divisor is one less than the count
    why: |
      The mean of [2, 4] comes out as 6.
    cause:
      path: calc/calc.go
      quote: |
        	for _, v := range values {
        		total += v
	}
        	return total / (len(values) - 1)
YAML
