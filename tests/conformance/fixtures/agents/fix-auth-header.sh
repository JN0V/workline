#!/bin/sh
# A cmd: agent fixing docs/tech/auth.md of the `documented` fixture, its
# header written as $1 says: `moves-checked` vouches for the doc, moving
# `checked` to the commit the task names (as `checked` or `judged`, whichever
# it offers); `sets-judged` leaves `checked` and records `judged`. $2 says
# why "one hour" goes (ADR-0014, step 2): `uncited`, no claim; `cites-code`,
# the setting; `cites-elsewhere`, a doc that is not a source. It reads the
# commits and the numbered lines from the prompt, as a real agent would.
prompt=$(cat)
old=$(printf '%s\n' "$prompt" | sed -n 's/^ *4 | checked: //p' | head -1)
new=$(printf '%s\n' "$prompt" | sed -n 's/.*sets `checked: \([0-9a-f]*\)`.*/\1/p' | head -1)
at=$(printf '%s\n' "$prompt" | sed -n 's/.*`judged: \([0-9a-f]*\)`.*/\1/p' | head -1)
[ -n "$new" ] || new=$at
if [ "$1" = sets-judged ]; then
  header="     checked: $old
    +judged: $at"
  count="@@ -3,3 +3,4 @@"
else
  header="    -checked: $old
    +checked: $new"
  count="@@ -3,3 +3,3 @@"
fi
cat <<YAML
- patch: |
    --- a/docs/tech/auth.md
    +++ b/docs/tech/auth.md
    $count
     sources: [src/auth/token.go]
$header
     ---
    @@ -8,3 +8,3 @@
     ## Token refresh
     
    -Access tokens last one hour.
    +Access tokens last two hours.
YAML
case $2 in
cites-code) cat <<'YAML'
- claim:
    lines: "10"
    status: contradicted
    quote: "Access tokens last one hour."
    source:
      path: src/auth/token.go
      quote: "const TokenTTL = 7200"
    why: "7200 seconds are two hours"
YAML
;;
cites-elsewhere) cat <<'YAML'
- claim:
    lines: "10"
    status: contradicted
    source:
      path: docs/product/sign-in.md
      quote: "You stay signed in"
YAML
;;
esac
