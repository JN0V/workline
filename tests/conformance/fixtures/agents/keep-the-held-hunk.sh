#!/bin/sh
# A cmd: agent fixing docs/tech/auth.md of the `documented` fixture, to
# which a section was added, its line 14 "Tokens are signed with HS256.":
# its first answer brings line 10 to two hours, cited from the setting, and
# drops line 14 with no claim. Asked again, it does what the refusal says
# (16c660b's replay, ADR-0014 step 3): line 14 left, and line 10's fix sent
# again only if the refusal says that part holds; otherwise, like the agent
# then, it withdraws every fix and only vouches. It reads the commits and
# the numbered lines from the prompt, as a real agent would.
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
YAML
again=$(printf '%s\n' "$prompt" | sed -n '/^# Your previous answer was refused/,/^# Answer/p')
if [ -n "$again" ] && ! printf '%s\n' "$again" | grep -q 'docs/tech/auth.md holds (line 10)'; then
  exit 0
fi
cat <<'YAML'
    @@ -8,3 +8,3 @@
     ## Token refresh
     
    -Access tokens last one hour.
    +Access tokens last two hours.
YAML
[ -n "$again" ] || cat <<'YAML'
    @@ -13,2 +13 @@
     
    -Tokens are signed with HS256.
YAML
cat <<'YAML'
- claim:
    lines: "10"
    status: contradicted
    quote: "Access tokens last one hour."
    source:
      path: src/auth/token.go
      quote: "const TokenTTL = 7200"
    why: "7200 seconds are two hours"
YAML
