#!/bin/sh
# Keeps the line's verdict on a commit as `workline follow --verdict` keeps
# it (ADR-0042): a note under refs/notes/workline/verdicts, pushed to
# origin. For the merge policy's cases:
#   sh "$FIXTURES/verdict.sh" <commit> <status> [<findings, a JSON list>]
set -eu
head=$(git rev-parse "$1^{commit}")
git notes --ref=refs/notes/workline/verdicts add -f -m "{\"head\": \"$head\", \"status\": \"$2\", \"findings\": ${3:-[]}}" "$head"
git push -q origin refs/notes/workline/verdicts
