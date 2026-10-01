#!/bin/sh
# A cmd: agent removing the line of docs/tech/auth.md (`documented`, a line
# naming `RefreshToken` added at line 12) that names a function, citing it
# as gone from the code (ADR-0014, step 2); it leaves `checked` and records
# `judged`, as it read no more than that.
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
    @@ -10,3 +11,1 @@
     Access tokens last one hour.
    -
    -Call \`RefreshToken\` to renew a token.
- claim:
    lines: "12"
    status: gone
    name: RefreshToken
YAML
