#!/bin/sh
# A cmd: agent for the `drifted` cases, to prove their grades without a real
# one (checks_test.go). It changes headers only: for each doc put before it,
# it vouches (`checked` moved to the commit the task names) or records
# `judged`, as $1 says for all, `careful` vouching for the clean control
# alone. Every doc of the fixture holds `checked` on its third line.
mode=$1
prompt=$(cat)
printf '%s\n' "$prompt" | awk -v mode="$mode" '
/^## / { doc = substr($0, 4) }
/your patch sets `checked: [0-9a-f]+`/ {
  match($0, /`checked: [0-9a-f]+`/); want[doc] = substr($0, RSTART + 10, RLENGTH - 11); order[++n] = doc
}
/^ *3 \| checked: / { if (doc in want && !(doc in old)) { sub(/^ *3 \| checked: /, ""); old[doc] = $0 } }
END {
  print "- patch: |"
  for (i = 1; i <= n; i++) {
    d = order[i]
    vouch = mode == "vouches" || mode == "careful" && d == "docs/clock/events.md"
    print "    --- a/" d; print "    +++ b/" d
    if (vouch) {
      print "    @@ -3 +3 @@"; print "    -checked: " old[d]; print "    +checked: " want[d]
    } else {
      print "    @@ -3 +3,2 @@"; print "     checked: " old[d]; print "    +judged: " want[d]
    }
  }
}'
