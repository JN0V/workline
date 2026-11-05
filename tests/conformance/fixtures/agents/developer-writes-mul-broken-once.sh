#!/bin/sh
# A cmd: agent writing #3's mul in lib.sh with a brace left out: lib.sh
# does not parse, a typo of the agent's (#1083). Asked again with the
# error (build-broken), it writes it whole; with ALWAYS set, it never
# does.
prompt=$(cat)
end=''
if [ -z "$ALWAYS" ] && printf '%s\n' "$prompt" | grep -q 'build-broken'; then
	end='}'
fi
cat <<YAML
- patch:
    file: lib.sh
    content: |
      add() { echo \$((\$1 + \$2)); }
      mul() { echo \$((\$1 * \$2)); $end
YAML
