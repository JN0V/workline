#!/bin/sh
# A cmd: agent writing #3's mul, and in lib.sh the forge tokens it was
# handed — none, when the engine kept them from it — and whether its own
# login reached it.
cat >/dev/null
seen=""
for v in GH_TOKEN GITHUB_TOKEN GITLAB_TOKEN WORKLINE_GITLAB_TOKEN CI_JOB_TOKEN WORKLINE_APP_PRIVATE_KEY; do
	eval "set=\${$v+x} val=\${$v-}"
	[ -n "$set" ] && [ "$val" != workline-sealed ] && seen="$seen $v" # the seal's placeholder opens nothing
done
login=no
[ -n "${ANTHROPIC_API_KEY:-}" ] && login=yes
cat <<YAML
- patch:
    file: lib.sh
    content: |
      # tokens seen:${seen:- none}; login seen: $login
      add() { echo \$((\$1 + \$2)); }
      mul() { echo \$((\$1 * \$2)); }
YAML
