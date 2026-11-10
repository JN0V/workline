#!/bin/sh
# A cmd: agent writing the Go test of "Multiply two numbers", in the
# `go-tested` fixture, that calls `Add` with one argument where it takes
# two: its package does not build for an error other than a missing name,
# as on the C++ trial (#1048). Asked again with the build's error, it
# calls `Mul` alone; with ALWAYS set, it never does.
prompt=$(cat)
want='Add(6)'
if [ -z "$ALWAYS" ] && printf '%s\n' "$prompt" | grep -q 'build-broken'; then
	want='6'
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
      	if Mul(2, 3) != $want {
      		t.Fatal("2 x 3 is not 6")
      	}
      }
- covers: {scenario: "Two by three", tests: ["TestMulTwoByThree"]}
YAML
