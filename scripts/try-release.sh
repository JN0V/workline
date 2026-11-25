#!/usr/bin/env bash
# Try a release of workline on the permanent trial copy, under the person's
# own gh session: the release and doctor, the copy's upgrade pull request
# (the person labels it), then one canned issue merged by policy.
# Why and how: docs/adr/0071-the-line-runs-a-pinned-release-tried-on-a-copy.md
# Exit: 0 green, 1 red, 2 waiting on the person, 3 refused. A command that
# fails is refused before the lock is taken, red after it.
set -euo pipefail

REPO=${TRY_REPO:-JN0V/TraceGlass-trial}
WORKLINE=${TRY_WORKLINE:-JN0V/workline}
POLL=${TRY_POLL:-300}
CHECKS_MINUTES=${TRY_CHECKS_MINUTES:-90}
LABEL_MINUTES=${TRY_LABEL_MINUTES:-120}
DISPATCH_MINUTES=${TRY_DISPATCH_MINUTES:-15}
GIT_URL=${TRY_GIT_URL:-https://github.com}   # where the repositories are cloned from
LOCK=workline-try-lock   # a tag on the copy: one try at a time, whoever runs it

usage() {
  cat >&2 <<'EOF'
usage: try-release.sh [--dry-run] [--minutes N] <vX.Y.Z>
  --dry-run    print the plan, call no forge, change nothing
  --minutes N  how long the canned issue gets to reach a merge (default 240)
Run it again after a stop: it finds the pull request and the issue it made.
Environment: TRY_REPO (JN0V/TraceGlass-trial), TRY_WORKLINE (JN0V/workline),
TRY_POLL seconds between two reads (300), TRY_CHECKS_MINUTES (90),
TRY_LABEL_MINUTES, the wait for the person's label and the merge (120),
TRY_DISPATCH_MINUTES, how long a fix green and accepted waits for its merge
before the script dispatches the copy's workline merge workflow once (15).
EOF
  exit 3
}

dry=false
minutes=240
version=
while [ $# -gt 0 ]; do
  case $1 in
    --dry-run) dry=true ;;
    --minutes) [ $# -ge 2 ] || usage; minutes=$2; shift ;;
    --) shift; [ $# -eq 1 ] || usage; version=$1 ;;
    -*) usage ;;
    *) [ -z "$version" ] || usage; version=$1 ;;
  esac
  shift
done
[ -n "$version" ] || usage
case $minutes in ''|*[!0-9]*) echo "refused: --minutes takes a whole number, not $minutes" >&2; exit 3 ;; esac
# A published version alone: vX.Y.Z or vX.Y.Z-rc.N. Never a ref, never a
# word starting with a dash, so no command takes it for an option.
if ! [[ $version =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-rc\.[0-9]+)?$ ]]; then
  echo "refused: $version is not a release (vX.Y.Z or vX.Y.Z-rc.N)" >&2
  exit 3
fi

os=$(uname -s | tr '[:upper:]' '[:lower:]')
case $(uname -m) in x86_64|amd64) arch=amd64 ;; aarch64|arm64) arch=arm64 ;; *) arch=$(uname -m) ;; esac
asset="workline_${os}_${arch}.tar.gz"
branch="workline-$version"

# The canned issues: small, real bugs of the copy, a reproduced bug being
# what merges by policy at `normal`. Each opens test-first
# (workline:needs-tests) with its kind, so the copy's line starts on it
# and its merge needs no person. A try takes the first one never opened
# on the copy; each one fixed is spent. Add one when the last is spent,
# with its kind.
canned_titles=(
  "An outline of 65536 by 65536 pixels with no pixels crashes instead of being refused"
  "A matrix of 8 numbers crashes the composition with an index error"
  "A focal length of 0 gives a paper aspect ratio that is not a number"
  "A snapshot saved with a negative number is counted but never listed"
  "A scale factor of 0 makes the overlay vanish for good"
  "A viewport zoom that is not a number sticks for good"
  "A restored session keeps a pan its zoom does not allow"
)
canned_kinds=(bug bug bug bug bug bug bug)
canned_body() {
  case $1 in
    0) cat <<'EOF'
