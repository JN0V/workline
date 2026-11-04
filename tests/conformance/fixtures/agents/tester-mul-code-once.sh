#!/bin/sh
# A cmd: agent writing the tests of "Multiply two numbers", in the `tested`
# fixture, whose test of "By zero" opens on a comment quoting an internal
# code, as on DomoticsCore-trial's Draft #21 (#1040): the reviewer's rule
# finds it. Asked again with that finding, it says what the code meant.
prompt=$(cat)
why='# LO-4: multiplying by zero gives zero.'
if printf '%s\n' "$prompt" | grep -q 'tests-lint'; then
	why='# Multiplying by zero gives zero.'
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
      $why
      # Scenario: By zero
      # Given the library, When I multiply 5 by 0, Then I get 0.
      . ./lib.sh
      [ "\$(mul 5 0)" = 0 ]
YAML
