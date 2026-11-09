#!/bin/sh
# A forge plugged by a command (docs/spec/forge-command.md) that does not
# know `merge-requests`: merge request 2, from feat/x, closes issue 7.
set -eu
cat > /dev/null
case "$WORKLINE_FORGE_OPERATION" in
merge-request-branch) echo '{"branch": "feat/x", "base": "main", "here": true, "title": "Multiply", "body": "Closes #7", "draft": false}' ;;
merge-requests) echo '{"error": "merge-requests: not known"}' ;;
*) echo '{}' ;;
esac
