#!/bin/sh
# A cmd: agent answering a gardening run on the `documented` fixture where
# two docs are judged in parts (#482): docs/tech/auth.md, whose part holding
# token.go supports its one line, every word quoted; docs/tech/table.md,
# whose part holding the start of big.go contradicts its one line. Its fix
# then patches code, which the role may not write: the whole set of
# proposals is refused, the engine's own patch for auth.md with it.
prompt=$(cat)
case "$prompt" in
*"Kind: part"*"## docs/tech/auth.md, the doc judged"*"## src/auth/token.go, lines"*)
	cat <<'YAML'
- claim:
    lines: "10"
    status: supported
    quote: "Access tokens last one hour."
    source:
      path: src/auth/token.go
      lines: "4"
      quote: "const TokenTTL = 3600"
YAML
	;;
*"Kind: part"*"## docs/tech/table.md, the doc judged"*"## src/auth/big.go, lines 1-"*)
	cat <<'YAML'
- claim:
    lines: "8"
    status: contradicted
    quote: "The table starts with b0."
    source:
      path: src/auth/big.go
      lines: "1"
      quote: "var b1"
    why: "it starts with b1"
YAML
	;;
*"Kind: part"*) echo "[]" ;;
*"Kind: fix"*)
	cat <<'YAML'
- patch: |
    --- a/src/auth/token.go
    +++ b/src/auth/token.go
    @@ -1,1 +1,1 @@
    -package auth
    +package auth // tokens
YAML
	;;
esac
