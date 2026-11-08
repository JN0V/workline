#!/bin/sh
# A cmd: agent whose call says it spent $1 fresh tokens in, none out, and
# that proposes nothing: an issue's runs adding up to its backstop
# (ADR-0064, §5).
cat >/dev/null
printf 'tokens-in: %s\ntokens-out: 0\n' "$1" > "$WORKLINE_CALL"
echo "[]"
