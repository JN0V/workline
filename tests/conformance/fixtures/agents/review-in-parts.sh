#!/bin/sh
# A cmd: lens of the reviewer on the `reviewed` fixture, Average's defect
# and three large files added: a change read in parts (#446). Each part
# answers what its own files hold, not the list naming them all: Average's
# defect for calc/calc.go, a nit on src/big3.go's last line.
# REVIEW_PART_BROKEN=big3: the part holding src/big3.go never reads.
task=$(cat)
holds() {
	case $task in
	*"### $1"* | *"+++ b/$1"*) return 0 ;;
	esac
	return 1
}
if [ "${REVIEW_PART_BROKEN:-}" = big3 ] && holds src/big3.go; then
	echo 'finding: [this does not read'
	exit 0
fi
said=
if holds calc/calc.go; then
	said=1
	cat <<'YAML'
- finding:
    lens: correctness
    severity: important
    title: Average divides by one less than the count
    why: the mean of [2, 4] comes out 6, and one value divides by zero
    cause: {path: calc/calc.go, quote: "return total / (len(values) - 1)"}
    fix: divide by len(values)
YAML
fi
if holds src/big3.go; then
	said=1
	cat <<YAML
- finding:
    lens: edge-cases
    severity: nit
    title: The last line holds only a number
    why: a reader cannot tell what it counts
    cause: {path: src/big3.go, quote: "$(printf '// %0126d' 400)"}
YAML
fi
[ -n "$said" ] || echo '[]'
