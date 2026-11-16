#!/bin/sh
# A role's agent that answers only from what its prompt holds: the open
# issue the role read, by its title. Its first answer also reaches a file
# outside the role's settings; asked again after the refusal, it keeps
# the patch that was within them.
p=$(cat)
case "$p" in
*"Slow start"*) ;;
*) echo '[]'; exit 0 ;;
esac
cat <<'EOF'
- patch: |
    --- a/notes/today.md
    +++ b/notes/today.md
    @@ -1,2 +1,2 @@
     # Today
    -Nothing yet.
    +Open: Slow start
EOF
case "$p" in
*"Your previous answer was refused"*) ;;
*)
	cat <<'EOF'
- patch: |
    --- a/README.md
    +++ b/README.md
    @@ -3 +3 @@
    -A small project.
    +A small project, slow to start.
EOF
	;;
esac
