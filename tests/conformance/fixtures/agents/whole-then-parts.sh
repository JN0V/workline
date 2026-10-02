#!/bin/sh
# A cmd: agent answering both tasks of a gardening run on the `documented`
# fixture: the `suspect` task, confirming the first doc put before it
# (confirm-each-doc.sh); the parts of docs/tech/auth.md, the one holding
# token.go contradicting line 10; and the fix of what they found.
prompt=$(cat)
case "$prompt" in
*"Kind: part"*"## src/auth/token.go, lines"*)
	cat <<'YAML'
- claim:
    lines: "10"
    status: contradicted
    quote: "Access tokens last one hour."
    source: {path: src/auth/token.go, lines: "4", quote: "const TokenTTL = 7200"}
    why: "7200 seconds is two hours"
YAML
	;;
*"Kind: part"*) echo "[]" ;;
*"Kind: fix"*)
	cat <<'YAML'
- patch: |
    --- a/docs/tech/auth.md
    +++ b/docs/tech/auth.md
    @@ -8,3 +8,3 @@
     ## Token refresh
     
    -Access tokens last one hour.
    +Access tokens last two hours.
- claim:
    lines: "10"
    status: contradicted
    source: {path: src/auth/token.go, quote: "const TokenTTL = 7200"}
YAML
	;;
*) printf '%s\n' "$prompt" | sh "$(dirname "$0")/confirm-each-doc.sh" ;;
esac
