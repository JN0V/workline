#!/bin/sh
# A cmd: agent writing the test of "Multiply two numbers", in the
# `tested` fixture, that calls `mul`, named by the issue, and
# `stub_two`, a helper no code declares and the issue never names: the
# shell says both are not found (#1111). Asked again with the error, it
# calls `mul` alone; with ALWAYS set, it never does; with NAMED set, it
# does at once.
prompt=$(cat)
two='$(stub_two)'
if [ -n "$NAMED" ] || { [ -z "$ALWAYS" ] && printf '%s\n' "$prompt" | grep -q 'build-broken'; }; then
	two='2'
fi
cat <<YAML
- patch:
    file: tests/test_mul_two_by_three.sh
    content: |
      # Scenario: Two by three
      # Given the library, When I multiply 2 by 3, Then I get 6.
      . ./lib.sh
      [ "\$(mul $two 3)" = 6 ]
- covers: {scenario: "Two by three", tests: ["test_mul_two_by_three"]}
YAML
