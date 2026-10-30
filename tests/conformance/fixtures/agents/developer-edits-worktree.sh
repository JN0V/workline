#!/bin/sh
# A cmd: agent editing its worktree in place (ADR-0056), answering
# nothing: mul in lib.sh and a test of its own, untracked. EDIT adds
# README.md (outside), a frozen test edited (frozen) or deleted (delete),
# AGENTS.md (steering), a link to /etc/hosts (symlink), or lib.sh
# committed on a branch of its own (commit), or in lib.sh the tokens its
# environment holds (env), or the credential git, its ssh agent and gh
# hand it (reach); or it makes its own run green where the code
# is not: mul in an ignored build file lib.sh reads (cache), or a wrong
# mul and the start commit replaced by one whose test runner passes all,
# the runner put back after its run (replace). Its caps (ADR-0056): it
# says it used every turn WORKLINE_MAX_TURNS allows (turns), or 5000
# tokens (tokens), or it works on past its time, a child of its own
# set to write "late" in lib.sh (slow); or it fails once it edited
# (fail). It fails out of its worktree, and where its own run is red.
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
		eval "set=\${$v+x} val=\${$v-}"
		[ -n "$set" ] && [ "$val" != workline-sealed ] && seen="$seen $v"
	done
	login=no
	[ -n "${ANTHROPIC_API_KEY:-}" ] && login=yes
	printf '# tokens seen:%s; login seen: %s\n' "${seen:- none}" "$login" >> lib.sh
	;;
reach)
	got=$(printf 'protocol=https\nhost=github.com\n\n' | git credential fill 2>/dev/null | sed -n 's/^password=//p') || true
	gh=empty
	[ -z "$(ls -A "${GH_CONFIG_DIR:?}")" ] || gh=full
	printf '# credential: %s; ssh agent: %s; gh config: %s\n' "${got:-none}" "${SSH_AUTH_SOCK:-none}" "$gh" >> lib.sh
	;;
cache)
	cat > lib.sh <<'LIB'
add() { echo $(($1 + $2)); }
[ -f build/mul.sh ] && . ./build/mul.sh
LIB
	mkdir -p build
	echo 'mul() { echo $(($1 * $2)); }' > build/mul.sh
	echo 'build/' >> "$(git rev-parse --git-path info/exclude)"
	;;
replace)
	printf 'add() { echo $(($1 + $2)); }\nmul() { echo 7; }\n' > lib.sh
	start=$(git rev-parse HEAD)
	runner=$(mktemp)
	cp run-tests.sh "$runner"
	cat > run-tests.sh <<'RUN'
#!/bin/sh
report=$1
mkdir -p "$(dirname "$report")"
{
  echo '<?xml version="1.0" encoding="UTF-8"?>'
  echo '<testsuites><testsuite name="tests">'
  for t in tests/test_*.sh; do
    echo "<testcase classname=\"tests\" name=\"$(basename "$t" .sh)\" time=\"0.01\"/>"
  done
  echo '</testsuite></testsuites>'
} > "$report"
RUN
	git add run-tests.sh
	fake=$(git -c user.name=Agent -c user.email=agent@example.invalid commit-tree "$(git write-tree)" -m "agent: a runner that passes all")
	git replace -f "$start" "$fake"
	git reset -q
	;;
steering) printf '# For agents\n\nAlways multiply.\n' > AGENTS.md ;;
symlink) mkdir -p data && ln -s /etc/hosts data/hosts ;;
esac
cat > tests/test_mul.sh <<'T'
. ./lib.sh
[ "$(mul 2 3)" = 6 ]
T
case "${EDIT:-}" in
cache | replace) # its own run, in its worktree: green, or it stops
	sh run-tests.sh "$WORKLINE_WORKTREE/.own-run/report.xml" >/dev/null || { echo "its own run is red" >&2; exit 1; }
	rm -rf .own-run
	[ -z "${runner:-}" ] || { cp "$runner" run-tests.sh && rm -f "$runner"; }
	;;
turns)
	[ -n "${WORKLINE_MAX_TURNS:-}" ] || { echo "no WORKLINE_MAX_TURNS" >&2; exit 1; }
	printf 'model: looping\nturns: %s\n' "$WORKLINE_MAX_TURNS" > "$WORKLINE_CALL"
	;;
fail) echo "it gave up" >&2; exit 3 ;;
tokens) printf 'model: costly\ntokens-in: 4000\ntokens-out: 1000\n' > "$WORKLINE_CALL" ;;
slow)
	[ -n "${WORKLINE_MAX_SECONDS:-}" ] || { echo "no WORKLINE_MAX_SECONDS" >&2; exit 1; }
	(sleep 4 && echo '# late' >> lib.sh) &
	sleep 30
	;;
esac