A person loading a huge reference picture gets an index crash from the outline filter, where a picture whose pixels do not fit its size is meant to be refused with a clear message.

Example: `OutlineFilter.filter(IntArray(0), 65536, 65536, 10)` throws `ArrayIndexOutOfBoundsException`. `65536 * 65536` overflows `Int` to 0, so the size check `pixels.size == width * height` passes for an empty array. `OutlineFilter.filter(IntArray(3), 2, 2, 10)` is refused with `IllegalArgumentException`, as expected.

## Validation

- **Scenario: a size whose product overflows is refused**
  - Given an empty pixel array
  - When the outline filter runs on 65536 by 65536 pixels
  - Then it is refused with an `IllegalArgumentException` naming the size
- **Scenario: a picture of the right size is still filtered**
  - Given a 2 by 2 picture with one dark pixel
  - When the outline filter runs with a threshold of 10
  - Then its edge pixels come back opaque black

---

## Scope

`core/overlay/src/main/kotlin/io/github/jn0v/traceglass/core/overlay/OutlineFilter.kt`: the size check of `filter` only, computed without overflow (as a `Long`).
EOF
    ;;
    1) cat <<'EOF'
A person whose overlay composes a matrix of the wrong size gets an index crash, where a 3 by 3 matrix of 9 numbers is meant to be required with a clear message.

Example: `MatrixUtils.multiply(FloatArray(8), MatrixUtils.identity())` throws `ArrayIndexOutOfBoundsException: Index 8 out of bounds for length 8`. `MatrixUtils.multiply(FloatArray(10), MatrixUtils.identity())` returns a result, its tenth number silently ignored. Two matrices of 9 numbers multiply as expected.

## Validation

- **Scenario: a matrix of 8 numbers is refused**
  - Given a matrix of 8 numbers
  - When it is multiplied by the identity
  - Then it is refused with an `IllegalArgumentException` saying a matrix holds 9 numbers
- **Scenario: a matrix of 10 numbers is refused**
  - Given a matrix of 10 numbers
  - When it is multiplied by the identity
  - Then it is refused the same way
- **Scenario: two 3 by 3 matrices still multiply**
  - Given a translation and a scale
  - When they are multiplied
  - Then the result is their product, as today

---

## Scope

`core/overlay/src/main/kotlin/io/github/jn0v/traceglass/core/overlay/MatrixUtils.kt`: a size check at the start of `multiply` only; `compose` goes through it.
EOF
    ;;
    2) cat <<'EOF'
A person whose focal length is estimated at 0 gets a paper aspect ratio that is not a number, where a focal length that is not positive is meant to be refused as degenerate (null), as the constrained homography already does.

Example: with `rect` the 210 by 297 rectangle `(0,0) (210,0) (210,297) (0,297)` and `moved` the same rectangle moved by (100, 50), `HomographySolver.correctAspectRatio(rect, moved, 0f, 540f, 960f)` returns `NaN`. With a focal length of 800, the same call returns 0.7070707. `solveConstrainedHomography` returns null for a focal length of 0.

## Validation

- **Scenario: a focal length of 0 is refused**
  - Given a 210 by 297 rectangle and the same rectangle moved by (100, 50)
  - When the aspect ratio is corrected with a focal length of 0
  - Then the result is null
- **Scenario: a negative focal length is refused**
  - Given the same two rectangles
  - When the aspect ratio is corrected with a focal length of -800
  - Then the result is null
- **Scenario: a real focal length still corrects**
  - Given the same two rectangles
  - When the aspect ratio is corrected with a focal length of 800
  - Then the result is 210 / 297, within 0.01

---

## Scope

`core/overlay/src/main/kotlin/io/github/jn0v/traceglass/core/overlay/HomographySolver.kt`: a guard at the start of `correctAspectRatio` only, returning null for a focal length that is not positive and finite.
EOF
    ;;
    3) cat <<'EOF'
A time-lapse whose snapshot is saved with a negative number counts a frame its video never gets: the file is written and counted, but never listed.

