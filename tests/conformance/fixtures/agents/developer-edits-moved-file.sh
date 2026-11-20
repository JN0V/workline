#!/bin/sh
# A cmd: agent editing, in its worktree, a file main moved after the
# tester's Draft was pushed (#1941): mul added to src/lib.sh, where
# main's lib.sh now sources add from.
set -eu
cat >/dev/null
echo 'mul() { echo $(($1 * $2)); }' >> src/lib.sh
