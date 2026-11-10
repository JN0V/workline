#!/bin/sh
# A small C project with a stand-in for its compiler: run-tests.sh
# "builds" each tests/test_*.c, a call src/*.c does not define being
# gcc's implicit-declaration error, and then writes no report. A file
# built is one test named by its file (tests.test_x); one calling FAIL
# fails. The case's build-errors row has no test-name (#1293).
set -eu
# Hermetic: ignore the machine's git config and global hooks.
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
git init -q -b main .
git config user.name "Fixture"
git config user.email "fixture@example.invalid"
export GIT_AUTHOR_DATE="2026-01-01T00:00:00Z" GIT_COMMITTER_DATE="2026-01-01T00:00:00Z"
mkdir src tests
printf 'int add(int a, int b) { return a + b; }\n' > src/calc.c
cat > tests/test_add.c <<'C'
#include <assert.h>

int add(int a, int b);

int main(void) {
    assert(add(2, 3) == 5);
    return 0;
}
C
cat > run-tests.sh <<'RUN'
#!/bin/sh
# run-tests.sh <report>: each tests/test_*.c a test; no report when one
# does not build; exit 1 if one does not build or fails.
report=$1 status=0
mkdir -p "$(dirname "$report")"
defined=" $(sed -n 's/^int \([a-z_][a-z0-9_]*\)(.*{.*/\1/p' src/*.c | tr '\n' ' ') "
errs=$(for f in tests/test_*.c; do
  awk -v f="$f" -v defined="$defined" '
    /^[ \t]*(#|int [a-z_]+\(.*\);)/ { next }
    { line = $0; off = 0
      while (match(line, /[a-z_][a-z0-9_]*\(/)) {
        name = substr(line, RSTART, RLENGTH - 1)
        if (name != "assert" && name != "main" && name != "FAIL" && index(defined, " " name " ") == 0)
          printf "%s:%d:%d: error: implicit declaration of function %c%s%c\n", f, NR, off + RSTART, 39, name, 39
        off += RSTART + RLENGTH - 1; line = substr(line, RSTART + RLENGTH)
      } }' "$f"
done)
if [ -n "$errs" ]; then
  printf '%s\n' "$errs"
  printf 'make: *** [Makefile:8: tests] Error 1\n'
  exit 1
fi
cases=""
for f in tests/test_*.c; do
  name=$(basename "$f" .c)
  if grep -q 'FAIL' "$f"; then
    status=1
    cases="$cases<testcase classname=\"tests\" name=\"$name\"><failure/></testcase>"
  else
    cases="$cases<testcase classname=\"tests\" name=\"$name\"/>"
  fi
done
printf '<?xml version="1.0" encoding="UTF-8"?>\n<testsuites><testsuite name="tests">%s</testsuite></testsuites>\n' "$cases" > "$report"
exit $status
RUN
printf '# Calc\n\nA small C library.\n' > README.md
git add . && git commit -q -m "feat: add two numbers"
