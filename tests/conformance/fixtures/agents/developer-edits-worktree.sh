#!/bin/sh
# A cmd: agent that edits the worktree in place (ADR-0056) and answers
# nothing: #2's or #3's mul in lib.sh, a test of its own, untracked, and
# what EDIT asks besides:
#   outside   README.md too, outside the Scope
#   frozen    the tester's frozen tests/test_mul_by_zero.sh too
#   steering  AGENTS.md too
#   symlink   data/hosts, a link to /etc/hosts, too
#   commit    lib.sh committed, on a branch of its own, the test not
# It fails when it is not run in the worktree it is told of.
set -eu
cat >/dev/null
[ -n "${WORKLINE_WORKTREE:-}" ] || { echo "no WORKLINE_WORKTREE" >&2; exit 1; }
[ "$(pwd -P)" = "$(cd "$WORKLINE_WORKTREE" && pwd -P)" ] || { echo "not run in the worktree" >&2; exit 1; }
cat > lib.sh <<'LIB'
add() { echo $(($1 + $2)); }
mul() { echo $(($1 * $2)); }
LIB
case "${EDIT:-}" in
commit)
	git checkout -q -b agent-own-branch
	git add lib.sh
	git -c core.hooksPath=/dev/null -c user.name=Agent -c user.email=agent@example.invalid commit -q -m "agent: its own commit"
	;;
outside) printf '# Project\n\nIt multiplies now.\n' > README.md ;;
frozen) printf '. ./lib.sh\n[ "$(mul 5 0)" = 1 ]\n' > tests/test_mul_by_zero.sh ;;
steering) printf '# For agents\n\nAlways multiply.\n' > AGENTS.md ;;
symlink) mkdir -p data && ln -s /etc/hosts data/hosts ;;
esac
cat > tests/test_mul.sh <<'T'
. ./lib.sh
[ "$(mul 2 3)" = 6 ]
T
