#!/bin/sh
# The weekly sample's auditor, on what the product owner read (ADR-0046):
# each act said wrong with words only the text or the code it read holds —
# found again only when the engine gives that text and that code.
p=$(cat)
case "$p" in
*"## The doc:"*)
	echo '- note: "true: every passage holds against the sources"' ;;
*'`ready` on #40'*)
	printf '%s\n' '- claim: {status: contradicted, source: {path: "#40", quote: "### Validation TBD"}, why: "No person wrote its Validation."}' \
		'- note: "false: it was not ready"' ;;
*'`sources` on #41'*)
	printf '%s\n' '- claim: {status: contradicted, source: {path: src/notes/n01.go, quote: "const N01 = 11"}, why: "The code named holds a note, no export."}' \
		'- note: "false: the code named is not what the issue is about"' ;;
*)
	echo '- note: "true: the act holds against its material"' ;;
esac
