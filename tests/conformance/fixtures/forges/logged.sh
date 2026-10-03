#!/bin/sh
# A forge plugged by a command (docs/spec/forge-command.md), for the cases.
# It writes each request, one line, to .git/forge-requests.jsonl in the
# repository it runs in, and answers as a forge holding issue 8, ready to be
# read, and merge request 1 from `feature`. FORGE_FAIL makes every request
# fail (the forge unreachable); FORGE_REFUSE makes it answer an error.
set -eu
req=$(cat)
printf '%s\n' "$req" >> .git/forge-requests.jsonl
if [ -n "${FORGE_FAIL:-}" ]; then
	echo "$FORGE_FAIL" >&2
	exit 1
fi
if [ -n "${FORGE_REFUSE:-}" ]; then
	printf '{"error": "%s"}\n' "$FORGE_REFUSE"
	exit 0
fi
case "$WORKLINE_FORGE_OPERATION" in
issue) printf '%s\n' '{"id": 8, "title": "Export as CSV", "body": "## Need\nExport as CSV\n## Verification\nA test exports three rows\n## Validation\nThe product owner opens it\n## Scope\nsrc/export", "labels": ["workline:to-refine"]}' ;;
open-merge-requests) echo '{"branches": []}' ;;
merge-request-branch) echo '{"branch": "feature", "base": "main", "here": true}' ;;
open-issue | keep-issue | open-merge-request) echo '{"id": 7}' ;;
*) echo '{}' ;;
esac
