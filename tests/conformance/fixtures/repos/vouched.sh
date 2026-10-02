#!/bin/sh
# Docs the documentalist vouched for, week by week, for the weekly sample
# (ADR-0014, step 4). In ISO week 2026-W01 (29 December to 4 January) it
# moved `checked` on docs/tech/auth.md over a passage the code made false
# (tokens last two hours now), and a person moved it on docs/tech/people.md.
# In 2026-W02 it vouched for eleven notes, all true.
set -eu
# Hermetic: ignore the machine's git config and global hooks.
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
git init -q -b main .
git config user.name "Fixture"
git config user.email "fixture@example.invalid"
at() { export GIT_AUTHOR_DATE="$1" GIT_COMMITTER_DATE="$1"; }
bot() { git commit -q -m "docs: bring the docs in line with the code" -m "Workline-Role: documentalist
Workline-Model: claude:claude-sonnet-5"; }

at "2025-12-22T10:00:00Z"
mkdir -p src/auth src/notes docs/tech docs/notes
cat > src/auth/token.go <<'CODE'
package auth

// Tokens last one hour.
const TokenTTL = 3600

// Signed with RS256.
const Alg = "HS256"
CODE
cat > docs/tech/auth.md <<'DOC'
---
sources: [src/auth/token.go]
checked: HEAD
---
# Authentication

Access tokens last one hour.

Tokens are signed with HS256.
DOC
cat > docs/tech/people.md <<'DOC'
---
sources: [src/auth/token.go]
checked: HEAD
---
# Who signs tokens

The service signs them.
DOC
for n in 01 02 03 04 05 06 07 08 09 10 11; do
  printf 'package notes\n\nconst N%s = %s\n' "$n" "$n" > "src/notes/n$n.go"
  printf -- '---\nsources: [src/notes/n%s.go]\nchecked: HEAD\n---\n# Note %s\n\nN%s is %s.\n' "$n" "$n" "$n" "$n" > "docs/notes/n$n.md"
done
git add . && git commit -q -m "feat: start the project"
C=$(git rev-parse --short HEAD)
sed -i "s/checked: HEAD/checked: $C/" docs/tech/*.md docs/notes/*.md
git add . && git commit -q -m "docs: confirm the docs against the code"

at "2025-12-30T10:00:00Z"
sed -i 's/TokenTTL = 3600/TokenTTL = 7200/' src/auth/token.go
git commit -qam "feat(auth): tokens last two hours"
C=$(git rev-parse --short HEAD)

# The documentalist vouches for auth.md at that commit, its text unchanged.
at "2025-12-31T10:00:00Z"
sed -i "s/^checked: .*/checked: $C/" docs/tech/auth.md
sed -i "3a verified: agent:documentalist" docs/tech/auth.md
git add docs/tech/auth.md && bot

# A person moves people.md's: not the documentalist's to sample.
at "2026-01-02T10:00:00Z"
sed -i "s/^checked: .*/checked: $C/" docs/tech/people.md
sed -i "3a verified: human:fixture" docs/tech/people.md
git commit -qam "docs: confirm who signs tokens"

# 2026-W02: eleven notes vouched for, all true.
at "2026-01-06T10:00:00Z"
C=$(git rev-parse --short HEAD)
sed -i "s/^checked: .*/checked: $C/" docs/notes/*.md
git add docs/notes && bot
