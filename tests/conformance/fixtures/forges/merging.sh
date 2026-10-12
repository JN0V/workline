#!/bin/sh
# A forge plugged by a command (docs/spec/forge-command.md) holding one
# merge request for the merge policy's cases (ADR-0042): 7, from `feature`
# into main, a person's, labelled workline:accepted by dana, who can
# write; its auto-merge off. Each request is written, one line, to
# .git/forge-requests.jsonl; it keeps a verdict on `feature`'s head, and
# holds a yes as it was given.
set -eu
req=$(cat)
printf '%s\n' "$req" >> .git/forge-requests.jsonl
case "$WORKLINE_FORGE_OPERATION" in
open-merge-requests) echo '{"branches": []}' ;;
merge-requests) echo '{"merge-requests": [{"id": 7, "branch": "feature", "base": "main", "here": true, "author": "dana", "labels": ["workline:accepted"]}]}' ;;
merge-state) echo '{"label-events": [{"label": "workline:accepted", "added": true, "author": "dana", "insider": true, "writer": true, "created": "2026-01-04T10:00:00Z"}], "pushes": [{"head": "'"$(git rev-parse feature)"'", "at": "2026-01-04T09:00:00Z"}]}' ;;
auto-merge) echo '{"merged": false}' ;;
verdict) # the verdict it keeps: `feature`'s head passed
	case "$req" in
	*"$(git rev-parse feature)"*) echo '{"class": "pass", "why": "the line passed"}' ;;
	*) echo '{}' ;;
	esac ;;
guarded) echo '{"why": ""}' ;;
*) echo '{}' ;;
esac
