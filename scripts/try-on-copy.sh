#!/usr/bin/env bash
# Try a release or a ref of workline on a scratch copy of a real repository
# (ADR-0071 §2, docs/github/try-on-copy.md): one issue runs end to end,
# product owner, tester, developer, on GitHub's own CI; the try asserts forge
# facts only, then deletes its scratch repository.
#
#   scripts/try-on-copy.sh [--dry-run] [--keep] [--timeout MIN] <version|ref> [source[@tag]]
#
# Exit: 0 green, 1 red, 2 usage, 3 another try is running, 4 setup failed.
set -euo pipefail

usage() {
	cat <<'EOF'
usage: scripts/try-on-copy.sh [--dry-run] [--keep] [--timeout MIN] [--seed FILE] <version|ref> [source[@tag]]

  <version>   a published release, v0.78.4: downloaded as the pin would
  <ref>       a commit, a branch or pull/<n>/head of workline: its engine
              built, its templates (defaults/ci/github) installed
  source      the repository copied, default JN0V/TraceGlass-trial@try-base-1

  --dry-run   print the plan, touch nothing on GitHub
  --keep      keep the scratch repository after the try
  --timeout   minutes to wait for the line, default 90
  --seed      the seeded issue (first line its title, the rest its body);
              required for a source other than the default

Environment: CLAUDE_CODE_OAUTH_TOKEN, and WORKLINE_APP_PRIVATE_KEY (the
App's .pem) or WORKLINE_APP_PRIVATE_KEY_FILE; WORKLINE_APP_ID, else the
source's variable; gh logged in with repo, workflow and delete_repo.
EOF
}

dry=0 keep=0 timeout_min=90 seed_file=""
args=()
while [ $# -gt 0 ]; do
	case $1 in
	--dry-run) dry=1 ;;
	--keep) keep=1 ;;
	--timeout) timeout_min=${2:?--timeout needs minutes}; shift ;;
	--seed) seed_file=${2:?--seed needs a file}; shift ;;
	-h | --help) usage; exit 0 ;;
	-*) echo "try: unknown option $1" >&2; usage >&2; exit 2 ;;
	*) args+=("$1") ;;
	esac
	shift
