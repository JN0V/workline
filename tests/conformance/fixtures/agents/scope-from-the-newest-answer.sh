#!/bin/sh
# A cmd: agent playing the product owner on issue 9: it writes into the
# Scope the word of the answer its task holds. Given the answer about a
# returned sale, it writes that; given only the earlier ones, it writes
# theirs.
task=$(cat)
case "$task" in
*"returned sale"*)
  scope='`src/export/csv.go`: the writer. A returned sale is counted, as Ann answered last.'
  ;;
*)
  scope='`src/export/csv.go`: the writer. The last day of a month is read.'
  ;;
esac
printf '%s\n' "- refine: {issue: 9, scope: \"$scope\", sources: [src/export/csv.go], why: \"Ann's answers settle it: written into the Scope.\"}"
