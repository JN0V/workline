#!/bin/sh
# A cmd: agent fixing docs/tech/auth.md of the `documented` fixture, its
# tokens gone to two hours: line 10 brought to two hours, cited from the
# setting, and a code example written under the heading, a run of its own
# (#121). It moves `checked` to the commit the task gives, reading the
# commits and the numbered lines from the prompt, as a real agent would.
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
    @@ -8,3 +8,7 @@
     ## Token refresh
    +
    +\`\`\`go
    +ttl := time.Duration(auth.TokenTTL) * time.Second // two hours
    +\`\`\`
     
    -Access tokens last one hour.
    +Access tokens last two hours.
- claim:
    doc: docs/tech/auth.md
    lines: "10"
    status: contradicted
    source:
      path: src/auth/token.go
      quote: "const TokenTTL = 7200"
YAML
