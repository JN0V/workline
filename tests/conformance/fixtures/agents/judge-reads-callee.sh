#!/bin/sh
# A cmd: judge that may read (ADR-0045), answering in plain text as any
# agent does: shown the body of sum, which ranges over values[1:], it says
# yes; not shown it, it asks to read sum, by its name.
case $(cat) in
*'range values[1:]'*) echo '- note: "yes: sum ranges over values[1:], leaving out the first value."' ;;
*) echo '- read: {name: sum, why: "the cause calls sum, whose body is not shown"}' ;;
esac
