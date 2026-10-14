#!/bin/sh
# The weekly sample's auditor, as a judge reading each act's material would
# answer it (ADR-0046): every act planted wrong is found false, quoting the
# material; a doc vouched for stands. A finding is asked blind: should the
# judge's answer, or whether it kept the finding, reach the prompt, the
# answer is refused (`leaked`), and the case fails.
#
# AUDIT=agree answers each act as standing, a blind finding as its own
# reading of the code says: the dropped one does not fail, the kept one
# does. AUDIT=misquote answers each act wrong with quotes found nowhere, or
# only in a comment of the code. AUDIT_TOKENS says the tokens each call
# used, as an agent reports them.
p=$(cat)
[ -z "${AUDIT_TOKENS:-}" ] || printf 'tokens-in: %s\ntokens-out: 100\n' "$AUDIT_TOKENS" > "$WORKLINE_CALL"
case "$p" in
*"no N12 is declared"*|*finding-judged-no*|*"(verified"*|*"finding kept"*|*"finding dropped"*)
	echo '- note: "leaked: the judge'"'"'s answer reached the auditor"'; exit 0 ;;
esac
mode=${AUDIT:-wrong}
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
