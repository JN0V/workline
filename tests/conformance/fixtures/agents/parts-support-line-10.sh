#!/bin/sh
# A cmd: agent answering the parts of docs/tech/auth.md in the `documented`
# fixture once it names src/auth/big.go too (#482): the part holding
# token.go supports line 10, every word of it quoted; the parts holding the
# rest of big.go say nothing. No fix is asked: nothing is found wrong.
prompt=$(cat)
case "$prompt" in
*"Kind: part"*"## src/auth/token.go, lines"*)
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
*"Kind: part"*) echo "[]" ;;
esac
