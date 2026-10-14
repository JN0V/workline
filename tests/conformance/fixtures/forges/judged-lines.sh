#!/bin/sh
# judged-lines.sh <n> <commit>: writes n lines the reviewer hid in its
# summary comment for findings judged on earlier runs (ADR-0046), in place
# of @judged@ in the simulated forge's file (FORGE_FILE): old/0001.go the
# oldest, each at the commit given, escaped as that JSON file holds a
# comment. For a case whose comment grows past its size.
set -eu
n=$1 at=$2
c=$(cat "$FORGE_FILE")
{
	printf '%s' "${c%%@judged@*}"
	i=1
	while [ "$i" -le "$n" ]; do
		f=$(printf '%04d' "$i")
		printf '%s' "<!-- workline:judged {\\\"lens\\\":\\\"correctness\\\",\\\"where\\\":\\\"old/$f.go:1\\\",\\\"at\\\":\\\"$at\\\",\\\"yes\\\":false,\\\"finder\\\":\\\"claude\\\\u002dsonnet\\\\u002d5\\\",\\\"judge\\\":\\\"claude\\\\u002dopus\\\\u002d5\\\",\\\"title\\\":\\\"An old finding, judged on an earlier run, its title as long as a real one\\\"} -->\\n"
		i=$((i + 1))
	done
	printf '%s' "${c#*@judged@}"
} > "$FORGE_FILE.new"
mv "$FORGE_FILE.new" "$FORGE_FILE"
