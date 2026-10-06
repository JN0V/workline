#!/bin/sh
# A cmd: judge answering only from what it is shown: yes when it reads the
# body of sum, which ranges over values[1:]; else no, for want of the code.
if grep -q 'range values\[1:\]'; then
	echo '- note: "yes: sum ranges over values[1:], leaving out the first value."'
else
	echo '- note: "no: the code that would show it, sum, is not given."'
fi
