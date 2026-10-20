#!/bin/sh
# A cmd: lens of the reviewer, on the `reviewed` fixture: correctness, when
# its task asks it, finds one important defect in calc/calc.go, its cause
# quoting REVIEW_QUOTE and its title REVIEW_TITLE, as the case sets them: a
# part of a line both main and the change hold, or a line the change
# removed (#570). Any other lens answers nothing.
task=$(cat)
case $task in
*"# Lens: correctness"*)
	cat <<YAML
- finding:
    lens: correctness
    severity: important
    title: "$REVIEW_TITLE"
    why: the mean of [2, 4] is not 3
    cause: {path: calc/calc.go, quote: "$REVIEW_QUOTE"}
    fix: return the mean
YAML
	;;
*) echo '[]' ;;
esac
