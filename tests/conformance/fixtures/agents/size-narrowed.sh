#!/bin/sh
# A cmd: agent playing the product owner asked on issue 3's size
# (ADR-0064, step 3): it narrows the Scope a person wrote to the one
# file its need needs ($1, lib.sh by default), then sends it back to its
# tests; with $2 `unblock-first`, it writes the unblock before the
# refine.
cat >/dev/null
f=${1:-lib.sh}
refine="- refine: {issue: 3, scope: \"- \`$f\`: multiply beside add.\", sources: [$f], why: \"the wide file is not what the need needs\"}"
unblock='- unblock: {issue: 3, to: tests, why: "its Scope narrowed to the file its need needs"}'
if [ "${2:-}" = unblock-first ]; then
  printf '%s\n%s\n' "$unblock" "$refine"
else
  printf '%s\n%s\n' "$refine" "$unblock"
fi
