#!/bin/sh
# A cmd: agent for the reviewer's cases, to prove their scoring without a
# real one (checks_test.go). A lens answers the file $1/<lens>.yaml, or
# nothing found; as the judge, it says no to a finding whose title holds
# REFUSE, yes to any other. A lens's call reports 100 tokens in and 10 out,
# a judge's 1000 and 100, so the test can tell whose tokens went where.
dir=$1
prompt=$(cat)
lens=$(printf '%s\n' "$prompt" | sed -n 's/^# Lens: //p' | head -n 1)
if [ -n "$lens" ]; then
	printf 'model: fake-lens\ntokens-in: 100\ntokens-out: 10\n' > "$WORKLINE_CALL"
	if [ -f "$dir/$lens.yaml" ]; then cat "$dir/$lens.yaml"; else echo '[]'; fi
	exit 0
fi
printf 'model: fake-judge\ntokens-in: 1000\ntokens-out: 100\n' > "$WORKLINE_CALL"
case $prompt in
*REFUSE*) echo '- note: "no: the code handles it"' ;;
*) echo '- note: "yes: it fails as the finding says"' ;;
esac
