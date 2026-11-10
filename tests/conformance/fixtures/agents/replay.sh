#!/bin/sh
# A cmd: agent replaying recorded answers of a real run, as they came:
#
#   replay.sh <first answer> [<word> <next answer>]
#
# It answers the first file; asked again with a task holding <word> (the
# engine's feedback on the first answer), the next one. The files are
# answers recorded on the DomoticsCore trial (round 5), copied verbatim;
# an answer's <file>.covers, when there is one, is given after it: the
# covers entries the answer's format asks since (ADR-0065, change 4),
# which the recorded answers predate.
prompt=$(cat)
dir=$(dirname "$0")
answer=$1
if [ $# -ge 3 ] && printf '%s\n' "$prompt" | grep -qF -- "$2"; then
	answer=$3
fi
cat "$dir/$answer"
if [ -f "$dir/$answer.covers" ]; then
	echo
	cat "$dir/$answer.covers"
fi
