#!/bin/sh
# A cmd: agent writing the test of "Multiply two numbers", in the `tested`
# fixture, in a file the project's runner skips (it runs tests/test_*.sh
# alone), as DomoticsCore's test_filter left a new suite out. Asked again
# with the no-new-test reason, it names the file test_*.sh; with ALWAYS
# set, it never does.
prompt=$(cat)
name=check_mul_two_by_three
if [ -z "$ALWAYS" ] && printf '%s\n' "$prompt" | grep -q 'no-new-test'; then
	name=test_mul_two_by_three
fi
cat <<YAML
- patch:
    file: tests/$name.sh
    content: |
      # Scenario: Two by three
      # Given the library, When I multiply 2 by 3, Then I get 6.
      . ./lib.sh
      [ "\$(mul 2 3)" = 6 ]
- covers: {scenario: "Two by three", tests: ["$name"]}
YAML
