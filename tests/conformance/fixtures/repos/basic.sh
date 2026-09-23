#!/bin/sh
# A small project with one source file, one doc and a clean history.
set -eu
# Hermetic: ignore the machine's git config and global hooks.
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
git init -q -b main .
git config user.name "Fixture"
git config user.email "fixture@example.invalid"
export GIT_AUTHOR_DATE="2026-01-01T00:00:00Z" GIT_COMMITTER_DATE="2026-01-01T00:00:00Z"
printf 'package main\n\nfunc main() {}\n' > main.go
printf '# Project\n\nA small project.\n' > README.md
git add . && git commit -q -m "feat: start the project"
