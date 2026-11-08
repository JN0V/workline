#!/bin/sh
# A cmd: agent playing the product owner asked on issue 3's size
# (ADR-0064, step 4): its need is one file, atomic, so it grants a tier
# ($1, large by default) and sends the issue back to its tests, or, with
# $2 `blocked`, keeps it blocked for a person.
cat >/dev/null
echo "- size: {issue: 3, tier: ${1:-large}, why: \"one file of 100,000 characters is the whole need: no smaller part proves it\"}"
if [ "${2:-}" = blocked ]; then
  echo '- blocked: {issue: 3, why: "a larger size waits for a person", person: true}'
else
  echo '- unblock: {issue: 3, to: tests, why: "granted a larger size"}'
fi
