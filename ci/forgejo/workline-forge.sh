#!/bin/sh
# workline's forge on Forgejo or Gitea (Codeberg included), plugged by a
# command (docs/spec/forge-command.md):
#
#   forge: 'cmd:sh ci/forgejo/workline-forge.sh'     # .workline/config.yaml
#
# One request a run: JSON on the input, its answer as JSON on the output;
# an exit other than 0 says the forge is unreachable. Needs curl and jq, and:
#
#   FORGEJO_URL    https://codeberg.org, or your instance (no trailing /)
#   FORGEJO_TOKEN  a token with the repository's issue and pull request
#                  scopes, write; read-only is enough for `workline item`;
#                  `comments` needs its user to administer the repository
#   FORGEJO_REPO   owner/name (default: read from the `origin` remote)
#
# Forgejo and Gitea answer the GitHub-like API /api/v1. A sample, written
# against their API documentation: not yet tried on a live instance.
set -eu

: "${FORGEJO_URL:?set FORGEJO_URL, e.g. https://codeberg.org}"
: "${FORGEJO_TOKEN:?set FORGEJO_TOKEN}"
if [ -z "${FORGEJO_REPO:-}" ]; then
	FORGEJO_REPO=$(git remote get-url origin | sed -E 's#\.git$##; s#^.*[:/]([^/]+/[^/]+)$#\1#')
fi
base="$FORGEJO_URL/api/v1/repos/$FORGEJO_REPO"
req=$(cat)
arg() { printf '%s' "$req" | jq -r "$1"; }
arg_json() { printf '%s' "$req" | jq -c "$1"; }

# api METHOD PATH [JSON]: the answer on the output; fails on any HTTP error.
api() {
	if [ $# -ge 3 ]; then
		printf '%s' "$3" | curl -fsS -X "$1" -H "Authorization: token $FORGEJO_TOKEN" \
			-H 'Content-Type: application/json' --data-binary @- "$base$2"
	else
		curl -fsS -X "$1" -H "Authorization: token $FORGEJO_TOKEN" "$base$2"
	fi
}

# all PATH: every page of a list, as one array.
all() {
	page=1
	out='[]'
	while :; do
		sep='?'
		case "$1" in *\?*) sep='&' ;; esac
		got=$(api GET "$1${sep}limit=50&page=$page")
		[ "$(printf '%s' "$got" | jq length)" -eq 0 ] && break
		out=$(printf '%s\n%s' "$out" "$got" | jq -sc add)
		page=$((page + 1))
	done
	printf '%s' "$out"
}

# The comment holding a marker, by id, on an issue or pull request (they
# share numbers and comments on Forgejo, as on GitHub).
comment_with() { # number marker
	api GET "/issues/$1/comments" | jq -r --arg m "$2" 'map(select(.body | contains($m))) | .[0].id // empty'
}

open_issue_titled() { # title
	all "/issues?state=open&type=issues" | jq -r --arg t "$1" 'map(select(.title == $t)) | .[0].number // empty'
}

label_id() { # name; created when missing
	id=$(all "/labels" | jq -r --arg n "$1" 'map(select(.name == $n)) | .[0].id // empty')
	if [ -z "$id" ]; then
		id=$(api POST "/labels" "$(jq -nc --arg n "$1" '{name: $n, color: "#ededed"}')" | jq -r .id)
	fi
	printf '%s' "$id"
}

case "$WORKLINE_FORGE_OPERATION" in
issue)
	api GET "/issues/$(arg .id)" | jq -c '{id: .number, title, body: (.body // ""), labels: [.labels[].name]}'
	;;
