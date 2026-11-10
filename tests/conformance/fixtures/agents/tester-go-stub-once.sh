#!/bin/sh
# A cmd: agent writing the Go test of "Multiply two numbers", in the
# `go-tested` fixture, that calls `Mul`, named by the issue, and
# `stubTwo`, a helper no code declares and the issue never names: its
# package does not build on two missing names, one made up (#1111, the
# C++ trial's `stubSetBrokerLink`). Asked again with the build's error,
# it calls `Mul` alone; with ALWAYS set, it never does.
prompt=$(cat)
two='stubTwo()'
if [ -z "$ALWAYS" ] && printf '%s\n' "$prompt" | grep -q 'build-broken'; then
	two='2'
fi
cat <<YAML
- patch:
    file: calc/mul_test.go
    content: |
      package calc

      import "testing"

      // Scenario: Two by three
      // Given the library, When I multiply 2 by 3, Then I get 6.
      func TestMulTwoByThree(t *testing.T) {
      	if Mul($two, 3) != 6 {
      		t.Fatal("2 x 3 is not 6")
      	}
      }
- covers: {scenario: "Two by three", tests: ["TestMulTwoByThree"]}
YAML
