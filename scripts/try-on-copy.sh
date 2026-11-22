#!/usr/bin/env bash
# Try a ref or a release of workline on a real GitHub copy, in one command.
#
#   scripts/try-on-copy.sh <version|ref> [owner/workline]
#
# X.Y.Z (or vX.Y.Z) is a version: the scratch repository downloads that
# release as the pin does. Anything else is a ref (a pull request head, a
# branch, a commit): the engine is built from it (cmd/workline) and its
# workflow templates (defaults/ci/github) are installed.
#
# A scratch repository is made afresh from the trial copy at a fixed tag.
# The trial copy is only read: only the scratch repository changes.
# Needs gh (logged in, rights to create repositories), git, go for a ref,
# tar, sha256sum.
# Prints green or red and the link of the run; exits 0 on green only.
set -euo pipefail

TRIAL=JN0V/TraceGlass-trial
BASE_TAG=${TRY_BASE_TAG:-try-base-1}   # a fixed tag of the trial copy, never moved
TIMEOUT=${TRY_TIMEOUT:-3600}           # seconds the issue gets to end

arg=${1:?usage: try-on-copy.sh <version|ref> [owner/workline]}
WORKLINE=${2:-JN0V/workline}

# One try at a time: a run of try.yml in progress or queued, other than this one.
running=$(gh run list -R "$WORKLINE" --workflow try.yml --limit 50 --json databaseId,status,url \
  --jq '[.[] | select(.status == "in_progress" or .status == "queued") | select((.databaseId|tostring) != "'"${GITHUB_RUN_ID:-0}"'")] | .[0].url // empty')
if [ -n "$running" ]; then
  echo "red: a try is already running: $running" >&2
  exit 1
fi

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

owner=${GITHUB_REPOSITORY_OWNER:-$(gh api user --jq .login)}
scratch="$owner/workline-try-$(date +%Y%m%d%H%M%S)"

# The scratch repository, made afresh from the trial copy at its tag
# (read only: a clone of the tag, pushed to the scratch repository alone).
gh repo create "$scratch" --private >/dev/null
git clone --quiet --depth 1 --branch "$BASE_TAG" "https://github.com/$TRIAL.git" "$work/scratch"
cd "$work/scratch"
git checkout --quiet -B main
git remote remove origin
git remote add scratch "https://github.com/$scratch.git"

mkdir -p .github/workflows
case "$arg" in
  [0-9]*.[0-9]*.[0-9]*|v[0-9]*.[0-9]*.[0-9]*)
    pin=v${arg#v}   # the release as published, downloaded as the pin would
    src="https://github.com/$WORKLINE"
    git clone --quiet --depth 1 --branch "$pin" "$src.git" "$work/workline"
    ;;
  *)
    git clone --quiet "https://github.com/$WORKLINE.git" "$work/workline"
    git -C "$work/workline" fetch --quiet origin "$arg" 2>/dev/null || true
    git -C "$work/workline" checkout --quiet "$arg" 2>/dev/null || git -C "$work/workline" checkout --quiet FETCH_HEAD
    pin=try-$(git -C "$work/workline" rev-parse --short=12 HEAD)
    # That commit's engine, as a release of the scratch repository.
    mkdir -p "$work/dist"
    (cd "$work/workline" && CGO_ENABLED=0 go build -o "$work/dist/workline" ./cmd/workline)
    tar -czf "$work/workline_linux_amd64.tar.gz" -C "$work/dist" workline
    (cd "$work" && sha256sum workline_linux_amd64.tar.gz > checksums.txt)
    src="https://github.com/$scratch"
    ;;
esac

# That commit's workflow templates (defaults/ci/github), pinned to the release tried.
for f in "$work"/workline/defaults/ci/github/*.yml; do
  case "$(basename "$f")" in issue-form-need.yml|release-candidate.yml) continue ;; esac
  sed -e "s|^\(  WORKLINE_VERSION: \)[^ ]*|\1$pin|" \
      -e "s|https://github.com/JN0V/workline/releases/download|$src/releases/download|" \
      "$f" > ".github/workflows/$(basename "$f")"
done
if [ -f "$work/workline/defaults/ci/github/issue-form-need.yml" ]; then
  mkdir -p .github/ISSUE_TEMPLATE
  cp "$work/workline/defaults/ci/github/issue-form-need.yml" .github/ISSUE_TEMPLATE/need.yml
fi

git add -A
git -c user.name=workline-try -c user.email=try@localhost commit --quiet -m "try: workflows of $arg"
git push --quiet scratch main

if [ "$src" = "https://github.com/$scratch" ]; then
  gh release create "$pin" -R "$scratch" --title "$pin" --notes "engine built from $arg" \
    "$work/workline_linux_amd64.tar.gz" "$work/checksums.txt" >/dev/null
fi

# One issue, end to end.
gh label create workline:ready -R "$scratch" --force >/dev/null
issue=$(gh issue create -R "$scratch" --title "Try $arg" --label workline:ready \
  --body "A ready issue of the try of $arg.")
echo "issue: $issue"

# Forge facts: a pull request opened, its checks green, its labels set.
deadline=$(( $(date +%s) + TIMEOUT ))
verdict=red
why="no pull request opened"
while [ "$(date +%s)" -lt "$deadline" ]; do
  pr=$(gh pr list -R "$scratch" --state all --json number --jq '.[0].number // empty')
  if [ -n "$pr" ]; then
    checks=$(gh pr view "$pr" -R "$scratch" --json statusCheckRollup \
      --jq '[.statusCheckRollup[] | (.conclusion // .state // .status)] | if length == 0 then "none" elif all(. == "SUCCESS" or . == "SKIPPED" or . == "NEUTRAL") then "green" elif any(. == "FAILURE" or . == "ERROR" or . == "CANCELLED" or . == "TIMED_OUT") then "red" else "pending" end')
    labels=$(gh pr view "$pr" -R "$scratch" --json labels --jq '.labels | length')
    why="pull request #$pr: checks $checks, $labels labels"
    if [ "$checks" = green ] && [ "$labels" -gt 0 ]; then verdict=green; break; fi
    if [ "$checks" = red ]; then break; fi
  fi
  sleep 30
done

url="https://github.com/$scratch/actions"
if [ -n "${GITHUB_RUN_ID:-}" ]; then url="${GITHUB_SERVER_URL:-https://github.com}/$WORKLINE/actions/runs/$GITHUB_RUN_ID"; fi
echo "$verdict: $why"
echo "run: $url"
echo "scratch repository: https://github.com/$scratch (delete it when read)"
[ "$verdict" = green ] || exit 1
