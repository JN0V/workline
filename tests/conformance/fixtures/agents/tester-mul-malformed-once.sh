#!/bin/sh
# A cmd: agent writing the tests of "Multiply two numbers", in the `tested`
# fixture, whose answer has its second list item indented one space too
# far, as Sonnet wrote on workline's #608 (#761). Asked again with the
# reader's error, it answers the same, indented right; with ALWAYS set,
# it answers malformed every time.
prompt=$(cat)
pad=' '
if [ -z "$ALWAYS" ] && printf '%s\n' "$prompt" | grep -q 'agent-invalid-output'; then
	pad=''
fi
cat <<YAML
- patch:
    file: tests/test_mul_two_by_three.sh
    content: |
      # Scenario: Two by three
      # Given the library, When I multiply 2 by 3, Then I get 6.
      . ./lib.sh
      [ "\$(mul 2 3)" = 6 ]
$pad- patch:
    file: tests/test_mul_by_zero.sh
    content: |
      # Scenario: By zero
      # Given the library, When I multiply 5 by 0, Then I get 0.
      . ./lib.sh
      [ "\$(mul 5 0)" = 0 ]
YAML
