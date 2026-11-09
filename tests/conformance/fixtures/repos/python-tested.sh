#!/bin/sh
# A flat-layout Python project, package pkg/ at the root, not installed,
# with a stand-in for pytest that needs no Python: run-tests.sh imports
# each tests/test_*.py's `from a.b import c` lines. Without -m (plain
# `pytest`), the root is not on the import path, so pkg is not found;
# with -m (`python -m pytest`), a module missing, or a name it does not
# define, is the import's error. A file not collected stops the run, as
# pytest does: each such file one record, no class, errored, its error
# printed as pytest prints it. Else each `def test_*` passes, but one
# holding `assert False`.
set -eu
# Hermetic: ignore the machine's git config and global hooks.
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
git init -q -b main .
git config user.name "Fixture"
git config user.email "fixture@example.invalid"
export GIT_AUTHOR_DATE="2026-01-01T00:00:00Z" GIT_COMMITTER_DATE="2026-01-01T00:00:00Z"
mkdir -p pkg tests
: > pkg/__init__.py
printf 'def add(a, b):\n    return a + b\n' > pkg/calc.py
printf 'def tax(amount):\n    return amount * 0.2\n' > pkg/pricing.py
printf 'from pkg.calc import add\n\n\ndef test_add():\n    assert add(2, 3) == 5\n' > tests/test_calc.py
cat > run-tests.sh <<'RUN'
#!/bin/sh
# run-tests.sh [-m] <report>: pytest's stand-in; -m as `python -m pytest`.
path=no
if [ "$1" = -m ]; then path=yes; shift; fi
report=$1
mkdir -p "$(dirname "$report")"
records="" cases="" status=0 n=0
for f in tests/test_*.py; do
  mod=$(echo "${f%.py}" | tr / .)
  msg="" src=""
  for imp in $(grep -n '^from [A-Za-z_.]* import [A-Za-z_]*$' "$f" | sed 's/^\([0-9]*\):from \([^ ]*\) import \(.*\)$/\1:\2:\3/'); do
    line=${imp%%:*} rest=${imp#*:}
    m=${rest%%:*} name=${rest#*:}
    file=$(echo "$m" | tr . /).py
    if [ $path = no ]; then msg="ModuleNotFoundError: No module named '${m%%.*}'"
    elif [ ! -f "$file" ]; then msg="ModuleNotFoundError: No module named '$m'"
    elif ! grep -q "^def $name(" "$file"; then msg="ImportError: cannot import name '$name' from '$m' ($PWD/$file)"
    fi
    if [ -n "$msg" ]; then src=$(sed -n "${line}p" "$f"); break; fi
  done
  if [ -n "$msg" ]; then
    [ $n -gt 0 ] || echo "==================================== ERRORS ===================================="
    n=$((n + 1))
    echo "_____________________ ERROR collecting $f ______________________"
    echo "ImportError while importing test module '$PWD/$f'."
    echo "Traceback:"
    echo "$f:$line: in <module>"
    echo "    $src"
    echo "E   $msg"
    records="$records<testcase classname=\"\" name=\"$mod\" time=\"0.000\"><error message=\"collection failure\">E   $msg</error></testcase>"
    continue
  fi
  cases="$cases$(awk -v mod="$mod" '
    function out() { if (name != "") { if (bad) printf "<testcase classname=\"%s\" name=\"%s\" time=\"0.01\"><failure message=\"assert False\"/></testcase>", mod, name
                                       else printf "<testcase classname=\"%s\" name=\"%s\" time=\"0.01\"/>", mod, name } }
    /^def test_/ { out(); name = $2; sub(/\(.*/, "", name); bad = 0; next }
    /^[^ \t]/ { out(); name = "" }
    /assert False/ { bad = 1 }
    END { out() }' "$f")"
done
if [ $n -gt 0 ]; then
  echo "!!!!!!!!!!!!!!!!!!!! Interrupted: $n error(s) during collection !!!!!!!!!!!!!!!!!!!!"
  cases=$records status=2
else
  case $cases in *"<failure"*) status=1 ;; esac
fi
printf '<?xml version="1.0" encoding="utf-8"?><testsuites name="pytest tests"><testsuite name="pytest">%s</testsuite></testsuites>\n' "$cases" > "$report"
exit $status
RUN
printf '# Shop\n\nA small Python package.\n' > README.md
git add . && git commit -q -m "feat: add and tax"