Example: on an empty storage, `FileSnapshotStorage.getSnapshotCount()` returns 0. `saveSnapshot(byteArrayOf(1), -1)` writes `snapshot_-001.jpg`; then `getSnapshotCount()` returns 1 while `getSnapshotFiles()` lists 0 files, as the name does not match the snapshot pattern.

## Validation

- **Scenario: a negative number is refused**
  - Given an empty snapshot storage
  - When a snapshot is saved with the number -1
  - Then it is refused with an `IllegalArgumentException` naming -1
  - And no file is written, and the count stays 0
- **Scenario: a snapshot numbered 0 is still saved**
  - Given an empty snapshot storage
  - When a snapshot is saved with the number 0
  - Then `snapshot_0000.jpg` is written, listed and counted once

---

## Scope

`feature/timelapse/src/main/kotlin/io/github/jn0v/traceglass/feature/timelapse/FileSnapshotStorage.kt`: a check at the start of `saveSnapshot` only.
EOF
    ;;
    4) cat <<'EOF'
A person whose pinch reports a scale factor of 0 loses the overlay for the rest of the session: every later pinch multiplies 0, so its scale never comes back.

Example: on a new `TracingViewModel`, `onOverlayScale(0f)` then `onOverlayScale(2f)` leaves `uiState.overlayScale` at 0.0. `onOverlayScale(2f)` alone gives 2.0. `onOverlayGesture` multiplies the same factor by its zoom, with no check either.

## Validation

- **Scenario: a scale factor of 0 is ignored**
  - Given an overlay at scale 1
  - When it is scaled by 0, then by 2
  - Then its scale is 2
- **Scenario: a negative or not-a-number factor is ignored**
  - Given an overlay at scale 1
  - When it is scaled by -1, then by NaN
  - Then its scale is still 1
- **Scenario: a pinch zoom of 0 is ignored**
  - Given an overlay at scale 1
  - When a gesture reports a zoom of 0
  - Then its scale is still 1
- **Scenario: real factors still accumulate**
  - Given an overlay at scale 1
  - When it is scaled by 2, then by 0.5
  - Then its scale is 1, as today

---

## Scope

`feature/tracing/src/main/kotlin/io/github/jn0v/traceglass/feature/tracing/TracingViewModel.kt`: `onOverlayScale` and the zoom of `onOverlayGesture` only; a factor that is not positive and finite is ignored.
EOF
    ;;
    5) cat <<'EOF'
A person whose zoom gesture reports a factor that is not a number gets a viewport zoom stuck at NaN: every later zoom multiplies NaN, and the clamp to 1 to 5 lets NaN through.

Example: with the overlay locked, `onViewportZoom(Float.NaN)` then `onViewportZoom(2f)` leaves `uiState.viewportZoom` at NaN. `onViewportZoom(2f)` alone gives 2.0.

## Validation

- **Scenario: a zoom factor that is not a number is ignored**
  - Given a locked overlay at viewport zoom 1
  - When the viewport is zoomed by NaN
  - Then its zoom is still 1, and its pan is still 0 by 0
- **Scenario: a real zoom still applies after it**
  - Given a locked overlay zoomed by NaN
  - When the viewport is zoomed by 2
  - Then its zoom is 2
- **Scenario: the zoom is still clamped**
  - Given a locked overlay at viewport zoom 1
  - When the viewport is zoomed by 10
  - Then its zoom is 5, as today

---

## Scope

`feature/tracing/src/main/kotlin/io/github/jn0v/traceglass/feature/tracing/TracingViewModel.kt`: `onViewportZoom` only; a factor that is not finite is ignored.
EOF
    ;;
    6) cat <<'EOF'
A person resuming a locked session whose saved pan is beyond what its saved zoom allows sees the picture off-centre, until the first pan makes it jump back.

Example: a session saved locked, at viewport zoom 1 with a pan of (500, -800). On the default 1080 by 1920 view, after `restoreSession()` and `onResumeSessionAccepted()`, `uiState` has a pan of (500.0, -800.0). At zoom 1 no pan is allowed: `onViewportPan` clamps each axis to (zoom - 1) × view size / 2.

## Validation

