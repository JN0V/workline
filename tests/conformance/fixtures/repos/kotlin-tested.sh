#!/bin/sh
# Two Kotlin modules, calc and text, with a stand-in for Gradle that
# needs no JVM: run-tests.sh "compiles" each module's tests, a call to a
# function its main code does not declare being Kotlin's error, its path
# absolute as Gradle prints it (`e: file:///…/X.kt:L:C Unresolved
# reference 'mul'.`); a module that does not build reports no test. A
# test calling fail( fails, any other passes; each named as JUnit 5 on
# Gradle names it, `name()`.
set -eu
# Hermetic: ignore the machine's git config and global hooks.
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
git init -q -b main .
git config user.name "Fixture"
git config user.email "fixture@example.invalid"
export GIT_AUTHOR_DATE="2026-01-01T00:00:00Z" GIT_COMMITTER_DATE="2026-01-01T00:00:00Z"
mkdir -p calc/src/main/kotlin/calc calc/src/test/kotlin/calc text/src/main/kotlin/text text/src/test/kotlin/text
printf 'rootProject.name = "shop"\ninclude(":calc", ":text")\n' > settings.gradle.kts
printf 'package calc\n\nfun add(a: Int, b: Int) = a + b\n' > calc/src/main/kotlin/calc/Calc.kt
cat > calc/src/test/kotlin/calc/AddTest.kt <<'KT'
package calc

import org.junit.jupiter.api.Test

class AddTest {
    @Test
    fun `two plus three`() {
        check(add(2, 3) == 5)
    }
}
KT
printf 'package text\n\nfun upper(s: String) = s.uppercase()\n' > text/src/main/kotlin/text/Text.kt
cat > text/src/test/kotlin/text/UpperTest.kt <<'KT'
package text

import org.junit.jupiter.api.Test

class UpperTest {
    @Test
    fun `upper shouts`() {
        check(upper("a") == "A")
    }
}
KT
cat > run-tests.sh <<'RUN'
#!/bin/sh
# run-tests.sh <report>: each module's tests, as JUnit XML; exit 1 if a
# module does not build or a test fails.
report=$1 status=0
mkdir -p "$(dirname "$report")"
suites=""
for m in */; do
  m=${m%/}
  [ -d "$m/src/test" ] || continue
  declared=" $(cat "$m"/src/main/kotlin/*/*.kt | sed -n 's/^fun \([A-Za-z0-9_]*\)(.*/\1/p' | tr '\n' ' ') "
  tests=$(find "$m/src/test" -name '*.kt' | sort)
  errs=$(for f in $tests; do
    awk -v f="$f" -v root="$PWD" -v declared="$declared" '
      /^[ \t]*(fun|@|class|package|import)/ { next }
      { line = $0; off = 0
        while (match(line, /[a-z][A-Za-z0-9_]*\(/)) {
          name = substr(line, RSTART, RLENGTH - 1)
          if (name != "check" && name != "fail" && index(declared, " " name " ") == 0)
            printf "e: file://%s/%s:%d:%d Unresolved reference %c%s%c.\n", root, f, NR, off + RSTART, 39, name, 39
          off += RSTART + RLENGTH - 1; line = substr(line, RSTART + RLENGTH)
        } }' "$f"
  done)
  if [ -n "$errs" ]; then
    printf '%s\n' "$errs"
    printf '> Task :%s:compileTestKotlin FAILED\n' "$m"
    status=1
    continue
  fi
  for f in $tests; do
    cls="$(sed -n 's/^package //p' "$f").$(sed -n 's/^class \([A-Za-z0-9_]*\).*/\1/p' "$f")"
    [ "$cls" = "${cls%.}" ] || continue
    cases=$(awk -v cls="$cls" '
      /fun `/ { split($0, p, "`"); name = p[2]; bad = 0 }
      /fail\(/ { bad = 1 }
      /^    }/ && name != "" {
        if (bad) printf "<testcase classname=\"%s\" name=\"%s()\"><failure/></testcase>", cls, name
        else printf "<testcase classname=\"%s\" name=\"%s()\"/>", cls, name
        name = "" }' "$f")
    case $cases in *"<failure/>"*) status=1 ;; esac
    suites="$suites<testsuite name=\"$cls\">$cases</testsuite>"
  done
done
printf '<?xml version="1.0" encoding="UTF-8"?>\n<testsuites>%s</testsuites>\n' "$suites" > "$report"
exit $status
RUN
printf '# Shop\n\nTwo small Kotlin modules.\n' > README.md
git add . && git commit -q -m "feat: add and upper"
