#!/bin/sh
# A cmd: judge that, as the repository finder, cannot tell; as the judge
# of a finding, says yes.
case $(cat) in
*'**The repository.**'*) echo '- note: "unsure: whether a caller passes an empty list is in no code read."' ;;
*) echo '- note: "yes: Mean passes an empty list on to Average, which divides by its length."' ;;
esac
