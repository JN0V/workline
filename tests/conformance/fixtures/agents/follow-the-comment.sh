#!/bin/sh
# A cmd: agent following a stale code comment over the code (workline #29),
# in the `documented` fixture whose comment was made to say "thirty minutes"
# while the setting still says 3600: it rewrites "one hour", citing the
# comment. Asked again after a refusal, $1 says what it does: `stubborn`
# answers the same; `confirms` vouches for the doc as it is, which is true.
# $2, when given, is the source the doc names and the comment is in, in
# place of src/auth/token.go.
src=${2:-src/auth/token.go}
prompt=$(cat)
old=$(printf '%s\n' "$prompt" | sed -n 's/^ *4 | checked: //p' | head -1)
new=$(printf '%s\n' "$prompt" | sed -n 's/.*sets `checked: \([0-9a-f]*\)`.*/\1/p' | head -1)
if [ "$1" = confirms ] && printf '%s\n' "$prompt" | grep -q 'Your previous answer was refused'; then
  cat <<YAML
- patch: |
    --- a/docs/tech/auth.md
    +++ b/docs/tech/auth.md
    @@ -3,3 +3,3 @@
     sources: [$src]
    -checked: $old
    +checked: $new
     ---
YAML
  exit 0
fi
cat <<YAML
- patch: |
    --- a/docs/tech/auth.md
    +++ b/docs/tech/auth.md
    @@ -3,3 +3,3 @@
     sources: [$src]
    -checked: $old
    +checked: $new
     ---
    @@ -8,3 +8,3 @@
     ## Token refresh
     
    -Access tokens last one hour.
    +Access tokens last thirty minutes.
- claim:
    lines: "10"
    status: contradicted
    quote: "Access tokens last one hour."
    source:
      path: $src
      quote: "Tokens last thirty minutes."
    why: "the code says thirty minutes"
YAML