all-issues)
	# Open and closed, pull requests left out. Forgejo and Gitea keep no
	# close reason: `reason` is left out, and the engine reads a closed
	# issue as done (docs/spec/forge-command.md).
	all "/issues?state=all&type=issues" | jq -c '{issues: [.[] | select(.pull_request == null)
		| {id: .number, title, body: (.body // ""), labels: [(.labels // [])[].name], closed: (.state == "closed")}]}'
	;;
comments)
	# Oldest first, each with its author and its day; `insider` when the author may
	# write to the repository (Forgejo's permission: write, admin or owner),
	# asked once an author — which needs the token's user to administer the
	# repository: a lookup refused fails the operation, loud, rather than
	# taking everyone for an outsider. An author gone (404) is outside.
	# Forgejo's API has no bot field: its system users (a negative id: the
	# ghost, the actions user) and a login ending in "-bot" or "[bot]" are
	# bots, whose `agreed` never counts.
	got=$(all "/issues/$(arg .target.id)/comments")
	perms='{}'
	tmp=$(mktemp)
	# One login a line, read whole; put in the URL escaped, curl's globbing
	# off: a login like `ci[bot]` is a name, not a range.
	printf '%s' "$got" | jq -r '[.[].user.login] | unique | .[]' > "$tmp.logins"
	while IFS= read -r u; do
		code=$(curl -gsS -o "$tmp" -w '%{http_code}' -H "Authorization: token $FORGEJO_TOKEN" \
			"$base/collaborators/$(jq -rn --arg u "$u" '$u | @uri')/permission") || exit 1
		case "$code" in
		200) p=$(jq -r '.permission // "none"' "$tmp") ;;
		404) p=none ;;
		*)
			echo "the permission of $u could not be read (HTTP $code): the token's user must administer the repository" >&2
			rm -f "$tmp" "$tmp.logins"
			exit 1
			;;
		esac
		perms=$(printf '%s' "$perms" | jq -c --arg u "$u" --arg p "$p" '. + {($u): $p}')
	done < "$tmp.logins"
	rm -f "$tmp" "$tmp.logins"
	printf '%s' "$got" | jq -c --argjson perms "$perms" '{comments: [.[] | {body: (.body // ""), author: .user.login,
		insider: ($perms[.user.login] | IN("write", "admin", "owner")),
		bot: ((.user.id // 0) < 0 or (.user.login | test("(-bot|\\[bot\\])$")))} + (if .created_at then {created: .created_at} else {} end)]}'
	;;
comment | sticky)
	n=$(arg .target.id)
	marker=$(arg .marker)
	text=$(printf '%s' "$req" | jq -c '{body: (.body + "\n\n" + .marker)}')
	id=$(comment_with "$n" "$marker")
	if [ -n "$id" ]; then
		[ "$WORKLINE_FORGE_OPERATION" = sticky ] && api PATCH "/issues/comments/$id" "$text" >/dev/null
	elif [ "$WORKLINE_FORGE_OPERATION" = comment ] || [ "$(arg .create)" = true ]; then
		api POST "/issues/$n/comments" "$text" >/dev/null
	fi
	echo '{}'
	;;
label)
	n=$(arg .target.id)
	# One name a line, read whole: a label's name may hold spaces.
	ids=$(arg '(.add // [])[]' | while IFS= read -r l; do label_id "$l" && echo; done)
	if [ -n "$ids" ]; then
		api POST "/issues/$n/labels" "$(printf '%s' "$ids" | jq -Rsc 'split("\n") | map(select(. != "") | tonumber) | {labels: .}')" >/dev/null
	fi
	arg '(.remove // [])[]' | while IFS= read -r l; do
		id=$(api GET "/issues/$n/labels" | jq -r --arg n "$l" 'map(select(.name == $n)) | .[0].id // empty')
		if [ -n "$id" ]; then api DELETE "/issues/$n/labels/$id" >/dev/null; fi
	done
	echo '{}'
	;;
open-issue)
	n=$(open_issue_titled "$(arg .title)")
	if [ -n "$n" ]; then
		marker=$(arg .marker)
		if [ -z "$(comment_with "$n" "$marker")" ]; then
			api POST "/issues/$n/comments" "$(printf '%s' "$req" | jq -c '{body: (.body + "\n\n" + .marker)}')" >/dev/null
		fi
	else
		n=$(api POST "/issues" "$(printf '%s' "$req" | jq -c '{title, body: (.body + "\n\n" + .marker)}')" | jq -r .number)
	fi
	jq -nc --argjson n "$n" '{id: $n}'
	;;
keep-issue)
	n=$(open_issue_titled "$(arg .title)")
	if [ -n "$n" ]; then
		api PATCH "/issues/$n" "$(arg_json '{body}')" >/dev/null
	elif [ "$(arg .create)" = true ]; then
		n=$(api POST "/issues" "$(arg_json '{title, body}')" | jq -r .number)
	else
		n=0
	fi
	jq -nc --argjson n "$n" '{id: $n}'
	;;
open-merge-request)
	n=$(all "/pulls?state=open" | jq -r --arg b "$(arg .branch)" 'map(select(.head.ref == $b)) | .[0].number // empty')
	if [ -n "$n" ]; then
		api PATCH "/pulls/$n" "$(arg_json '{title, body}')" >/dev/null
	else
		n=$(api POST "/pulls" "$(arg_json '{head: .branch, base, title, body}')" | jq -r .number)
	fi
	jq -nc --argjson n "$n" '{id: $n}'
	;;
open-merge-requests)
	all "/pulls?state=open" | jq -c --arg p "$(arg .prefix)" '{branches: [.[].head.ref | select(startswith($p))] | sort}'
	;;
merge-request-branch)
	api GET "/pulls/$(arg .id)" | jq -c '{branch: .head.ref, here: (.head.repo.full_name == .base.repo.full_name), title: .title, body: (.body // "")}'
	;;
*)
	printf '{"error": "unknown operation %s"}\n' "$WORKLINE_FORGE_OPERATION"
	;;
esac
