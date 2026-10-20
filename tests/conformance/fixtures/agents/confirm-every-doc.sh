#!/bin/sh
# A cmd: agent answering the documentalist's "suspect" task: it finds nothing
# wrong in any doc put before it, one patch a doc. A doc it may vouch for gets
# `checked` moved to the commit the task asks for, and `verified` when the
# header has none; one it may not, `judged` at the commit the task names. It
# reads each doc's path, its commits and its numbered lines from the prompt,
# as a real agent would; with none to answer, an empty list.
awk '
  function flush(   i, e, n, out, has_v, has_j, line_i, ch) {
    if (!doc || !(want || judge) || h[1] != "---") return
    for (e = 2; e <= hn && h[e] != "---"; e++) {
      if (h[e] ~ /^verified: /) has_v = 1
      if (h[e] ~ /^judged: /) has_j = 1
    }
    if (e > hn) return
    n = 0
    for (i = 1; i <= e; i++) {
      line_i = h[i]
      if (want && line_i ~ /^checked: [0-9a-f]+$/) {
        out[++n] = "    -" line_i; out[++n] = "    +checked: " want; ch = 1
        if (!has_v) out[++n] = "    +verified: agent:documentalist"
        added += has_v ? 0 : 1
      } else if (!want && has_j && line_i ~ /^judged: /) {
        out[++n] = "    -" line_i; out[++n] = "    +judged: " judge; ch = 1
      } else if (!want && !has_j && line_i ~ /^checked: [0-9a-f]+$/) {
        out[++n] = "     " line_i; out[++n] = "    +judged: " judge; ch = 1
        added++
      } else {
        out[++n] = "     " line_i
      }
    }
    if (!ch) return
    print "- patch: |"
    print "    --- a/" doc
    print "    +++ b/" doc
    printf "    @@ -1,%d +1,%d @@\n", e, e + added
    for (i = 1; i <= n; i++) print out[i]
    any = 1
  }
  /^## [^ ]+\.md$/ { flush(); doc = $2; want = ""; judge = ""; listing = 0; seen = 0; hn = 0; added = 0; delete h; next }
  doc && !want && match($0, /sets `checked: [0-9a-f]+`/) { want = substr($0, RSTART + 15, RLENGTH - 16) }
  doc && !judge && match($0, /and sets `judged: [0-9a-f]+`, the commit it is judged at/) { s = substr($0, RSTART + 18); judge = substr(s, 1, index(s, "`") - 1) }
  doc && /^The doc as it is now, with its line numbers:/ { listing = 1; next }
  listing && /^```$/ { if (seen) listing = 0; seen = !seen; next }
  listing && /^ *[0-9]+ \| / { n = $1 + 0; sub(/^ *[0-9]+ \| /, ""); if (n == hn + 1) h[++hn] = $0 }
  END { flush(); if (!any) print "[]" }'
