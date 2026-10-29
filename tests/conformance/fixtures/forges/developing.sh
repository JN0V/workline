#!/bin/sh
# A forge plugged by a command (docs/spec/forge-command.md), for the
# developer's cases: issue 3 ready, the tester's Draft 1 from
# workline/tester/issue-3 holding its record in workline's own comment.
# Each request is written, one line, to .git/forge-requests.jsonl; a
# merge request opened is 2.
set -eu
req=$(cat)
printf '%s\n' "$req" >> .git/forge-requests.jsonl
case "$WORKLINE_FORGE_OPERATION" in
issues) printf '%s\n' '{"issues": [{"id": 3, "title": "Multiply two numbers", "labels": ["workline:ready"], "body": "## Need\n\nI want to multiply.\n\n## Scope\n\n- `lib.sh`: a `mul` function beside `add`.\n"}]}' ;;
comments)
	case "$req" in
	*'"merge-request"'*) printf '%s\n' '{"comments": [{"own": true, "body": "The tests of #3, frozen.\n\n<!-- workline:tester-record {\"issue\":3,\"base\":\"main\",\"files\":{\"tests/test_mul_by_zero.sh\":\"8f123a0bacbd681063880b867df38dd216784b97\",\"tests/test_mul_two_by_three.sh\":\"2dee0641bd7a79057ade7bff5af43b603125f543\"}} -->\n<!-- workline:sticky=tester/tests -->"}]}' ;;
	*) echo '{"comments": []}' ;;
	esac ;;
merge-requests) echo '{"merge-requests": [{"id": 1, "branch": "workline/tester/issue-3", "base": "main", "here": true, "draft": true}]}' ;;
open-merge-requests) echo '{"branches": []}' ;;
open-merge-request) echo '{"id": 2}' ;;
*) echo '{}' ;;
esac
