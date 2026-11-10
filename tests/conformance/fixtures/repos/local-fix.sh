#!/bin/sh
# The tested project on the local forge (`forge: local`, #1322): issue #3,
# a bug; the tester's Draft of it, !1, on workline/tester/issue-3; the
# developer's fix, !2, on workline/developer/3, its record and the
# reviewer's verdict on its head written as the roles write them. The
# line's pass on that head is in .git/verdict.json, as `workline route
# --verdict` writes it. main checked out, clean.
set -eu
sh "$(dirname "$0")/tested-draft.sh"
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
export GIT_AUTHOR_DATE="2026-01-03T00:00:00Z" GIT_COMMITTER_DATE="2026-01-03T00:00:00Z"
git checkout -q -b workline/developer/3
printf 'add() { echo $(($1 + $2)); }\nmul() { echo $(($1 * $2)); }\n' > lib.sh
git commit -q -am "fix: mul for #3" -m "Workline-Role: developer"
git checkout -q main
from=$(git rev-parse workline/tester/issue-3)
head=$(git rev-parse workline/developer/3)
mkdir -p .git/workline/issues .git/workline/merge-requests
cat > .git/workline/issues/3.md <<EOF
---
title: mul is missing
state: open
labels: [bug, 'workline:developing']
---

## Need

\`mul 2 3\` fails: there is no mul.
EOF
cat > .git/workline/merge-requests/1.md <<EOF
---
title: 'test: the acceptance tests of #3'
state: open
labels: []
branch: workline/tester/issue-3
base: main
draft: true
---

The tester's acceptance tests of #3.
EOF
cat > .git/workline/merge-requests/2.md <<EOF
---
title: 'fix: mul for #3'
state: open
labels: []
branch: workline/developer/3
base: main
---

Closes #3

<!-- workline-comment -->
**The developer**'s record of this pull request, for its next runs and the auditor.

<!-- workline:developer-act {"act":"code","issue":3,"from":"$from","acceptance":["tests/test_mul_two_by_three.sh","tests/test_mul_by_zero.sh"],"code":["lib.sh"],"proof":{"result":"green","passed":3}} -->

<!-- workline:sticky=developer/code -->

<!-- workline-comment -->
**workline reviewer** — 0 findings on the change.

No finding on the commits reviewed.

<!-- workline:at head=$head -->
<!-- workline:record reviewed=$head runs=1 read=all -->

<!-- workline:sticky=reviewer/summary -->
EOF
printf '{"head": "%s", "status": "pass", "steps": [{"name": "reviewer", "status": "pass"}]}\n' "$head" > .git/verdict.json
