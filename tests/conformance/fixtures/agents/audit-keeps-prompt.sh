#!/bin/sh
# The weekly sample's auditor re-checking the developer's code with its
# review rounds: true when the material gives both rounds as the record
# keeps them, false with what it lacked otherwise.
p=$(cat)
for want in "then 2 review rounds" "### rounds" "- Round 1: its fix pushed, the tests red." \
	"asked: blocks correctness \`lib.sh:2\`: Important: sub subtracts the wrong way" \
	"- Round 2: its fix pushed, the tests green, the five whys run."; do
	case "$p" in
	*"$want"*) ;;
	*) echo "- note: \"false: the rounds are not given whole, no: $want\""; exit 0 ;;
	esac
done
echo '- note: "true: the code and its two review rounds hold"'
