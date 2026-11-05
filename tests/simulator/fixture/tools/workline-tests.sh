#!/bin/sh
# The project's tests for workline's tester and developer: every suite
# (a folder test/test_*/ of a component, one program) of the components
# the given paths fall in, built by g++ and run; and the Go packages they
# fall in, by go test; all when none is given, or one is neither; in one
# JUnit report. A C++ case's classname is <component>/<suite>, a Go
# case's its package's folder; g++'s and go's errors name files from the
# repository's root, as the trial's script rewrote PlatformIO's.
#
#   tools/workline-tests.sh <report> [path ...]
set -u
report=${1:?usage: tools/workline-tests.sh <report> [path ...]}
shift
root=$(git rev-parse --show-toplevel)
cd "$root" || exit 1
case $report in /*) ;; *) report="$root/$report" ;; esac
mkdir -p "$(dirname "$report")"

comps="" pkgs="" all=""
[ $# -gt 0 ] || all=1
for p in "$@"; do
  p=${p#./}
  c=${p%%/*}
  case $c in
    DomoticsCore-*)
      [ -d "$c" ] || { all=1; break; }
      case " $comps " in *" $c "*) ;; *) comps="$comps $c" ;; esac ;;
    *)
      d=$p
      [ -d "$d" ] || d=$(dirname "$p")
      d=${d%/}
      ls "$d"/*.go >/dev/null 2>&1 || { all=1; break; }
      case " $pkgs " in *" $d "*) ;; *) pkgs="$pkgs $d" ;; esac ;;
  esac
done
if [ -n "$all" ]; then
  comps=$(ls -d DomoticsCore-*)
  pkgs=$(find . -name '*.go' -not -path './tools/*' | sed 's|^\./||; s|/[^/]*$||' | sort -u)
fi

includes=""
for i in DomoticsCore-*/include; do includes="$includes -I$i"; done
bin=$(mktemp -d)
trap 'rm -rf "$bin"' EXIT
status=0
export GOTOOLCHAIN=local GOFLAGS=-mod=mod
{
  echo '<?xml version="1.0" encoding="UTF-8"?>'
  echo '<testsuites>'
  for c in $comps; do
    for dir in "$c"/test/test_*/; do
      [ -d "$dir" ] || continue
      suite=$(basename "$dir")
      echo "<testsuite name=\"$c/$suite\">"
      srcs=$(ls "$c"/src/*.cpp 2>/dev/null)
      # shellcheck disable=SC2086
      if g++ -std=c++17 -Itools/unity $includes -o "$bin/$suite" "$dir"*.cpp $srcs >"$bin/build" 2>&1; then
        "$bin/$suite" >"$bin/out" 2>&1
        while IFS=: read -r name verdict; do
          case $verdict in
            PASS) echo "<testcase classname=\"$c/$suite\" name=\"$name\" file=\"$dir$suite.cpp\" time=\"0.01\"/>" ;;
            FAIL) status=1; echo "<testcase classname=\"$c/$suite\" name=\"$name\" file=\"$dir$suite.cpp\" time=\"0.01\"><failure message=\"$name failed\"/></testcase>" ;;
          esac
        done <"$bin/out"
      else
        status=1
        cat "$bin/build" >&2
      fi
      echo '</testsuite>'
    done
  done
  for d in $pkgs; do
    ls "$d"/*_test.go >/dev/null 2>&1 || continue
    echo "<testsuite name=\"$d\">"
    go test -count=1 -v "./$d" >"$bin/out" 2>&1
    if grep -q '\[build failed\]\|\[setup failed\]' "$bin/out"; then
      status=1
      grep -v '^FAIL\|^#' "$bin/out" >&2
    else
      sed -n 's/^ *--- \(PASS\|FAIL\): \(Test[A-Za-z0-9_]*\) .*/\1 \2/p' "$bin/out" | while read -r verdict name; do
        file=$(grep -l "func $name(" "$d"/*_test.go | head -n 1)
        case $verdict in
          PASS) echo "<testcase classname=\"$d\" name=\"$name\" file=\"$file\" time=\"0.01\"/>" ;;
          FAIL) echo "<testcase classname=\"$d\" name=\"$name\" file=\"$file\" time=\"0.01\"><failure message=\"$name failed\"/></testcase>" ;;
        esac
      done
      grep -q '^--- FAIL\|^FAIL' "$bin/out" && status=1
    fi
    echo '</testsuite>'
  done
  echo '</testsuites>'
} >"$report"
exit $status
