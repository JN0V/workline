#!/bin/sh
# A forge plugged by a command (docs/spec/forge-command.md) that cannot
# say how a pull request ended (it knows no `closed-with`): issue 3 is
# claimed since 2026-01-01, its record says its code was written, its
# pull request is not open. Each request is written, one line, to
# .git/forge-requests.jsonl.
set -eu
req=$(cat)
printf '%s\n' "$req" >> .git/forge-requests.jsonl
case "$WORKLINE_FORGE_OPERATION" in
issues) printf '%s\n' '{"issues": [{"id": 3, "title": "Multiply", "labels": ["workline:developing"], "body": "## Need\n\nI want to multiply.\n"}]}' ;;
comments)
	case "$req" in
	*'"kind":"issue"'*) printf '%s\n' '{"comments": ["**The developer** takes this issue.\n\n<!-- workline:developer-act {\"act\":\"code\",\"issue\":3,\"code\":[\"lib.sh\"],\"body\":\"8a828a06be19\"} -->\n<!-- workline:sticky=developer/claim -->"]}' ;;
	*) echo '{"comments": []}' ;;
	esac ;;
trail) echo '{"labeled": "2026-01-01T09:00:00Z"}' ;;
merge-requests) echo '{"merge-requests": []}' ;;
open-merge-requests) echo '{"branches": []}' ;;
*) echo '{}' ;;
esac
