#!/bin/sh
# A cmd: agent answering a reviewer's lens as Sonnet did on PR #137 (#138):
# its `why` starts with a backtick, written plain, which no YAML reader
# reads. Asked again with what the reader said, it writes the texts as
# block scalars, code and quotes as they are.
prompt=$(cat)
case "$prompt" in
*"Your previous answer was refused"*"cannot start any token"*"block scalar"*)
	cat <<'YAML'
- finding:
    severity: nit
    title: "The divisor: one less than the count"
    why: |
      `Average` divides by `len(values) - 1`: # the mean of [2, 4] is 6
    cause:
      path: calc/calc.go
      quote: |
        return total / (len(values) - 1)
YAML
	;;
*)
	cat <<'YAML'
- finding:
    severity: nit
    title: The divisor is one less than the count
    why: `Average` divides by `len(values) - 1`: the mean of [2, 4] is 6
    cause: {path: calc/calc.go, quote: "return total / (len(values) - 1)"}
YAML
	;;
esac
