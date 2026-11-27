#!/bin/sh
# A cmd: agent, which reads no file: it says in a note whether its prompt
# gives it a repository to read, a checkout of the commit judged.
prompt=$(cat)
case "$prompt" in
*'a checkout of the commit judged'*) echo '- note: a repository was offered to read' ;;
*) echo '- note: no repository to read was given' ;;
esac
