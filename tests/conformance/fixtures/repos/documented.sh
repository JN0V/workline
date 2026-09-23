#!/bin/sh
# A project whose docs declare their sources: code -> technical doc -> product doc.
set -eu
# Hermetic: ignore the machine's git config and global hooks.
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
git init -q -b main .
git config user.name "Fixture"
git config user.email "fixture@example.invalid"
export GIT_AUTHOR_DATE="2026-01-01T00:00:00Z" GIT_COMMITTER_DATE="2026-01-01T00:00:00Z"
mkdir -p src/auth docs/tech docs/product
printf 'package auth\n\n// Tokens last one hour.\nconst TokenTTL = 3600\n' > src/auth/token.go
cat > docs/tech/auth.md <<'DOC'
---
type: reference
sources: [src/auth/token.go]
checked: HEAD
---
# Authentication

## Token refresh

Access tokens last one hour.
DOC
cat > docs/product/sign-in.md <<'DOC'
---
type: concept
sources: [docs/tech/auth.md#token-refresh]
checked: HEAD
---
# Staying signed in

You stay signed in for one hour without doing anything.
DOC
git add . && git commit -q -m "feat(auth): sign users in with one-hour tokens"
# Replace the placeholder with the real commit, as a person confirming the docs would.
C=$(git rev-parse --short HEAD)
sed -i "s/checked: HEAD/checked: $C/" docs/tech/auth.md docs/product/sign-in.md
git add . && git commit -q -m "docs: confirm the auth docs against the code"
