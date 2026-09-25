#!/bin/sh
# Runs the evaluation on the local main branch, in a worktree of its own, and
# appends the scores to the repository's results.tsv. It skips a run when the
# commit already has enough runs and no model changed since the last one: the
# same code on the same models adds little after a few runs.
#
#   tests/evaluation/schedule/run.sh [repository]   # WORKLINE_EVAL: the agent, default claude
#
# The systemd units next to it run it every night (docs/spec/conformance.md).
set -eu
repo=$(cd "${1:-$(dirname "$0")/../../..}" && pwd)
agent=${WORKLINE_EVAL:-claude}
runs=${WORKLINE_EVAL_RUNS:-3}              # runs per commit before skipping
cache=${XDG_CACHE_HOME:-$HOME/.cache}/workline
tree=$cache/eval-tree
results=$repo/tests/evaluation/results.tsv
seen=${WORKLINE_MODELS_SEEN:-$cache/models-seen.yaml}
stamp=$cache/eval-last-run
export PATH="$HOME/.local/go/bin:$HOME/.local/bin:$HOME/go/bin:/usr/local/bin:/usr/bin:/bin"

mkdir -p "$cache"
if [ -d "$tree" ]; then
	git -C "$tree" checkout -q --detach main
else
	git -C "$repo" worktree add -q --detach "$tree" main
fi
commit=$(git -C "$tree" rev-parse --short HEAD)

# runs of this commit so far: the fewest any case has had
cases=$(ls "$tree"/tests/evaluation/cases/*/*.yaml | wc -l)
done_runs=$(awk -F'\t' -v c="$commit" -v a="$agent" -v cases="$cases" '
	$2 == c && $3 == a { n[$6]++ }
	END { m = 0; k = 0; for (x in n) { k++; if (m == 0 || n[x] < m) m = n[x] } print (k < cases ? 0 : m) }' "$results")
model_changed=no
if [ -f "$seen" ] && { [ ! -f "$stamp" ] || [ "$seen" -nt "$stamp" ]; }; then
	model_changed=yes
fi
if [ "$done_runs" -ge "$runs" ] && [ "$model_changed" = no ]; then
	echo "workline eval: $commit has $done_runs runs with $agent, and no model changed: skipped"
	exit 0
fi

echo "workline eval: $commit, $agent (runs so far: $done_runs, model changed: $model_changed)"
cd "$tree"
WORKLINE_EVAL=$agent WORKLINE_EVAL_RESULTS=$results go test -count=1 -timeout 60m ./tests/evaluation/
touch "$stamp" # after the run: the models it saw answer are not a change
