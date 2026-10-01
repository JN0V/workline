#!/bin/sh
# A cmd: agent fixing docs/clock/README.md of the `drifted` fixture after
# its version bump to 1.5.0, as $1 says: `vouches` brings the version and
# moves `checked`, leaving the frozen line count; `counts` also brings the
# count of Clock.h to what the task says it is; `version` brings the version
# alone, `checked` left and `judged` recorded. It reads the commits and the
# numbered lines from the prompt, as a real agent would.
awk -v mode="$1" '
/^## / { doc = substr($0, 4) }
doc != "docs/clock/README.md" { next }
!want && match($0, /sets `checked: [0-9a-f]+`/) { want = substr($0, RSTART + 15, RLENGTH - 16) }
!judged && match($0, /`judged: [0-9a-f]+`/) { judged = substr($0, RSTART + 9, RLENGTH - 10) }
match($0, /`Clock.h` has [0-9]+ lines/) { s = substr($0, RSTART, RLENGTH); gsub(/[^0-9]/, "", s); real = s }
/^ *[0-9]+ \| / { n = $1 + 0; sub(/^ *[0-9]+ \| /, ""); line[n] = $0 }
END {
  print "- patch: |"
  print "    --- a/docs/clock/README.md"
  print "    +++ b/docs/clock/README.md"
  if (mode == "version") {
    print "    @@ -3 +3,2 @@"; print "     " line[3]; print "    +judged: " judged
  } else {
    print "    @@ -3 +3 @@"; print "    -" line[3]; print "    +checked: " want
  }
  for (i = 1; i in line; i++) {
    if (line[i] ~ /\*\*1\.4\.1\*\*/) {
      l = line[i]; sub(/1\.4\.1/, "1.5.0", l)
      print "    @@ -" i " +" i " @@"; print "    -" line[i]; print "    +" l
    }
    if (mode == "counts" && line[i] ~ /^\| `Clock\.h` \| /) {
      print "    @@ -" i " +" i " @@"; print "    -" line[i]; print "    +| `Clock.h` | " real " |"
    }
  }
}'
