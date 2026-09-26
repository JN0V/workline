#!/bin/sh
# A cmd: agent answering the documentalist's "suspect" task: it confirms the
# first doc put before it, moving `checked` to the commit the task asks for,
# whatever the doc. It reads the doc's path, the commit and the numbered lines
# from the prompt, as a real agent would.
awk '
  !doc && /^## [^ ]+\.md$/ { doc = $2 }
  doc && !want && match($0, /sets `checked: [0-9a-f]+`/) { want = substr($0, RSTART + 15, RLENGTH - 16) }
  doc && /^ *[0-9]+ \| / { n = $1 + 0; sub(/^ *[0-9]+ \| /, ""); line[n] = $0; if (!at && $0 ~ /^checked: /) at = n }
  END {
    if (!doc || !want || !at) exit 0
    print "- patch: |"
    print "    --- a/" doc
    print "    +++ b/" doc
    printf "    @@ -%d,3 +%d,3 @@\n", at - 1, at - 1
    print "     " line[at - 1]
    print "    -" line[at]
    print "    +checked: " want
    print "     " line[at + 1]
  }'
