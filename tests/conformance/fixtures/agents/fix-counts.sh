#!/bin/sh
# A cmd: agent bringing each line count the task says is off to the engine's
# number, and nothing else, with no claim: the engine's count is the evidence
# (ADR-0014, step 2). Each doc keeps `checked` and records `judged`. $1
# `drop-approx` also drops the "~" before a count, the number now exact;
# `reword` says anew the rest of a line whose count it fixes ("Watch for
# growth." becomes "Well under the limit."), and rewords "lines per file" as
# "lines in a file", taking out only "per". It reads the counts, the commits and the numbered lines from the prompt, as a
# real agent would.
awk -v mode="$1" '
function flush(   i, l) {
  if (doc == "" || judged == "") return
  print "    --- a/" doc
  print "    +++ b/" doc
  print "    @@ -3 +3,2 @@"; print "     " line[3]; print "    +judged: " judged
  for (i = 1; i in line; i++) {
    reword = mode == "reword" && line[i] ~ /lines per file/
    if (!(i in fix) && !reword) continue
    l = line[i]
    if (i in fix && !(mode == "drop-approx" && sub("~" stated[i], real[i], l))) sub(stated[i], real[i], l)
    if (mode == "reword") { sub(/ Watch for growth\./, " Well under the limit.", l); sub(/lines per file/, "lines in a file", l) }
    print "    @@ -" i " +" i " @@"; print "    -" line[i]; print "    +" l
  }
}
/^## / { flush(); doc = substr($0, 4); judged = ""; delete line; delete fix; delete stated; delete real; next }
!judged && match($0, /`judged: [0-9a-f]+`/) { judged = substr($0, RSTART + 9, RLENGTH - 10) }
/^- `[^`]*` has [0-9]+ lines, not ~?[0-9,]+ \(line [0-9]+,/ {
  s = $0; sub(/^- `[^`]*` has /, "", s); r = s; sub(/ .*/, "", r)
  sub(/^[0-9]+ lines, not ~?/, "", s); st = s; sub(/ .*/, "", st)
  n = s; sub(/^[^(]*\(line /, "", n); sub(/,.*/, "", n)
  fix[n + 0] = 1; stated[n + 0] = st; real[n + 0] = r
}
/^ *[0-9]+ \| / { k = $1 + 0; sub(/^ *[0-9]+ \| /, ""); line[k] = $0 }
BEGIN { print "- patch: |" }
END { flush() }'
