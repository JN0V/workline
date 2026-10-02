#!/bin/sh
# A cmd: agent judging two docs of the `documented` fixture, docs/tech/auth.md
# and docs/tech/session.md (sources: src/auth/session.go), as Sonnet answered
# on DomoticsCore (ADR-0014 step 4, HeapTracker's pitfall): it fixes auth.md's
# line 10 with a right claim that names no `doc`, and only records `judged`
# in session.md. With AMBIGUOUS set, it fixes session.md's line 10 too, both
# claims naming no `doc`, the one for auth.md quoting src/auth/token.go, a
# source of both docs then. It reads the commits from the prompt.
prompt=$(cat)
auth=$(printf '%s\n' "$prompt" | sed -n 's/^ *4 | checked: //p' | sed -n 1p)
session=$(printf '%s\n' "$prompt" | sed -n 's/^ *4 | checked: //p' | sed -n 2p)
new=$(printf '%s\n' "$prompt" | sed -n 's/.*sets `checked: \([0-9a-f]*\)`.*/\1/p' | head -1)
sources=$(printf '%s\n' "$prompt" | sed -n 's/^ *3 | sources: //p' | sed -n 2p)
cat <<YAML
- patch: |
    --- a/docs/tech/auth.md
    +++ b/docs/tech/auth.md
    @@ -3,3 +3,3 @@
     sources: [src/auth/token.go]
    -checked: $auth
    +checked: $new
     ---
    @@ -8,3 +8,3 @@
     ## Token refresh
     
    -Access tokens last one hour.
    +Access tokens last two hours.
- claim:
    lines: "10"
    status: contradicted
    quote: "Access tokens last one hour."
    source:
      path: src/auth/token.go
      quote: "const TokenTTL = 7200"
    why: "7200 seconds are two hours"
YAML
if [ -z "$AMBIGUOUS" ]; then
	cat <<YAML
- patch: |
    --- a/docs/tech/session.md
    +++ b/docs/tech/session.md
    @@ -3,3 +3,4 @@
     sources: $sources
     checked: $session
    +judged: $new
     ---
YAML
	exit 0
fi
cat <<YAML
- patch: |
    --- a/docs/tech/session.md
    +++ b/docs/tech/session.md
    @@ -3,3 +3,3 @@
     sources: $sources
    -checked: $session
    +checked: $new
     ---
    @@ -8,3 +8,3 @@
     ## Length
     
    -Sessions end after one hour.
    +Sessions end after two hours.
- claim:
    lines: "10"
    status: contradicted
    source:
      path: src/auth/session.go
      quote: "const SessionTTL = 7200"
YAML
