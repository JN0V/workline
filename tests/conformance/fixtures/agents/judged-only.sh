#!/bin/sh
# A cmd: agent finding nothing wrong in docs/tech/auth.md of the `documented`
# fixture, yet not vouching for it: its patch only records `judged`, at the
# commit the task names, and a note says why. It reads the commits and the
# numbered lines from the prompt, as a real agent would.
prompt=$(cat)
old=$(printf '%s\n' "$prompt" | sed -n 's/^ *4 | checked: //p' | head -1)
at=$(printf '%s\n' "$prompt" | sed -n 's/.*`judged: \([0-9a-f]*\)`.*/\1/p' | head -1)
cat <<YAML
- patch: |
    --- a/docs/tech/auth.md
    +++ b/docs/tech/auth.md
    @@ -3,3 +3,4 @@
     sources: [src/auth/token.go]
     checked: $old
    +judged: $at
     ---
- note: "Every sentence I read holds; the refresh section names nothing the sources show either way."
YAML
