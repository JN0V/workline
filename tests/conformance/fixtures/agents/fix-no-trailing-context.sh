#!/bin/sh
# A cmd: agent fixing docs/tech/auth.md of the `documented` fixture, its last
# hunk ending on the changed line, with no context after it, though the doc
# goes on: agents write hunks so, and the engine applies them.
prompt=$(cat)
old=$(printf '%s\n' "$prompt" | sed -n 's/^ *4 | checked: //p' | head -1)
new=$(printf '%s\n' "$prompt" | sed -n 's/.*sets `checked: \([0-9a-f]*\)`.*/\1/p' | head -1)
cat <<YAML
- patch: |
    --- a/docs/tech/auth.md
    +++ b/docs/tech/auth.md
    @@ -3,3 +3,3 @@
     sources: [src/auth/token.go]
    -checked: $old
    +checked: $new
     ---
    @@ -8,3 +8,3 @@
     ## Token refresh
     
    -Access tokens last one hour.
    +Access tokens last two hours.
YAML
