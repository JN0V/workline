#!/bin/sh
# A project released as v1.2.0, followed by two conventional commits.
set -eu
# Hermetic: ignore the machine's git config and global hooks.
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
git init -q -b main .
git config user.name "Fixture"
git config user.email "fixture@example.invalid"
export GIT_AUTHOR_DATE="2026-01-01T00:00:00Z" GIT_COMMITTER_DATE="2026-01-01T00:00:00Z"
printf 'package main\n\nfunc main() {}\n' > main.go
printf '# Changelog\n\n## v1.2.0\n\n- First stable release.\n' > CHANGELOG.md
git add . && git commit -q -m "feat: first stable release"
git tag -a v1.2.0 -m "v1.2.0"
printf 'package main\n\nfunc Export() {}\n' > export.go
git add . && git commit -q -m "feat(export): let users export their data as CSV"
printf '\n// fixed\n' >> main.go
git add . && git commit -q -m "fix: stop crashing on an empty input file"
