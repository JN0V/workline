#!/bin/sh
# A cmd: developer in a review round: fixes #3's mul in lib.sh, says what
# it did, and, when the prompt gives the shared five whys skill after the
# task, the analysis as its own note.
prompt=$(cat)
cat <<'YAML'
- patch:
    file: lib.sh
    content: |
      add() { echo $(($1 + $2)); }
      mul() { echo $(($1 * $2)); }
- note: "mul multiplies now: the operator was +, copied from add."
YAML
case "$prompt" in
*"# Skill: five-whys"*)
	cat <<'YAML'
- note: |
    Five whys
    Problem: the reviewer still finds mul adding after two rounds.
    1. Why does mul still add? Each round patched the call, not the operator.
    2. Why the call? The finding names the line of the call.
    Root cause: the fix followed the symptom's line, not mul's body.
    Action (code): mul's body multiplies, its acceptance test checks 2 by 3.
YAML
	;;
esac
