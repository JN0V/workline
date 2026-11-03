#!/bin/sh
# The small project of backlog.sh, with a doc beside its code: an issue
# may name the doc, not the code it is about (#980).
set -eu
# Hermetic: ignore the machine's git config and global hooks.
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
git init -q -b main .
git config user.name "Fixture"
git config user.email "fixture@example.invalid"
export GIT_AUTHOR_DATE="2026-01-01T00:00:00Z" GIT_COMMITTER_DATE="2026-01-01T00:00:00Z"
mkdir -p src/export docs
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
printf '# Export\n\nThe export writes each row on a line.\n' > docs/export.md
git add . && git commit -q -m "feat: export rows as CSV"
