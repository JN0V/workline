#!/bin/sh
# A cmd: agent fixing docs/tech/auth.md of the `documented` fixture, to
# which a section was added, its line 14 "Tokens are signed with HS256.":
# it moves `checked`, brings line 10 to two hours, cited from the setting,
# and rewrites line 14 quoting it wrong ("RS256"), so that git cannot apply
# the patch whole, as agents misquote a line (DomoticsCore, ADR-0014 step 4).
# It reads the commits from the prompt, as a real agent would.
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
    @@ -13,2 +13,2 @@
     
    -Tokens are signed with RS256.
    +Tokens are signed with ES256.
- claim:
    lines: "10"
    status: contradicted
    quote: "Access tokens last one hour."
    source:
      path: src/auth/token.go
      quote: "const TokenTTL = 7200"
    why: "7200 seconds are two hours"
YAML
