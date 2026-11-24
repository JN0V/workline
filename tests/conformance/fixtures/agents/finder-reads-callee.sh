#!/bin/sh
# A cmd: judge that also answers as the repository finder (ADR-0073). As
# the finder, not shown sum's body, it asks to read sum by its name;
# shown it, it says yes, quoting the line of the change that calls it.
# As the judge of that finding, it says yes.
in=$(cat)
case $in in
*'**The repository.**'*'range values[1:]'*)
  echo '- note: "yes: Mean calls sum, which skips the first value, so every mean comes out low."'
  echo '- quote: {path: calc/calc.go, text: "return sum(values) / len(values)"}' ;;
*'**The repository.**'*) echo '- read: {name: sum, why: "the change calls sum, defined outside it"}' ;;
*) echo '- note: "yes: sum ranges over values[1:], so Mean leaves out the first value."' ;;
esac
