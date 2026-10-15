#!/bin/sh
# A cmd: agent for the reviewer's cases, to prove their scoring without a
# real one (checks_test.go). A lens answers the file $1/<lens>.yaml, or
# nothing found; the lenses asked together answer each of theirs, each
# finding naming its lens; as the judge, it says no to a finding whose
# title holds REFUSE, quoting Checkout's first line (the coupon case's), yes
# to any other. A lens's call reports 100 tokens in
# and 10 out, a judge's 1000 and 100, so the test can tell whose tokens
# went where.
dir=$1
prompt=$(cat)
lenses=$(printf '%s\n' "$prompt" | sed -n 's/^# Lenses: //p' | head -n 1)
if [ -n "$lenses" ]; then
	printf 'model: fake-lens\ntokens-in: 100\ntokens-out: 10\n' > "$WORKLINE_CALL"
	answer=$(for lens in $(printf '%s' "$lenses" | tr ',' ' '); do
		if [ -f "$dir/$lens.yaml" ]; then sed "s/^- finding:\$/- finding:\n    lens: $lens/" "$dir/$lens.yaml"; fi
	done)
	if [ -n "$answer" ]; then printf '%s\n' "$answer"; else echo '[]'; fi
	exit 0
fi
lens=$(printf '%s\n' "$prompt" | sed -n 's/^# Lens: //p' | head -n 1)
if [ -n "$lens" ]; then
	printf 'model: fake-lens\ntokens-in: 100\ntokens-out: 10\n' > "$WORKLINE_CALL"
	if [ -f "$dir/$lens.yaml" ]; then cat "$dir/$lens.yaml"; else echo '[]'; fi
	exit 0
fi
printf 'model: fake-judge\ntokens-in: 1000\ntokens-out: 100\n' > "$WORKLINE_CALL"
case $prompt in
*REFUSE*) printf '%s\n' '- note: "no: the code handles it"' '- quote: {path: "shop/checkout.go", text: "func Checkout(items []Item, coupon string) (int, error) {"}' ;;
*) echo '- note: "yes: it fails as the finding says"' ;;
esac