done
if [ ${#args[@]} -lt 1 ] || [ ${#args[@]} -gt 2 ]; then usage >&2; exit 2; fi
case $timeout_min in '' | *[!0-9]*) echo "try: --timeout takes whole minutes" >&2; exit 2 ;; esac

default_source=JN0V/TraceGlass-trial
workline_repo=${TRY_WORKLINE_REPO:-JN0V/workline}
target=${args[0]}
source=${args[1]:-$default_source@try-base-1}
src_repo=${source%@*}
src_tag=try-base-1
[ "$source" != "$src_repo" ] && src_tag=${source#*@}
owner=${TRY_OWNER:-${src_repo%%/*}}
id=${GITHUB_RUN_ID:-$(date -u +%Y%m%d%H%M%S)}
scratch="$owner/workline-try-$id"

if [[ $target =~ ^v?[0-9]+\.[0-9]+\.[0-9]+(-rc\.[0-9]+)?$ ]]; then
	kind=version version=v${target#v}
else
	kind=ref version=""
fi
if [ -z "$seed_file" ] && [ "$src_repo" != "$default_source" ]; then
	echo "try: --seed FILE is needed for $src_repo: the default issue is $default_source's" >&2
	exit 2
fi

log() { printf '%s try: %s\n' "$(date -u +%H:%M:%S)" "$*"; }
die() { log "$1"; exit "${2:-4}"; }

plan() {
	echo "try of $kind ${version:-$target} on $scratch, a copy of $src_repo at $src_tag"
	if [ "$kind" = version ]; then
		echo "  1. engine: release $version of $workline_repo, downloaded by the templates as the pin would"
		echo "  2. templates: defaults/ci/github of $version"
	else
		echo "  1. engine: built from $target, published as a release engine-<sha> of $scratch"
		echo "  2. templates: defaults/ci/github of $target, downloading that release"
	fi
	echo "  3. create $scratch, public; push $src_repo at $src_tag with the templates as main"
	echo "  4. settings: variables WORKLINE_APP_ID, WORKLINE_RECONCILE_GAP_MINUTES; secret CLAUDE_CODE_OAUTH_TOKEN;"
	echo "     environments workline-app (the App's key, main only) and workline-heartbeat; labels, rulesets"
	echo "     of $src_repo; the pull_request_target event policy; the App installed on $scratch"
	echo "  5. open the seeded issue, labelled workline:to-refine, as the person"
	echo "  6. wait up to $timeout_min min: a pull request on workline/developer/<issue> opened by the line,"
	echo "     its checks green (but workline/accepted, the person's yes), the issue labelled"
	echo "     workline:needs-tests, workline:ready and workline:developing by the line"
	if [ $keep = 1 ]; then echo "  7. keep $scratch"; else echo "  7. delete $scratch"; fi
}

plan
if [ $dry = 1 ]; then
	echo "dry run: nothing done on GitHub"
	exit 0
fi

# Preflight: every tool and secret, and no other try running.
for tool in gh git curl openssl tar sha256sum; do
	command -v "$tool" >/dev/null || die "missing tool: $tool"
done
[ "$kind" = ref ] && { command -v go >/dev/null || die "missing tool: go, to build the ref's engine"; }
[ -n "${CLAUDE_CODE_OAUTH_TOKEN:-}" ] || die "CLAUDE_CODE_OAUTH_TOKEN is not set: the line's roles would run with no agent"
if [ -z "${WORKLINE_APP_PRIVATE_KEY:-}" ] && [ -z "${WORKLINE_APP_PRIVATE_KEY_FILE:-}" ]; then
	die "WORKLINE_APP_PRIVATE_KEY (or _FILE) is not set: the line needs the App's key"
fi
gh auth status >/dev/null 2>&1 || die "gh is not logged in"
running=$(gh repo list "$owner" --limit 1000 --json name --jq '.[].name | select(startswith("workline-try-"))')
if [ -n "$running" ]; then
	die "a try is running, or was kept: $owner/$(echo "$running" | head -1); one at a time (delete it with gh repo delete when it is left over)" 3
fi

work=$(mktemp -d)
created=0
cleanup() {
	status=$?
	if [ $created = 1 ]; then
		if [ $keep = 1 ]; then
			log "kept https://github.com/$scratch: delete it with gh repo delete $scratch --yes"
		elif gh repo delete "$scratch" --yes >/dev/null 2>&1; then
			log "deleted $scratch"
		else
			log "could not delete $scratch: delete it by hand, the next try refuses while it is there"
		fi
	fi
	rm -rf "$work"
	exit $status
}
trap cleanup EXIT
trap 'exit 130' INT TERM
trap 'log "failed at line $LINENO: $BASH_COMMAND"; exit 4' ERR

key="$work/app.pem"
(umask 077; if [ -n "${WORKLINE_APP_PRIVATE_KEY:-}" ]; then printf '%s\n' "$WORKLINE_APP_PRIVATE_KEY" >"$key"; else cp "$WORKLINE_APP_PRIVATE_KEY_FILE" "$key"; fi)
app_id=${WORKLINE_APP_ID:-$(gh variable get WORKLINE_APP_ID -R "$src_repo" 2>/dev/null || true)}
[ -n "$app_id" ] || die "WORKLINE_APP_ID: neither set nor a variable of $src_repo"
git_auth=(-c credential.helper= -c 'credential.helper=!gh auth git-credential')

# 1-2. The engine and the templates of the version or the ref.
log "fetching $workline_repo"
git clone -q --filter=blob:none "https://github.com/$workline_repo" "$work/workline"
wl() { git -C "$work/workline" "$@"; }
if [ "$kind" = version ]; then
	gh release view "$version" -R "$workline_repo" --json assets --jq '.assets[].name' 2>/dev/null |
		grep -qx workline_linux_amd64.tar.gz || die "$version: no published release of $workline_repo with workline_linux_amd64.tar.gz"
	wl -c advice.detachedHead=false checkout -q "refs/tags/$version"
	engine_version=$version
	download="https://github.com/$workline_repo/releases/download/"
else
	if ! sha=$(wl rev-parse -q --verify "$target^{commit}"); then
		wl fetch -q origin "$target" || die "$target: no such ref in $workline_repo"
		sha=$(wl rev-parse FETCH_HEAD)
	fi
	wl -c advice.detachedHead=false checkout -q "$sha"
	engine_version=engine-${sha:0:12}
	download="https://github.com/$scratch/releases/download/"
	log "building the engine of $sha"
	mkdir -p "$work/dist" "$work/assets"
	(cd "$work/workline" && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath \
		-ldflags "-s -w -X main.version=$engine_version" -o "$work/dist/workline" ./cmd/workline)
	tar -C "$work/dist" -czf "$work/assets/workline_linux_amd64.tar.gz" workline
	(cd "$work/assets" && sha256sum workline_linux_amd64.tar.gz >checksums.txt)
fi

log "copying $src_repo at $src_tag"
git clone -q "https://github.com/$src_repo" "$work/copy"
cp_() { git -C "$work/copy" "$@"; }
cp_ rev-parse -q --verify "refs/tags/$src_tag^{commit}" >/dev/null || die "$src_repo has no tag $src_tag"
cp_ checkout -q -B main "refs/tags/$src_tag"
wf="$work/copy/.github/workflows"
mkdir -p "$wf"
rm -f "$wf"/workline*.yml
for t in "$work/workline/defaults/ci/github"/workline*.yml; do
	# workline-release.yml holds a release tool's pull request: the copy has none.
	[ "$(basename "$t")" = workline-release.yml ] && continue
	sed -E -e "s#^([[:space:]]*WORKLINE_VERSION:).*#\\1 $engine_version#" \
		-e "s#https://github.com/$workline_repo/releases/download/#$download#g" "$t" >"$wf/$(basename "$t")"
done
for f in "$wf"/workline*.yml; do
	if grep -q 'releases/download' "$f" && ! grep -q "WORKLINE_VERSION: $engine_version" "$f"; then
		die "$(basename "$f") downloads the engine with no WORKLINE_VERSION line: the try cannot pin it"
	fi
done
cp_ add .github/workflows
# The scratch repository is not workline's: its commit takes no hook of the machine's.
cp_ -c user.name="workline try" -c user.email="workline-try@users.noreply.github.com" -c core.hooksPath=/dev/null \
	commit -q -m "ci: workline $engine_version on the try" --allow-empty

# 3. The scratch repository.
log "creating $scratch"
gh repo create "$scratch" --public --description "workline try of ${version:-$target}: a throwaway copy of $src_repo, deleted after" >/dev/null
created=1
cp_ "${git_auth[@]}" -c core.hooksPath=/dev/null push -q "https://github.com/$scratch" main:main
repo_id=$(gh api "repos/$scratch" --jq .id)
if [ "$kind" = ref ]; then
	gh release create "$engine_version" -R "$scratch" --target main --prerelease --title "$engine_version" \
		--notes "workline $sha, built for the try" "$work/assets/workline_linux_amd64.tar.gz" "$work/assets/checksums.txt" >/dev/null
fi

# 4. Settings, as the source has them.
log "settings"
gh variable set WORKLINE_APP_ID -R "$scratch" --body "$app_id"
gh variable set WORKLINE_RECONCILE_GAP_MINUTES -R "$scratch" --body 15
printf '%s' "$CLAUDE_CODE_OAUTH_TOKEN" | gh secret set CLAUDE_CODE_OAUTH_TOKEN -R "$scratch"
for env in workline-app workline-heartbeat; do
	wait=0
	[ $env = workline-heartbeat ] && wait=15
	gh api -X PUT "repos/$scratch/environments/$env" --input - >/dev/null <<EOF
{"wait_timer": $wait, "deployment_branch_policy": {"protected_branches": false, "custom_branch_policies": true}}
EOF
	gh api -X POST "repos/$scratch/environments/$env/deployment-branch-policies" -f name=main -f type=branch >/dev/null
done
gh secret set WORKLINE_APP_PRIVATE_KEY --env workline-app -R "$scratch" <"$key"
gh label clone "$src_repo" -R "$scratch" --force >/dev/null
if ! gh api -X POST "repos/$scratch/actions/policies" --input - >/dev/null 2>&1 <<'EOF'
{"name": "workline line", "enforcement": "active",
 "conditions": {"workflow_path": {"exclude": [], "include": [
   ".github/workflows/workline-bounds.yml", ".github/workflows/workline-merge.yml"]}},
 "rules": [{"type": "restrict_action_events",
   "parameters": {"allowed_events": ["pull_request_target", "issues", "schedule"]}}]}
EOF
then
	log "warning: no Actions event policy set on $scratch: from December 2, 2026, its pull_request_target runs are blocked"
fi

# The App on the scratch repository: its installation, read with a JWT.
b64() { openssl base64 -A | tr '+/' '-_' | tr -d '='; }
now=$(date +%s)
jwt_body="$(printf '{"alg":"RS256","typ":"JWT"}' | b64).$(printf '{"iat":%d,"exp":%d,"iss":"%s"}' $((now - 60)) $((now + 540)) "$app_id" | b64)"
jwt="$jwt_body.$(printf '%s' "$jwt_body" | openssl dgst -sha256 -sign "$key" | b64)"
read -r inst selection < <(gh api -H "Authorization: Bearer $jwt" app/installations \
	--jq ".[] | select(.account.login == \"$owner\") | \"\(.id) \(.repository_selection)\"" 2>/dev/null) || true
[ -n "${inst:-}" ] || die "the App $app_id is not installed on $owner"
if [ "$selection" != all ] && ! gh api -X PUT "user/installations/$inst/repositories/$repo_id" >/dev/null 2>&1; then
	die "the App's installation (selected repositories) could not take $scratch: install the App on all of $owner's repositories"
fi

# The rulesets, their required checks from any source: GitHub refuses an
# App's id the repository has not seen yet.
for rs in $(gh api "repos/$src_repo/rulesets" --jq '.[] | select(.source_type == "Repository") | .id'); do
	gh api "repos/$src_repo/rulesets/$rs" --jq 'del(.id, .node_id, ._links, .created_at, .updated_at, .source,
		.source_type, .current_user_can_bypass) | del(.rules[].parameters.required_status_checks[]?.integration_id)' |
		gh api -X POST "repos/$scratch/rulesets" --input - >/dev/null
done

# 5. The seeded issue, as the person.
if [ -n "$seed_file" ]; then
	title=$(head -1 "$seed_file")
	tail -n +2 "$seed_file" >"$work/seed.md"
else
	title="Only snapshot files count as time-lapse frames"
	cat >"$work/seed.md" <<'EOF'
As a contributor I want the time-lapse to count and compile only its own snapshot files, so a stray picture in its folder never shows up in the video.

## Example

The time-lapse folder holds `snapshot_0000.jpg`, `snapshot_0001.jpg`, `snapshot_0002.jpg` and a stray `cover.jpg`. Today the restore dialog says 4 frames, and the video ends on `cover.jpg`. It should say 3, and the video hold the 3 snapshots.

## Validation

- **Stray picture left out:** Given 3 snapshots and `cover.jpg` in the folder, When the snapshots are listed, Then exactly the 3 snapshots come back, in order.
- **Count:** Given the same folder, When the snapshots are counted, Then the count is 3.
- **Name without a number:** Given `snapshot_abc.jpg` in the folder, When the snapshots are listed, Then it is not among them.
EOF
fi
issue_url=$(gh issue create -R "$scratch" --title "$title" --body-file "$work/seed.md" --label enhancement --label workline:to-refine)
issue=${issue_url##*/}
log "seeded $issue_url"

# 6. Forge facts only.
deadline=$((SECONDS + timeout_min * 60))
verdict=red why="timed out after $timeout_min min" pr="" fail_since="" green_polls=0 seen=""
want_labels="workline:needs-tests workline:ready workline:developing"
line_labels() {
	gh api --paginate "repos/$scratch/issues/$issue/timeline" \
		--jq '.[] | select(.event == "labeled" and .actor.type == "Bot") | .label.name' 2>/dev/null | sort -u | tr '\n' ' ' || true
}
while [ $SECONDS -lt $deadline ]; do
	labels=$(line_labels)
	[ "$labels" != "$seen" ] && { log "issue labelled by the line: ${labels:-none yet}"; seen=$labels; }
	pr=$(gh pr list -R "$scratch" --state all --json number,headRefName,author \
		--jq ".[] | select(.headRefName == \"workline/developer/$issue\" and .author.is_bot) | .number" 2>/dev/null | head -1 || true)
	if [ -n "$pr" ]; then
		checks=$(gh pr checks "$pr" -R "$scratch" --json name,bucket \
			--jq '.[] | select(.name != "workline/accepted") | "\(.bucket) \(.name)"' 2>/dev/null || true)
		pending=$(grep -c '^pending' <<<"$checks" || true)
		failed=$(grep -cE '^(fail|cancel)' <<<"$checks" || true)
		missing=""
		for l in $want_labels; do [[ " $labels " == *" $l "* ]] || missing="$missing $l"; done
		if [ -n "$checks" ] && [ "$pending" = 0 ] && [ "$failed" = 0 ] && grep -qx 'pass judge' <<<"$checks" && [ -z "$missing" ]; then
			green_polls=$((green_polls + 1))
			# Twice in a row: a review round may start new checks.
			[ $green_polls -ge 2 ] && { verdict=green why="pull request opened by the line, its checks green, the labels set"; break; }
		else
			green_polls=0
		fi
		if [ -n "$checks" ] && [ "$pending" = 0 ] && [ "$failed" != 0 ]; then
			fail_since=${fail_since:-$SECONDS}
			# A round may fix a red check: red after 15 minutes of no change.
			[ $((SECONDS - fail_since)) -ge 900 ] && { why="checks red for 15 min: $(grep -E '^(fail|cancel)' <<<"$checks" | tr '\n' ';')"; break; }
		else
			fail_since=""
		fi
	fi
	sleep 60
done

# The facts, kept in the log: the scratch repository goes.
echo "--- facts"
echo "issue: $issue_url"
echo "labels set by the line: ${labels:-none}"
if [ -n "$pr" ]; then
	echo "pull request: https://github.com/$scratch/pull/$pr"
	gh pr checks "$pr" -R "$scratch" 2>/dev/null || true
else
	echo "pull request: none on workline/developer/$issue"
fi
gh run list -R "$scratch" --limit 40 --json workflowName,event,conclusion,url \
	--jq '.[] | "\(.conclusion // "running") \(.workflowName) (\(.event)) \(.url)"' 2>/dev/null || true
run_link=""
[ -n "${GITHUB_RUN_ID:-}" ] && run_link="${GITHUB_SERVER_URL:-https://github.com}/${GITHUB_REPOSITORY:-$workline_repo}/actions/runs/$GITHUB_RUN_ID"
echo "---"
if [ $verdict = green ]; then
	log "GREEN: $why. ${run_link:-$issue_url}"
else
	log "RED: $why. ${run_link:-$issue_url}"
fi
if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
	printf '### Try of %s: %s\n\n%s\n' "${version:-$target}" "$verdict" "$why" >>"$GITHUB_STEP_SUMMARY"
fi
[ $verdict = green ]
