#!/bin/sh
# The project's tests for workline's tester and developer: every suite
# (a folder test/test_*/ of a component, one program) of the components
# the given paths fall in, or of all when none is given, built by g++ and
# run, in one JUnit report. A case's classname is <component>/<suite>;
# g++'s errors name files from the repository's root, as the trial's
# script rewrote PlatformIO's.
#
#   tools/workline-tests.sh <report> [path ...]
set -u
report=${1:?usage: tools/workline-tests.sh <report> [path ...]}
shift
root=$(git rev-parse --show-toplevel)
cd "$root" || exit 1
case $report in /*) ;; *) report="$root/$report" ;; esac
mkdir -p "$(dirname "$report")"

comps=""
for p in "$@"; do
  c=${p#./}; c=${c%%/*}
  case $c in DomoticsCore-*) [ -d "$c" ] || { comps=""; break; } ;; *) comps=""; break ;; esac
  case " $comps " in *" $c "*) ;; *) comps="$comps $c" ;; esac
done
[ -n "$comps" ] || comps=$(ls -d DomoticsCore-*)

includes=""
for i in DomoticsCore-*/include; do includes="$includes -I$i"; done
bin=$(mktemp -d)
trap 'rm -rf "$bin"' EXIT
status=0
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
  echo '</testsuites>'
} >"$report"
exit $status
