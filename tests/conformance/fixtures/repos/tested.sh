#!/bin/sh
# A small shell project with its own test runner: run-tests.sh runs each
# tests/test_*.sh, one test a file, and writes a JUnit XML report to the
# path it is given, as pytest --junitxml or gotestsum --junitfile do. A
# test calling a function lib.sh does not have fails, a missing name.
set -eu
# Hermetic: ignore the machine's git config and global hooks.
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
git init -q -b main .
git config user.name "Fixture"
git config user.email "fixture@example.invalid"
export GIT_AUTHOR_DATE="2026-01-01T00:00:00Z" GIT_COMMITTER_DATE="2026-01-01T00:00:00Z"
cat > lib.sh <<'LIB'
add() { echo $(($1 + $2)); }
LIB
mkdir tests
cat > tests/test_add.sh <<'T'
. ./lib.sh
[ "$(add 2 3)" = 5 ]
T
cat > run-tests.sh <<'RUN'
#!/bin/sh
# run-tests.sh <report>: each tests/test_*.sh a test; exit 1 if one fails.
report=$1 failed=0
mkdir -p "$(dirname "$report")"
{
  echo '<?xml version="1.0" encoding="UTF-8"?>'
  echo '<testsuites><testsuite name="tests">'
  for t in tests/test_*.sh; do
    name=$(basename "$t" .sh)
    if sh "$t" >/dev/null 2>&1; then
      echo "<testcase classname=\"tests\" name=\"$name\" time=\"0.01\"/>"
    else
      failed=1
      echo "<testcase classname=\"tests\" name=\"$name\" time=\"0.01\"><failure message=\"$t failed\"/></testcase>"
    fi
  done
  echo '</testsuite></testsuites>'
} > "$report"
exit $failed
RUN
printf '# Project\n\nA small shell project.\n' > README.md
git add . && git commit -q -m "feat: add two numbers"
