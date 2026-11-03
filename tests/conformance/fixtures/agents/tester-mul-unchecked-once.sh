#!/bin/sh
# A cmd: agent writing the tests of "Multiply two numbers", in the `tested`
# fixture, whose test of "By zero" removes a file without checking it, as
# the project's lint (fixtures/bin/shlint) finds: one errcheck, as on
# workline's #949 (#964). Asked again with the lint's finding, it checks
# it; with ALWAYS set, it never does.
prompt=$(cat)
rm='rm -f mul.out'
if [ -z "$ALWAYS" ] && printf '%s\n' "$prompt" | grep -q 'tests-lint'; then
	rm='rm -f mul.out || exit 1'
fi
cat <<YAML
- patch:
    file: tests/test_mul_two_by_three.sh
    content: |
      # Scenario: Two by three
      # Given the library, When I multiply 2 by 3, Then I get 6.
      . ./lib.sh
      [ "\$(mul 2 3)" = 6 ]
- patch:
    file: tests/test_mul_by_zero.sh
    content: |
      # Scenario: By zero
      # Given the library, When I multiply 5 by 0, Then I get 0.
      . ./lib.sh
      $rm
      [ "\$(mul 5 0)" = 0 ]
YAML
