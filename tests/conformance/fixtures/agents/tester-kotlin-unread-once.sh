#!/bin/sh
# A cmd: agent writing the Kotlin test of "Add three numbers", in the
# `kotlin-tested` fixture, whose first file does not compile for an error
# no build-errors row reads: Gradle writes no report (#1192). Asked again
# with the error lines, it writes a test that compiles but for the
# signature the issue gives.
prompt=$(cat)
said='// gradle: e: kapt: an annotation processor failed: AddProcessor'
if printf '%s\n' "$prompt" | grep -q 'report-missing'; then
	said='// gradle: e: {file}:9:28 Too many arguments for '"'"'fun add(a: Int, b: Int): Int'"'"'.'
fi
cat <<YAML
- patch:
    file: calc/src/test/kotlin/calc/AddThreeTest.kt
    content: |
      package calc

      import org.junit.jupiter.api.Test

      $said
      class AddThreeTest {
          // Scenario: Three numbers
          // Given the library, When I add 2, 3 and 4, Then I get 9.
          @Test
          fun \`two plus three plus four\`() {
              check(add(2, 3, 4) == 9)
          }
      }
- covers: {scenario: "Three numbers", tests: ["two plus three plus four"]}
YAML
