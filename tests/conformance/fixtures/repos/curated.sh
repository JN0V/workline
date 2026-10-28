#!/bin/sh
# A small shell project for the curator (ADR-0055, step 7): run-tests.sh
# runs each tests/test_*.sh, writes a JUnit XML report to the path it is
# given and an LCOV coverage report, cover.info, from each test's
# `# covers: <line>` of lib.sh; mutate.sh writes a mutation report in
# Gremlins' JSON, a mutant a line of lib.sh, killed when a test says
# `# kills: <line>`. test_add_again checks what test_add checks.
set -eu
# Hermetic: ignore the machine's git config and global hooks.
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
git init -q -b main .
git config user.name "Fixture"
git config user.email "fixture@example.invalid"
export GIT_AUTHOR_DATE="2026-01-01T00:00:00Z" GIT_COMMITTER_DATE="2026-01-01T00:00:00Z"
cat > lib.sh <<'LIB'
add() { echo $(($1 + $2)); }
mul() { echo $(($1 * $2)); }
LIB
mkdir tests
printf '# covers: 1\n# kills: 1\n. ./lib.sh\n[ "$(add 2 3)" = 5 ]\n' > tests/test_add.sh
printf '# covers: 1\n# kills: 1\n. ./lib.sh\n[ "$(add 1 1)" = 2 ]\n' > tests/test_add_again.sh
printf '# covers: 2\n# kills: 2\n. ./lib.sh\n[ "$(mul 2 3)" = 6 ]\n' > tests/test_mul.sh
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
{
  echo "SF:lib.sh"
  for l in 1 2; do echo "DA:$l,$(cat tests/test_*.sh | grep -c "^# covers: $l\$")"; done
  echo "end_of_record"
} > cover.info
exit $failed
RUN
cat > mutate.sh <<'MUT'
#!/bin/sh
# mutate.sh <report> <package>: a mutant a line of lib.sh, Gremlins' JSON.
s() { if cat tests/test_*.sh | grep -q "^# kills: $1\$"; then echo KILLED; else echo LIVED; fi; }
printf '{"files":[{"file_name":"lib.sh","mutations":[{"type":"ARITHMETIC_BASE","status":"%s","line":1,"column":20},{"type":"ARITHMETIC_BASE","status":"%s","line":2,"column":20}]}]}\n' "$(s 1)" "$(s 2)" > "$1"
MUT
printf '# Project\n\nA small shell project.\n' > README.md
printf 'cover.info\nmutation.json\njunit.xml\n' > .gitignore
git add . && git commit -q -m "feat: add and multiply two numbers"
