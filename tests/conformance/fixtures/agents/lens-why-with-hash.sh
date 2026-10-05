#!/bin/sh
# A cmd: agent answering a reviewer's lens as Sonnet answered the product
# owner on the sandbox (#156): its texts written plain, holding ` #20`,
# which a YAML reader takes for a comment, dropping the rest of the line.
cat >/dev/null
cat <<'YAML'
- finding:
    severity: nit # the least
    title: The divisor is one less than the count, as in #20
    why: The mean of [2, 4] comes out as 6; the half about empty lists is already covered by #20.
    cause:
      path: calc/calc.go
      quote: return total / (len(values) - 1)
YAML
