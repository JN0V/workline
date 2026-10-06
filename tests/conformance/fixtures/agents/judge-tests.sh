#!/bin/sh
# A cmd: judge of a tests-lens finding, answering from what it is shown:
# asked whether a test exercises the function the cause declares, it reads
# the tests shown for a call to it, and names the test that makes one.
prompt=$(cat)
case $prompt in
*"a test exercises"*) ;;
*) echo '- note: "no: the question does not ask whether a test exercises it."'; exit 0 ;;
esac
name=$(printf '%s\n' "$prompt" | sed -n '/^Its cause, quoted:/,/^## /s/^func \([A-Za-z]*\).*/\1/p' | head -1)
test=$(printf '%s\n' "$prompt" | sed -n '/^## The tests/,$p' |
	awk -v call="$name(" '/^func Test/ { t = $2; sub(/\(.*/, "", t); next } t != "" && index($0, call) { print t; exit }')
if [ -n "$test" ]; then
	echo "- note: \"no: $test calls $name and checks what it returns.\""
else
	echo "- note: \"yes: no test shown calls $name.\""
fi
