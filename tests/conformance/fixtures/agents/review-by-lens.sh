#!/bin/sh
# A cmd: lens of the reviewer, on the `reviewed` fixture once Average divides
# by one less than the count: each lens its task asks finds a defect of its
# own on that line, naming its lens — correctness the division, edge-cases
# a single value — so a case tells which lenses a run asked (ADR-0050).
# REVIEW_LENS_BROKEN names a lens whose task gets an answer that never
# reads; with REVIEW_LENS_BROKEN_CALLS, only its first that many calls,
# counted across runs in the file REVIEW_LENS_COUNT names. Anything else
# answers nothing.
task=$(cat)
asks() {
	case $task in
	*"# Lens: $1"*) return 0 ;;
	esac
	return 1
}
broken() {
	[ -n "${REVIEW_LENS_BROKEN:-}" ] && asks "$REVIEW_LENS_BROKEN" || return 1
	[ -n "${REVIEW_LENS_BROKEN_CALLS:-}" ] || return 0
	mkdir -p "$(dirname "$REVIEW_LENS_COUNT")"
	n=$(cat "$REVIEW_LENS_COUNT" 2>/dev/null || echo 0)
	echo $((n + 1)) >"$REVIEW_LENS_COUNT"
	[ "$n" -lt "$REVIEW_LENS_BROKEN_CALLS" ]
}
if broken; then
	echo 'finding: [this does not read'
	exit 0
fi
said=
if asks correctness; then
	said=1
	cat <<'YAML'
- finding:
    lens: correctness
    severity: important
    title: Average divides by one less than the count
    why: the mean of [2, 4] comes out 6
    cause: {path: calc/calc.go, quote: "return total / (len(values) - 1)"}
    fix: divide by len(values)
YAML
fi
if asks edge-cases; then
	said=1
	cat <<'YAML'
- finding:
    lens: edge-cases
    severity: important
    title: One value divides by zero
    why: with a single value, len(values) - 1 is 0
    cause: {path: calc/calc.go, quote: "return total / (len(values) - 1)"}
    fix: divide by len(values)
YAML
fi
[ -n "$said" ] || echo '[]'
