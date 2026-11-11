#!/bin/sh
# A cmd: agent on a merge round (ADR-0067): its worktree holds the
# branch with main merged in, lib.sh's conflict markers in it. It keeps
# both sides, main's sub() and its own mul(), the markers gone. EDIT:
# leaves the markers (markers); or settles lib.sh and edits NOTES.md,
# main's file not in conflict (other); or keeps its own side only,
# dropping main's sub() (ours). It answers a note.
set -eu
cat >/dev/null
[ -n "${WORKLINE_WORKTREE:-}" ] || { echo "no WORKLINE_WORKTREE" >&2; exit 1; }
grep -q '^<<<<<<<' lib.sh || { echo "no conflict markers in lib.sh" >&2; exit 1; }
case "${EDIT:-}" in
markers) ;;
ours) printf 'add() { echo $(($1 + $2)); }\nmul() { echo $(($1 * $2)); }\n' > lib.sh ;;
*)
	printf 'add() { echo $(($1 + $2)); }\nsub() { echo $(($1 - $2)); }\nmul() { echo $(($1 * $2)); }\n' > lib.sh
	[ "${EDIT:-}" != other ] || echo 'Edited by the merge round.' >> NOTES.md
	;;
esac
echo "- note: \"lib.sh: kept main's sub() and this branch's mul()\""
