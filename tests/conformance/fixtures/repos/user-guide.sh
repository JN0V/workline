#!/bin/sh
# A user guide kept in its own repository, depending on the "documented" repository.
# The case that uses it builds "documented" first and passes its path as $API_REPO.
set -eu
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
git init -q -b main .
git config user.name "Fixture"
git config user.email "fixture@example.invalid"
export GIT_AUTHOR_DATE="2026-01-02T00:00:00Z" GIT_COMMITTER_DATE="2026-01-02T00:00:00Z"
API_HEAD=$(git -C "$API_REPO" rev-parse --short HEAD)
mkdir -p .workline guide
cat > .workline/config.yaml <<CFG
repos:
  api: {url: $API_REPO, branch: main}
CFG
cat > guide/sign-in.md <<DOC
---
type: concept
sources: [api:docs/tech/auth.md#token-refresh]
checked: {api: $API_HEAD}
---
# Staying signed in

You stay signed in for one hour without doing anything.
DOC
git add . && git commit -q -m "docs: explain how long users stay signed in"
