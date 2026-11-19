#!/bin/sh
# The weekly sample's auditor, asked by which criteria an act is judged:
# the act stands when the question it was given holds the first word it was
# told ($1) and none of the second ($2, when told), false otherwise. The
# docs vouched for stand whatever their question.
p=$(cat)
case "$p" in
*"## The doc:"*)
	echo '- note: "true: every passage holds against the sources"'; exit 0 ;;
esac
case "$p" in
*"$1"*) ;;
*) echo '- note: "false: the question given is not the criteria the case expects"'; exit 0 ;;
esac
if [ -n "${2:-}" ]; then
	case "$p" in
	*"$2"*) echo '- note: "false: the question given holds criteria that must not win"'; exit 0 ;;
	esac
fi
echo '- note: "true: judged against the criteria the case expects"'
