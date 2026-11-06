#!/bin/sh
# A cmd: agent replaying recorded answers of a real run, as they came:
#
#   replay.sh <first answer> [<word> <next answer>]
#
# It answers the first file; asked again with a task holding <word> (the
# engine's feedback on the first answer), the next one. The files are
# answers recorded on the DomoticsCore trial (round 5), copied verbatim.
prompt=$(cat)
dir=$(dirname "$0")
if [ $# -ge 3 ] && printf '%s\n' "$prompt" | grep -qF -- "$2"; then
	cat "$dir/$3"
else
	cat "$dir/$1"
fi
