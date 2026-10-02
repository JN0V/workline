#!/bin/sh
# A cmd: agent fixing line 10 of docs/tech/auth.md, in the `documented`
# fixture, whose first answer is YAML broken by an unescaped quote in a
# note, as Sonnet wrote on DomoticsCore (ADR-0014 step 4): `"1.4.1"` inside
# a double-quoted string. Asked again with the parse error, it answers the
# same, quoted right; asked again without it, the same broken answer.
prompt=$(cat)
old=$(printf '%s\n' "$prompt" | sed -n 's/^ *4 | checked: //p' | head -1)
new=$(printf '%s\n' "$prompt" | sed -n 's/.*sets `checked: \([0-9a-f]*\)`.*/\1/p' | head -1)
note='- note: "Line 3 says the fallback returns `"one hour"` when unset; I could not confirm it."'
if printf '%s\n' "$prompt" | grep -q '^# Your previous answer was refused' && printf '%s\n' "$prompt" | grep -q 'yaml: line'; then
	note="- note: 'Line 3 says the fallback returns \`\"one hour\"\` when unset; I could not confirm it.'"
fi
cat <<YAML
- patch: |
    --- a/docs/tech/auth.md
    +++ b/docs/tech/auth.md
    @@ -3,8 +3,8 @@
     sources: [src/auth/token.go]
    -checked: $old
    +checked: $new
     ---
     # Authentication
     
     ## Token refresh
     
    -Access tokens last one hour.
    +Access tokens last two hours.
- claim:
    doc: docs/tech/auth.md
    lines: "10"
    status: contradicted
    source:
      path: src/auth/token.go
      quote: "const TokenTTL = 7200"
$note
YAML
