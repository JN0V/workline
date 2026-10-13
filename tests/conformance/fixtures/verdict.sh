#!/bin/sh
# Writes the line's verdict on a commit as `workline route --verdict`
# writes it, for `workline follow --verdict` to keep (ADR-0042):
#   sh "$FIXTURES/verdict.sh" <commit> <status> <file> [<findings, a JSON list>]
set -eu
head=$(git rev-parse "$1^{commit}")
printf '{"head": "%s", "status": "%s", "findings": %s}\n' "$head" "$2" "${4:-[]}" > "$3"
