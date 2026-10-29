#!/bin/sh
# A cmd: agent for the developer's cases, to prove their scoring and
# measure the prompt without a real one (checks_test.go). It answers the
# issue the prompt names (# Issue #<n>) with developer-answers/$1/<n>.yaml,
# or nothing. It reports the prompt's size as tokens in, a character in
# four, and keeps the prompt in $WORKLINE_EVAL_PROMPTS/<n>.md when set.
dir=$(dirname "$0")/developer-answers/$1
prompt=$(cat)
n=$(printf '%s\n' "$prompt" | sed -n 's/^# Issue #\([0-9]*\):.*/\1/p' | head -n 1)
printf 'model: stand-in\ntokens-in: %d\ntokens-out: 0\n' $(($(printf '%s' "$prompt" | wc -c) / 4)) > "$WORKLINE_CALL"
if [ -n "${WORKLINE_EVAL_PROMPTS:-}" ]; then printf '%s\n' "$prompt" > "$WORKLINE_EVAL_PROMPTS/$n.md"; fi
if [ -f "$dir/$n.yaml" ]; then cat "$dir/$n.yaml"; else echo '[]'; fi
