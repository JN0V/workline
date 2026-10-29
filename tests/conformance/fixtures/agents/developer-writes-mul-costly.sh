#!/bin/sh
# A cmd: agent writing #3's mul right, and saying its call spent 4000
# tokens in and 1000 out: past a run's ai-max-tokens of 2000.
cat >/dev/null
printf 'model: costly\ntokens-in: 4000\ntokens-out: 1000\n' > "$WORKLINE_CALL"
cat <<'YAML'
- patch:
    file: lib.sh
    content: |
      add() { echo $(($1 + $2)); }
      mul() { echo $(($1 * $2)); }
YAML
