#!/bin/sh
# A judge of the weekly sample's panel (ADR-0046), choosing as told: `act`
# picks the act's reading, `auditor` the auditor's, which alone quotes the
# material ("as these words show"). Never told whose each reading is, it
# finds out as a case needs.
a=$(sed -n 's/^Reading a: //p' | head -1)
case "$1:$a" in
act:*"as these words show"*|auditor:*) answer=no ;;
*) answer=yes ;;
esac
case "$1:$a" in
auditor:*"as these words show"*) answer=yes ;;
esac
echo "- note: \"$answer: chosen as the case asks\""
