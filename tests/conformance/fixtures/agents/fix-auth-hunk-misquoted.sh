#!/bin/sh
# A cmd: agent fixing line 10 of docs/tech/auth.md, in the `documented`
# fixture, cited from the setting, with a hunk quoting the doc wrong as
# Sonnet did on DomoticsCore (ADR-0014 step 4), so that git cannot apply it:
# MISQUOTE=blank skips the blank line between the heading and the sentence
# (index's footer); MISQUOTE=context ends on a context line the doc does not
# have (core's "`IIComponent` placeholder"). It moves `checked`, reading the
# commits from the prompt, as a real agent would.
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
case "$MISQUOTE" in
blank) cat <<'YAML'
    @@ -8,2 +8,2 @@
     ## Token refresh
    -Access tokens last one hour.
    +Access tokens last two hours.
YAML
;;
context) cat <<'YAML'
    @@ -8,4 +8,4 @@
     ## Token refresh
     
    -Access tokens last one hour.
    +Access tokens last two hours.
     ## Signing placeholder
YAML
;;
esac
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
