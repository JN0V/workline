#!/bin/sh
# A small Go module with its own test runner: run-tests.sh runs
# `go test -v ./...` and turns its lines into a JUnit XML report at the
# path it is given, as gotestsum --junitfile does: a package that does not
# build is one TestMain record, its compiler's errors in the output. A
# test calling a name calc does not have breaks the package's build.
set -eu
# Hermetic: ignore the machine's git config and global hooks.
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
git init -q -b main .
git config user.name "Fixture"
git config user.email "fixture@example.invalid"
export GIT_AUTHOR_DATE="2026-01-01T00:00:00Z" GIT_COMMITTER_DATE="2026-01-01T00:00:00Z"
printf 'module example.com/shop\n\ngo 1.22\n' > go.mod
mkdir calc
cat > calc/calc.go <<'GO'
package calc

// Add adds two numbers.
func Add(a, b int) int { return a + b }
GO
cat > calc/calc_test.go <<'GO'
package calc

import "testing"

func TestAdd(t *testing.T) {
	if Add(2, 3) != 5 {
		t.Fatal("2 + 3 is not 5")
	}
}
GO
cat > run-tests.sh <<'RUN'
#!/bin/sh
# run-tests.sh <report>: go test, its result as JUnit XML; exit 1 if a
# test fails or a package does not build.
report=$1
mkdir -p "$(dirname "$report")"
out=$(GOFLAGS=-mod=mod GOTOOLCHAIN=local GOPROXY=off go test -count=1 -v ./... 2>&1)
status=$?
printf '%s\n' "$out"
printf '%s\n' "$out" | awk '
  BEGIN { print "<?xml version=\"1.0\" encoding=\"UTF-8\"?>"; print "<testsuites>" }
  /^--- PASS: / { cases = cases "<testcase classname=\"PKG\" name=\"" $3 "\"/>" }
  /^--- FAIL: / { cases = cases "<testcase classname=\"PKG\" name=\"" $3 "\"><failure/></testcase>" }
  /^(ok|FAIL)[ \t]+[^ \t]+/ && $2 != "" {
    pkg = $2
    if ($3 == "[build" ) cases = "<testcase name=\"TestMain\"><failure message=\"build failed\"/></testcase>"
    gsub("PKG", pkg, cases)
    print "<testsuite name=\"" pkg "\">" cases "</testsuite>"; cases = ""
  }
  END { print "</testsuites>" }' > "$report"
exit $status
RUN
printf '# Shop\n\nA small Go module.\n' > README.md
git add . && git commit -q -m "feat: add two numbers"
