#!/bin/sh
# A small project whose issues the product owner keeps: an export that once
# dropped the last row, fixed since.
set -eu
# Hermetic: ignore the machine's git config and global hooks.
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
git init -q -b main .
git config user.name "Fixture"
git config user.email "fixture@example.invalid"
export GIT_AUTHOR_DATE="2026-01-01T00:00:00Z" GIT_COMMITTER_DATE="2026-01-01T00:00:00Z"
mkdir -p src/export
cat > src/export/csv.go <<'GO'
package export

// WriteRows writes every row, the last one included.
func WriteRows(rows []string, write func(string)) {
	for i := 0; i < len(rows); i++ {
		write(rows[i])
	}
}
GO
printf '# Project\n\nExports rows as CSV.\n' > README.md
git add . && git commit -q -m "feat: export rows as CSV"
