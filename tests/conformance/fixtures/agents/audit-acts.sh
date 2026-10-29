#!/bin/sh
# The weekly sample's auditor, planted (ADR-0046): each wrong act found
# false with the material's words, a doc standing; `leaked` if a finding's
# judge reached the prompt. AUDIT=agree: each act stands. AUDIT=misquote:
# each said wrong with quotes found nowhere, or in a comment only.
# AUDIT_TOKENS: the tokens each call says it used.
p=$(cat)
[ -z "${AUDIT_TOKENS:-}" ] || printf 'tokens-in: %s\ntokens-out: 100\n' "$AUDIT_TOKENS" > "$WORKLINE_CALL"
case "$p" in
*"no N12 is declared"*|*finding-judged-no*|*"(verified"*|*"finding kept"*|*"finding dropped"*)
	echo '- note: "leaked: the judge'"'"'s answer reached the auditor"'; exit 0 ;;
esac
mode=${AUDIT:-wrong}
if [ "$mode" = defects ]; then # AUDIT=defects: each "Enn overflows" true, quoting its line
	n=$(printf '%s\n' "$p" | grep -o 'E[0-9][0-9] overflows' | head -1 | cut -c2-3)
	if [ -n "$n" ]; then
		printf '%s\n' "- claim: {status: contradicted, source: {path: src/notes/extra.go, quote: \"const E$n = $(expr "$n" + 0)\"}, why: \"It overflows.\"}" \
			'- note: "true: the code fails as the finding says"'
		exit 0
	fi
fi
case "$mode:$p" in
*"## The doc:"*)
	echo '- note: "true: every passage holds against the sources"' ;;
agree:*"Extra shadows N12"*)
	printf '%s\n' '- claim: {status: contradicted, source: {path: src/notes/extra.go, quote: "const Extra = 12"}, why: "Extra is declared once, shadowing nothing."}' \
		'- note: "false: the code does not fail as the finding says"' ;;
agree:*"Extra is not a note"*)
	printf '%s\n' '- claim: {status: contradicted, source: {path: src/notes/extra.go, quote: "const Extra = 12"}, why: "Extra is twelve, a note number."}' \
		'- note: "true: the code fails as the finding says"' ;;
agree:*)
	echo '- note: "true: the act holds against its material"' ;;
misquote:*"Extra is not a note"*)
	printf '%s\n' '- claim: {status: contradicted, source: {path: src/notes/extra.go, quote: "Extra is never a number"}, why: "said so."}' \
		'- note: "false: the code handles it"' ;;
misquote:*)
	printf '%s\n' '- claim: {status: contradicted, source: {path: "#20", quote: "the reporter withdrew it"}, why: "said so."}' \
		'- note: "false: it was not ready"' ;;
*"Extra is not a note"*)
	printf '%s\n' '- claim: {status: contradicted, source: {path: src/notes/extra.go, quote: "const Extra = 12"}, why: "Extra is a constant of its own."}' \
		'- note: "false: the code does not fail as the finding says"' ;;
*"Extra shadows N12"*)
	printf '%s\n' '- claim: {status: contradicted, source: {path: src/notes/extra.go, quote: "const Extra = 12"}, why: "Extra is declared beside the notes it shadows."}' \
		'- note: "true: the code fails as the finding says"' ;;
*"The message of commit"*)
	printf '%s\n' '- claim: {status: contradicted, quote: "and drop N01", source: {path: diff, quote: "+const Extra = 12"}, why: "The diff adds Extra and drops nothing."}' \
		'- note: "false: the message says more than the commit does"' ;;
*"The tests the tester wrote"*"### lib.sh (base)"*)
	printf '%s\n' '- claim: {status: contradicted, source: {path: tests/test_sub.sh, quote: "sub 3 5"}, why: "It subtracts 5 from 3."}' \
		'- claim: {status: contradicted, source: {path: scenarios, quote: "When I subtract 3 from 5, Then I get 2."}, why: "The scenario subtracts 3 from 5."}' \
		'- note: "false: the test checks another subtraction"' ;;
*"The tester kept its test"*)
	printf '%s\n' '- claim: {status: contradicted, source: {path: tests/test_sub.sh, quote: "= -2 ]"}, why: "The test wants -2, the scenario 2."}' \
		'- note: "false: the test does not check the scenario"' ;;
*"The tester rewrote its test"*"### before"*"### after"*)
	printf '%s\n' '- claim: {status: contradicted, source: {path: after, quote: "-ge 0 ]"}, why: "Any value not below zero passes."}' \
		'- claim: {status: contradicted, source: {path: scenarios, quote: "Then I get 2."}, why: "The scenario wants 2."}' \
		'- note: "false: the rewrite loosens the Then"' ;;
*"The tester's curator removed"*)
	printf '%s\n' '- claim: {status: contradicted, source: {path: diff, quote: "add 1 1"}, why: "1 + 1 is the only sum of equal numbers checked."}' \
		'- note: "false: the removed test checks what none kept does"' ;;
*"The developer's code for #3"*"### diff"*"### proof"*)
	printf '%s\n' '- claim: {status: contradicted, source: {path: diff, quote: "$(($2 - $1))"}, why: "It subtracts the first from the second."}' \
		'- note: "false: the code gives -2 for 5 minus 3"' ;;
*'`ready` on #20'*)
	printf '%s\n' '- claim: {status: contradicted, source: {path: "#20", quote: "### Validation TBD"}, why: "No person wrote its Validation."}' \
		'- note: "false: it was not ready"' ;;
*'`refine` on #21'*)
	printf '%s\n' '- claim: {status: contradicted, source: {path: "#21", quote: "as CSV and as JSON"}, why: "The section adds JSON."}' \
		'- claim: {status: contradicted, source: {path: "#21 comment 1", quote: "CSV only, never JSON."}, why: "The person asked for CSV only."}' \
		'- note: "false: the sections say more than the person asked"' ;;
*'`sources` on #22'*)
	printf '%s\n' '- claim: {status: contradicted, source: {path: src/notes/n01.go, quote: "const N01 = 01"}, why: "The code named holds a note, no export."}' \
		'- note: "false: the code named is not what the issue is about"' ;;
*'`order` on #23'*)
	printf '%s\n' '- claim: {status: contradicted, source: {path: "#23", quote: "Customers lose rows every night"}, why: "Urgent, set last."}' \
		'- note: "false: the issue contradicts the priority set"' ;;
*'`ask` on #24'*)
	printf '%s\n' '- claim: {status: contradicted, source: {path: "#24 comment 1", quote: "The format is CSV."}, why: "Answered before it asked."}' \
		'- note: "false: it was answered already"' ;;
*'`close-obsolete` on #25'*)
	printf '%s\n' '- claim: {status: contradicted, source: {path: "#25", quote: "Export the notes as XML."}, why: "The issue asks XML."}' \
		'- claim: {status: contradicted, source: {path: src/notes/n02.go, quote: "const N02 = 02"}, why: "The code holds no XML."}' \
		'- note: "false: the code does not do what the issue asks"' ;;
*'`split` on #26'*)
	printf '%s\n' '- claim: {status: contradicted, source: {path: "#26", quote: "send the CSV by mail"}, why: "No part holds it."}' \
		'- note: "false: a need of the parent is in no part"' ;;
*)
	echo '- note: "true: the act holds against its material"' ;;
esac
