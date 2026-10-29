#!/bin/sh
# A cmd: agent writing #3's mul while another run claims #3: the claim
# label lands on the simulated forge, beside the case's models-seen file.
cat >/dev/null
forge="$(dirname "$WORKLINE_MODELS_SEEN")/forge.json"
sed -i 's/"workline:ready"\]/"workline:ready","workline:developing"]/' "$forge"
cat <<'YAML'
- patch:
    file: lib.sh
    content: |
      add() { echo $(($1 + $2)); }
      mul() { echo $(($1 * $2)); }
YAML
