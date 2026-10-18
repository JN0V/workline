#!/bin/sh
# A Go project with lint debt on main, as fixtures/bin/golangci-lint reads
# it: three unchecked errors in clean.go, one more hidden by a //nolint
# with its reason, and a comment with no period, which godot would find.
set -eu
# Hermetic: ignore the machine's git config and global hooks.
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
git init -q -b main .
git config user.name "Fixture"
git config user.email "fixture@example.invalid"
export GIT_AUTHOR_DATE="2026-01-01T00:00:00Z" GIT_COMMITTER_DATE="2026-01-01T00:00:00Z"
cat > clean.go <<'EOF'
package main

import "os"

// clean removes what a run left.
func clean() {
	os.Remove("a.tmp")
	os.Remove("b.tmp")
	os.Remove("c.tmp")
	os.Remove("lock") //nolint:errcheck // gone already when the run ended well
}
EOF
cat > main.go <<'EOF'
package main

// main runs, then cleans up
func main() {
	clean()
}
EOF
git add . && git commit -q -m "feat: start the project"
