#!/bin/sh
# The tested project, and the tester's Draft for its issue #3 (ADR-0055):
# the branch workline/tester/issue-3 holds its two tests, red on main, as
# `workline test write --issue 3` commits them. Checked out on it.
set -eu
sh "$(dirname "$0")/tested.sh"
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
export GIT_AUTHOR_DATE="2026-01-02T00:00:00Z" GIT_COMMITTER_DATE="2026-01-02T00:00:00Z"
git checkout -q -b workline/tester/issue-3
cat > tests/test_mul_two_by_three.sh <<'T'
# Scenario: Two by three
# Given the library, When I multiply 2 by 3, Then I get 6.
. ./lib.sh
[ "$(mul 2 3)" = 6 ]
T
cat > tests/test_mul_by_zero.sh <<'T'
# Scenario: By zero
# Given the library, When I multiply 5 by 0, Then I get 0.
. ./lib.sh
[ "$(mul 5 0)" = 0 ]
T
git add tests && git commit -q -m "test: the acceptance tests of #3" -m "Workline-Role: tester"
