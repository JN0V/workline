#!/bin/sh
# A cmd: agent answering the documentalist's parts in YAML one item of which
# is malformed, as Sonnet once wrote (a brace too many): the part holding
# token.go gets one good claim and one broken line; the other, nothing; the
# fix call, the fix.
prompt=$(cat)
case "$prompt" in
*"Kind: part"*"## src/auth/token.go, lines"*)
	cat <<'YAML'
- claim: {lines: "10", status: contradicted, quote: "Access tokens last one hour.", source: {path: src/auth/token.go, lines: "4", quote: "const TokenTTL = 7200"}, why: "7200 seconds is two hours"}
- claim: {lines: "6", status: supported, quote: "# Authentication", source: {path: src/auth/token.go, lines: "1", quote: "package auth"}, why: "a brace too many"}}
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
YAML
	;;
esac
