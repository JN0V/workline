#!/bin/sh
# A project whose backlog is planted for the product owner's evaluation: an
# export that dropped the last row, fixed since, and five open issues on the
# local forge — one the fix made obsolete, one true and its duplicate, one
# true that the fix's code only looks to solve, and one need.
set -eu
# Hermetic: ignore the machine's git config and global hooks.
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
git init -q -b main .
git config user.name "Fixture"
git config user.email "fixture@example.invalid"
export GIT_AUTHOR_DATE="2026-01-01T00:00:00Z" GIT_COMMITTER_DATE="2026-01-01T00:00:00Z"
mkdir -p src/export .workline
cat > src/export/csv.go <<'GO'
package export

// WriteRows writes the rows, one line each.
func WriteRows(rows []string, write func(string)) {
	for i := 0; i < len(rows)-1; i++ {
		write(rows[i])
	}
}
GO
printf '# Project\n\nExports rows as CSV.\n' > README.md
printf 'forge: local\n' > .workline/config.yaml
git add . && git commit -q -m "feat: export rows as CSV"
first=$(git rev-parse --short HEAD)
export GIT_AUTHOR_DATE="2026-01-02T00:00:00Z" GIT_COMMITTER_DATE="2026-01-02T00:00:00Z"
sed -i 's/len(rows)-1/len(rows)/' src/export/csv.go
git commit -q -am "fix(export): write the last row"

issues=$(git rev-parse --path-format=absolute --git-common-dir)/workline/issues
mkdir -p "$issues"
issue() { # number, title, body: an open issue, its state confirmed at the first commit
	printf -- '---\ntitle: %s\nstate: open\nlabels: []\n---\n%s\n\n<!-- workline-comment -->\nWhat workline knows of this issue.\n\n```yaml\nsources: [src/export/csv.go]\nconfirmed: %s\n```\n\n<!-- workline:sticky=product-owner/state -->\n' \
		"$2" "$3" "$first" > "$issues/$1.md"
}
issue 1 "Export drops the last row" "Every CSV export is missing its last row: WriteRows in src/export/csv.go stops one row short."
issue 2 "Export writes no header row" "The CSV has no header line: column names are never written. WriteRows in src/export/csv.go only writes the rows."
issue 3 "Column names missing from the CSV" "The first line of the export is already data, there is no line with the column names. src/export/csv.go writes no header."
issue 4 "Empty rows become blank lines" "An empty row is written as a blank line, which breaks the importer on the other side. WriteRows in src/export/csv.go writes every row as it is."
issue 5 "Export to JSON" "Please add a JSON export next to the CSV one."
