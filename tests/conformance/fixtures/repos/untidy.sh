#!/bin/sh
# A project whose docs have drifted in every way the documentalist's hygiene
# checks look for. Budgets are set low so each problem takes a few lines.
set -eu
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
git init -q -b main .
git config user.name "Fixture"
git config user.email "fixture@example.invalid"
export GIT_AUTHOR_DATE="2026-01-01T00:00:00Z" GIT_COMMITTER_DATE="2026-01-01T00:00:00Z"
mkdir -p .workline src docs/cards
cat > .workline/config.yaml <<'CFG'
roles:
  documentalist:
    settings:
      budgets:
        doc-lines: 10
        section-words: 30
        card-words: {min: 8, max: 40}
        folder-lines: 30
        root-agent-file-lines: 5
CFG
cat > src/token.go <<'GO'
package auth

// RefreshToken renews a token before it expires.
func RefreshToken(t string) string { return t }

// Revoke ends a session.
func Revoke(t string) {}
GO
cat > docs/auth.md <<'DOC'
# Authentication

Call `RefreshToken` before a token expires, and `Revoke` to sign a user out.
See [sessions](sessions.md#how-long-a-session-lasts) and [the API](api.md).
Also [a missing part](sessions.md#no-such-part) and [the project site](https://example.com/workline).
DOC
cat > docs/sessions.md <<'DOC'
# Sessions

## How long a session lasts

A session lasts as long as its token, and a token lasts one hour unless it is
refreshed. When it expires, the user is asked to sign in again, and anything
they had not saved is kept in the browser until they come back to the page.

## Signing out

Signing out ends the session.
DOC
cat > docs/faq.md <<'DOC'
# Questions

## How long am I signed in?

A session lasts as long as its token, and a token lasts one hour unless it is
refreshed. When it expires, the user is asked to sign in again, and anything
they had not saved is kept in the browser until they come back to the page.
DOC
cat > docs/history.md <<'DOC'
# History

## The first year

The project started as a small tool to check commit messages. Then it learned
to release, then to keep documentation true to the code, and each time the new
job was written down as a role before any code was written for it, so that the
contract came first and the engine followed, one case at a time, with tests.
DOC
cat > docs/cards/token.md <<'DOC'
---
type: card
---
# Token

A token.
DOC
cat > AGENTS.md <<'DOC'
# For agents

Read docs/auth.md.
Read docs/sessions.md.
Read docs/faq.md.
Read docs/history.md.
Then start.
DOC
git add . && git commit -q -m "docs: describe authentication and sessions"
# The code moves on; the doc still names what was removed.
sed -i '/Revoke/d' src/token.go
git commit -qam "refactor(auth): drop session revocation"
