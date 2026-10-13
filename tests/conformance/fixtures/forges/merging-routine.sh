#!/bin/sh
# A forge plugged by a command (docs/spec/forge-command.md) holding one of
# workline's own merge requests for the merge policy's `normal` level
# (ADR-0042, amended): 9, from `workline/documentalist/suspect` into main,
# opened by workline's App, every push to its branch the App's, unlabelled,
# its auto-merge off. Each request is written, one line, to
# .git/forge-requests.jsonl; it keeps a verdict on the branch's head,
# holds a yes as it is given, and no issue.
set -eu
req=$(cat)
printf '%s\n' "$req" >> .git/forge-requests.jsonl
head=$(git rev-parse workline/documentalist/suspect)
case "$WORKLINE_FORGE_OPERATION" in
app) echo '{"login": "workline-app[bot]", "committer": "workline-app[bot]"}' ;;
open-merge-requests) echo '{"branches": []}' ;;
merge-requests) echo '{"merge-requests": [{"id": 9, "branch": "workline/documentalist/suspect", "base": "main", "here": true, "author": "workline-app[bot]", "bot": true, "labels": []}]}' ;;
merge-state) echo '{"pushes": [{"head": "'"$head"'", "at": "2026-01-04T09:00:00Z"}], "pushers": ["workline-app[bot]"]}' ;;
auto-merge) echo '{"merged": false}' ;;
verdict) # the verdict it keeps: the branch's head passed
	case "$req" in
	*"$head"*) echo '{"class": "pass", "why": "the line passed, every agent'"'"'s review whole"}' ;;
	*) echo '{}' ;;
	esac ;;
guarded) echo '{"why": ""}' ;;
issues) echo '{"issues": []}' ;;
*) echo '{}' ;;
esac
