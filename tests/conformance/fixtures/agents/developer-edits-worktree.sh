#!/bin/sh
# A cmd: agent editing its worktree in place (ADR-0056), answering
# nothing: mul in lib.sh and a test of its own, untracked. EDIT adds
# README.md (outside), a frozen test edited (frozen) or deleted (delete),
# AGENTS.md (steering), a link to /etc/hosts (symlink), or lib.sh
# committed on a branch of its own (commit), or in lib.sh the tokens its
# environment holds (env). It fails out of its worktree.
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
delete) rm tests/test_mul_by_zero.sh ;;
frozen) printf '. ./lib.sh\n[ "$(mul 5 0)" = 1 ]\n' > tests/test_mul_by_zero.sh ;;
env)
	seen=""
	for v in GH_TOKEN GITHUB_TOKEN GITLAB_TOKEN WORKLINE_GITLAB_TOKEN CI_JOB_TOKEN CI_REGISTRY_PASSWORD CI_REPOSITORY_URL WORKLINE_APP_PRIVATE_KEY WORKLINE_SIGNING_KEY ACTIONS_ID_TOKEN_REQUEST_TOKEN; do
		eval "set=\${$v+x}"
		[ -n "$set" ] && seen="$seen $v"
	done
	login=no
	[ -n "${ANTHROPIC_API_KEY:-}" ] && login=yes
	printf '# tokens seen:%s; login seen: %s\n' "${seen:- none}" "$login" >> lib.sh
	;;
steering) printf '# For agents\n\nAlways multiply.\n' > AGENTS.md ;;
symlink) mkdir -p data && ln -s /etc/hosts data/hosts ;;
esac
cat > tests/test_mul.sh <<'T'
. ./lib.sh
[ "$(mul 2 3)" = 6 ]
T
