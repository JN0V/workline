#!/bin/sh
# A cmd: agent answering the documentalist's "suspect" task: it confirms
# every doc put before it, each moving `checked` to the commit the task
# asks for it and saying who vouched, one patch a doc. It reads each doc's
# path, its commit and its numbered lines from the prompt, as a real agent
# would.
awk '
  function flush() {
    if (!doc || !want || !at) return
    print "- patch: |"
    print "    --- a/" doc
    print "    +++ b/" doc
    printf "    @@ -%d,3 +%d,4 @@\n", at - 1, at - 1
    print "     " line[at - 1]
    print "    -" line[at]
    print "    +checked: " want
    print "    +verified: agent:documentalist"
    print "     " line[at + 1]
  }
  /^## [^ ]+\.md$/ { flush(); doc = $2; want = ""; at = 0; listing = 0; delete line; next }
  doc && !want && match($0, /sets `checked: [0-9a-f]+`/) { want = substr($0, RSTART + 15, RLENGTH - 16) }
  doc && /^The doc as it is now, with its line numbers:/ { listing = 1; next }
  listing && /^```$/ { if (seen) listing = 0; seen = !seen; next }
  listing && /^ *[0-9]+ \| / { n = $1 + 0; sub(/^ *[0-9]+ \| /, ""); line[n] = $0; if (!at && $0 ~ /^checked: /) at = n }
  END { flush() }'