- **Scenario: a pan beyond zoom 1 is reset**
  - Given a saved locked session at zoom 1 with a pan of (500, -800)
  - When it is resumed on a 1080 by 1920 view
  - Then its pan is (0, 0)
- **Scenario: a pan beyond zoom 2 is clamped**
  - Given a saved locked session at zoom 2 with a pan of (900, -1200)
  - When it is resumed on a 1080 by 1920 view
  - Then its pan is (540, -960)
- **Scenario: a pan within its zoom is kept**
  - Given a saved locked session at zoom 2 with a pan of (300, -400)
  - When it is resumed on a 1080 by 1920 view
  - Then its pan is (300, -400)

---

## Scope

`feature/tracing/src/main/kotlin/io/github/jn0v/traceglass/feature/tracing/TracingViewModel.kt`: `onResumeSessionAccepted` only; the restored pan is clamped with the bounds `onViewportPan` uses.
EOF
    ;;
  esac
}

if $dry; then
  cat <<EOF
try of $version on $REPO (dry run: no forge is called, nothing changes)
1. lock: create the tag $LOCK on $REPO; refuse if it exists (another try runs); delete it at the end
2. release: download $asset and checksums.txt of $WORKLINE $version, check the checksum, check \`workline version\` says $version
3. doctor: run \`workline doctor\` on a clone of $REPO; clean means exit 0 and no ✗
4. upgrade: open or update the pull request $branch on $REPO: the templates of $WORKLINE's defaults/ci/github/, from the copy's pin to $version, merged into its .github/workflows/
5. checks: wait up to $CHECKS_MINUTES min for its checks, every $POLL s; green and mergeable, or red
6. label: print that the person labels it; wait up to $LABEL_MINUTES min for its merge (the script never labels)
7. issue: open the first canned issue never opened on $REPO, labelled workline:needs-tests and ${canned_kinds[0]}: "${canned_titles[0]}"
8. merge: wait up to $minutes min for a pull request closing it, merged with no person's label; one green and accepted by policy but not merged after $DISPATCH_MINUTES min: dispatch $REPO's workline merge workflow once
9. verdict: green or red, with the links
EOF
  exit 0
fi

for tool in gh git tar; do
  command -v "$tool" >/dev/null || { echo "refused: $tool is not on the PATH" >&2; exit 3; }
done
if command -v sha256sum >/dev/null; then sha_check() { sha256sum -c --quiet -; }
elif command -v shasum >/dev/null; then sha_check() { shasum -a 256 -c --quiet -; }
else echo "refused: neither sha256sum nor shasum is on the PATH" >&2; exit 3
fi

links=()
say_links() { for l in ${links[@]+"${links[@]}"}; do echo "  $l"; done; }
# verdict: the code the script chose; any other is a command's own.
verdict=
red() { echo "red: $*"; say_links; verdict=1; exit 1; }
waiting() { echo "waiting: $*"; say_links; verdict=2; exit 2; }

# The trap is set before the lock: whatever stops the script, the lock it
# took goes, and a command's own exit code (128 for git, 2 for tar) becomes
# the documented one: refused before the lock, red after it.
locked=false work=
cleanup() {
  local code=$?
  trap - EXIT
  if $locked; then
    gh api -X DELETE "repos/$REPO/git/refs/tags/$LOCK" >/dev/null 2>&1 \
      || echo "warning: the lock $LOCK stays on $REPO: delete it by hand" >&2
  fi
  [ -z "$work" ] || rm -rf "$work"
  if [ "$code" != 0 ] && [ "$code" != "$verdict" ]; then
    if $locked; then
      echo "red: a command failed (exit $code)"; say_links; code=1
    else
      echo "refused: a command failed (exit $code) before the try started" >&2; code=3
    fi
  fi
  exit "$code"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

# 1. One try at a time on the copy: creating a ref fails when it exists.
main_sha=$(gh api "repos/$REPO/commits/main" --jq .sha)
if ! lock_err=$(gh api -X POST "repos/$REPO/git/refs" -f ref="refs/tags/$LOCK" -f sha="$main_sha" 2>&1 >/dev/null); then
  if [[ $lock_err == *"Reference already exists"* ]]; then
    echo "refused: another try is running on $REPO (its tag $LOCK exists)." >&2
    echo "If none is, remove it: gh api -X DELETE repos/$REPO/git/refs/tags/$LOCK" >&2
  else
    echo "refused: the lock $LOCK cannot be created on $REPO: ${lock_err:-no reason given}" >&2
  fi
  verdict=3
  exit 3
fi
locked=true
work=$(mktemp -d)
echo "try of $version on $REPO, by $(gh api user --jq .login)"

# 2. The release, as the pin downloads it.
links+=("release: https://github.com/$WORKLINE/releases/tag/$version")
mkdir "$work/dl" "$work/bin"
gh release download -R "$WORKLINE" -D "$work/dl" -p "$asset" -p checksums.txt -- "$version" \
  || red "the release $version or its $asset cannot be downloaded"
(cd "$work/dl" && grep " $asset\$" checksums.txt | sha_check) || red "$asset does not match checksums.txt"
tar -xzf "$work/dl/$asset" -C "$work/bin" workline
got=$("$work/bin/workline" version)
[ "$got" = "$version" ] || red "the binary says $got, not $version"
echo "1. release: $asset, checksum ok, says $version"

# 3. doctor, on a clone of the copy.
git clone --quiet -- "$GIT_URL/$REPO.git" "$work/copy"
if ! "$work/bin/workline" doctor --repo "$work/copy" >"$work/doctor.txt" 2>&1 || grep -q '✗' "$work/doctor.txt"; then
  cat "$work/doctor.txt"
  red "workline doctor is not clean"
fi
echo "1. doctor: clean"

# 4. The upgrade pull request: merged already, or built afresh from the
# copy's main, pushed when its branch differs, opened when none is open.
pr_of() { gh pr list -R "$REPO" --head "$branch" --state "$1" --json number,url --jq '.[0] // empty | "\(.number) \(.url)"' 2>/dev/null || true; }
merged=$(pr_of merged)
pr=$(pr_of open)
fresh=false
if [ -z "$merged" ]; then
  from=$(grep -h 'WORKLINE_VERSION:' "$work/copy"/.github/workflows/workline*.yml 2>/dev/null | awk '{print $2}' | sort -u || true)
  [ -n "$from" ] && [ "$(printf '%s\n' "$from" | wc -l)" -eq 1 ] \
    || red "the copy's workflows do not pin one version: $(echo "$from" | tr '\n' ' ')"
  [ "$from" != "$version" ] || red "the copy runs $version already, with no upgrade pull request $branch merged to show it"
  for v in "$from" "$version"; do
    git clone --quiet --depth 1 --branch "$v" -- "$GIT_URL/$WORKLINE.git" "$work/wl-$v" 2>/dev/null \
      || red "no tag $v on $WORKLINE"
  done
  old="$work/wl-$from/defaults/ci/github" new="$work/wl-$version/defaults/ci/github"
  cd "$work/copy"
  git switch --quiet -c "$branch"
  for t in "$new"/*.yml; do
    name=$(basename "$t") f=.github/workflows/$(basename "$t")
    if [ ! -e "$old/$name" ]; then
      [ -e "$f" ] || cp "$t" "$f"        # a workflow new in this release
    elif [ -e "$f" ]; then               # one the copy left out stays out
      # The copy's own lines (a setup step) kept, the release's change taken.
      git merge-file --quiet "$f" "$old/$name" "$t" \
        || red "the release's change to $name conflicts with the copy's own lines"
    fi
  done
  git add .github/workflows
  if git diff --cached --quiet; then red "the templates from $from to $version change nothing on the copy"; fi
  git commit --quiet -m "ci: workline workflows at $version"
  remote=$(git ls-remote origin "refs/heads/$branch" | awk '{print $1}')
  if [ -n "$remote" ] && git fetch --quiet origin "refs/heads/$branch" \
    && [ "$(git rev-parse 'FETCH_HEAD^{tree}')" = "$(git rev-parse 'HEAD^{tree}')" ]; then
    :                                    # the branch holds this change already
  else
    git push --quiet --force-with-lease="refs/heads/$branch:$remote" origin "HEAD:refs/heads/$branch"
    fresh=true
  fi
  if [ -z "$pr" ]; then
    gh pr create -R "$REPO" --base main --head "$branch" --title "ci: workline workflows at $version" --body \
"Moves the copy's workline workflows from $from to $version: the templates' change between the two, the copy's own lines kept. A try of the release ([the try](https://github.com/$WORKLINE/blob/main/docs/adr/0071-the-line-runs-a-pinned-release-tried-on-a-copy.md#amendment-2026-11-24-a-try-is-a-script-on-the-permanent-copy)), opened by scripts/try-release.sh." >/dev/null
    pr=$(pr_of open)
    fresh=true
  fi
  cd - >/dev/null
fi

if [ -z "$merged" ]; then
  [ -n "$pr" ] || red "the upgrade pull request $branch did not open"
  read -r num url <<<"$pr"
  links+=("upgrade: $url")
  echo "2. upgrade pull request: $url"

  # 5. Its checks green, nothing in the way but the person's label. A fresh
  # pull request's checks are read after one wait, once they are queued.
  ! $fresh || sleep "$POLL"
  deadline=$(( $(date +%s) + CHECKS_MINUTES * 60 ))
  while :; do
    states=$(gh pr checks -R "$REPO" --json bucket --jq '[.[].bucket] | join(" ")' -- "$num" 2>/dev/null || true)
    case " $states " in *" fail "*|*" cancel "*) red "a check of the upgrade pull request failed" ;; esac
    if [ -n "$states" ] && [[ " $states " != *" pending "* ]]; then break; fi
    [ "$(date +%s)" -lt "$deadline" ] || red "its checks did not end within $CHECKS_MINUTES min"
    sleep "$POLL"
  done
  mergeable=$(gh pr view -R "$REPO" --json mergeable --jq .mergeable -- "$num" 2>/dev/null || true)
  [ "$mergeable" = MERGEABLE ] || red "the upgrade pull request is $mergeable, not mergeable"
  echo "2. checks: green; mergeable"

  # 6. The person's yes, never the script's.
  echo ">> Your yes: put the label workline:accepted on $url (this script never labels)."
  deadline=$(( $(date +%s) + LABEL_MINUTES * 60 ))
  until [ "$(gh pr view -R "$REPO" --json state --jq .state -- "$num" 2>/dev/null || true)" = MERGED ]; do
    [ "$(date +%s)" -lt "$deadline" ] \
      || waiting "the upgrade pull request is not merged after $LABEL_MINUTES min: label it, then run this again"
    sleep "$POLL"
  done
else
  links+=("upgrade: ${merged#* }")
fi
echo "2. upgrade pull request merged"

# 7. The canned issue: this version's (open, or closed by a merge), else
# the first whose bug is still there: never opened, or only closed as not
# planned. One fixed is spent; one open from another try is left to it.
issue=
for i in "${!canned_titles[@]}"; do
  title=${canned_titles[$i]}
  seen=$(gh issue list -R "$REPO" --state all --search "\"$title\" in:title" --json number,title,state,stateReason,body \
    --jq "[.[] | select(.title == \"$title\")] | (map(select(.stateReason != \"NOT_PLANNED\" and (.body | contains(\"workline-try: $version \")))) | .[0].number // \"\"), any(.stateReason == \"COMPLETED\" or .state == \"OPEN\")" \
    2>/dev/null) || red "the copy's issues cannot be read"
  ours=$(sed -n 1p <<<"$seen") taken=$(sed -n 2p <<<"$seen")
  if [ -n "$ours" ]; then issue=$ours; break; fi
  [ "$taken" = true ] && continue
  body="$(canned_body "$i")

<!-- workline-try: $version -->
Opened by the try of workline $version (scripts/try-release.sh)."
  issue=$(gh issue create -R "$REPO" --title "$title" --label workline:needs-tests --label "${canned_kinds[$i]}" --body "$body")
  issue=${issue##*/}
  break
done
[ -n "$issue" ] || red "every canned issue is spent: add one to canned_titles in scripts/try-release.sh"
links+=("issue: https://github.com/$REPO/issues/$issue")
echo "3. canned issue: https://github.com/$REPO/issues/$issue"

# 8. A pull request closing it, merged by policy: no person's label on it.
# Of its closing pull requests, the merged one counts. A label on an issue
# does not start the copy's merge workflow: one open, its checks green and
# workline/accepted set with no person's label, waits $DISPATCH_MINUTES min,
# then the workflow is dispatched once. The script never labels nor merges.
ready_jq='.state == "OPEN" and (.isDraft | not)
  and ([.labels[].name] | index("workline:accepted") == null)
  and ([.labels[].name] | index("workline:accepted-whole") == null)
  and any(.statusCheckRollup[]; .__typename == "StatusContext" and .context == "workline/accepted" and .state == "SUCCESS")
  and all(.statusCheckRollup[]; if .__typename == "StatusContext" then .state == "SUCCESS"
    else .conclusion == "SUCCESS" or .conclusion == "SKIPPED" or .conclusion == "NEUTRAL" end)'
ready_since= dispatched=false
# The dispatch is marked by a tag on the copy, kept across tries: a rerun
# after a stop sees it and dispatches nothing more for this issue.
mark="workline-try-dispatched-$issue"
if [ -n "$(gh api "repos/$REPO/git/matching-refs/tags/$mark" --jq '.[].ref' 2>/dev/null || true)" ]; then
  dispatched=true
  echo "3. the merge workflow was dispatched by an earlier try of this issue: not again"
fi
deadline=$(( $(date +%s) + minutes * 60 ))
while :; do
  read -r state closers <<<"$(gh issue view -R "$REPO" --json state,closedByPullRequestsReferences \
    --jq '"\(.state) \([.closedByPullRequestsReferences[].number] | join(","))"' -- "$issue" 2>/dev/null || true)"
  if [ "$state" = CLOSED ]; then
    found=
    for n in ${closers//,/ }; do
      read -r purl pstate labelled <<<"$(gh pr view -R "$REPO" --json url,state,labels \
        --jq '"\(.url) \(.state) \([.labels[].name] | index("workline:accepted") != null)"' -- "$n")"
      if [ "$pstate" = MERGED ]; then found=$purl; break; fi
    done
    [ -n "$found" ] || red "the canned issue closed with no merged pull request"
    links+=("merged: $found")
    [ "$labelled" = false ] || red "$found merged on a person's label, not by policy"
    break
  fi
  if ! $dispatched; then
    ready=
    for n in ${closers//,/ }; do
      if [ "$(gh pr view -R "$REPO" --json state,isDraft,labels,statusCheckRollup --jq "$ready_jq" -- "$n" 2>/dev/null || true)" = true ]; then
        ready=$n; break
      fi
    done
    if [ -z "$ready" ]; then ready_since=
    elif [ -z "$ready_since" ]; then ready_since=$(date +%s)
    fi
    if [ -n "$ready" ] && [ $(( $(date +%s) - ready_since )) -ge $(( DISPATCH_MINUTES * 60 )) ]; then
      gh api -X POST "repos/$REPO/git/refs" -f ref="refs/tags/$mark" -f sha="$main_sha" >/dev/null 2>&1 \
        || red "the dispatch marker $mark cannot be created on $REPO"
      if ! gh workflow run workline-merge.yml -R "$REPO" >/dev/null; then
        gh api -X DELETE "repos/$REPO/git/refs/tags/$mark" >/dev/null 2>&1 || true
        red "the copy's workline merge workflow cannot be dispatched"
      fi
      dispatched=true
      echo "3. https://github.com/$REPO/pull/$ready is green and accepted by policy, not merged after $DISPATCH_MINUTES min: dispatched the copy's workline merge workflow once"
    fi
  fi
  [ "$(date +%s)" -lt "$deadline" ] || red "no pull request merged for the canned issue within $minutes min"
  sleep "$POLL"
done
echo "3. merged by policy"

echo "green: $version tried on $REPO"
say_links
echo "The pin's pull request links both pull requests above under ## Tried."
