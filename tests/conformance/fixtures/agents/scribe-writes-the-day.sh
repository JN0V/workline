#!/bin/sh
# An agent that writes the day's line in the notes, whatever it is asked.
cat >/dev/null
cat <<'EOF'
- patch: |
    --- a/notes/today.md
    +++ b/notes/today.md
    @@ -1,2 +1,2 @@
     # Today
    -Nothing yet.
    +Open: Slow start
EOF
