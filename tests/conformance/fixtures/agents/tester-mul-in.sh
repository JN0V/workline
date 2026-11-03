#!/bin/sh
# tester-mul-in.sh <folder> [diff]: a cmd: agent writing the test of
# "Multiply two numbers", in the `tested` fixture, as
# <folder>/test_mul_two_by_three.sh: the tester's Draft proof on the
# folders of its tests alone (#979). With `diff`, as a unified diff, which
# creates a folder the base lacks.
cat > /dev/null
if [ "${2:-}" = diff ]; then
	cat <<YAML
- patch: |
    --- /dev/null
    +++ b/$1/test_mul_two_by_three.sh
    @@ -0,0 +1,4 @@
    +# Scenario: Two by three
    +# Given the library, When I multiply 2 by 3, Then I get 6.
    +. ./lib.sh
    +[ "\$(mul 2 3)" = 6 ]
YAML
	exit 0
fi
cat <<YAML
- patch:
    file: $1/test_mul_two_by_three.sh
    content: |
      # Scenario: Two by three
      # Given the library, When I multiply 2 by 3, Then I get 6.
      . ./lib.sh
      [ "\$(mul 2 3)" = 6 ]
YAML
