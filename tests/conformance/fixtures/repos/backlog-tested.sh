#!/bin/sh
# The backlog project with its tests: an export that once dropped the last
# row, fixed since, and a test that proves it — what a parent's Verification
# may name as proof (ADR-0029).
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
cat > src/export/csv_test.go <<'GO'
package export

import "testing"

func TestWriteRowsKeepsTheLast(t *testing.T) {
	var got []string
	WriteRows([]string{"a", "b", "c"}, func(r string) { got = append(got, r) })
	if len(got) != 3 {
		t.Fatalf("wrote %d rows, want 3", len(got))
	}
}
GO
git add . && git commit -q -m "test: three rows written, three read back"
