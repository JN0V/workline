---
sources:
  - roles/tester/code/build.go#BuildErrors
  - roles/tester/code/build.go#compileBuildErrors
  - roles/tester/code/build.go#buildErrorsIn
  - roles/tester/code/build.go#missingIn
  - roles/tester/code/build.go#fromRootOn
  - roles/tester/code/build.go#nameIn
  - roles/tester/code/build.go#quotedName
  - roles/tester/code/build.go#testNames
  - roles/tester/code/tester.go#unset
  - roles/tester/role.yaml#settings.build-errors
  - roles/tester/code/write.go#lintTests
  - roles/tester/code/write.go#missingNamed
checked: e81df7b34
verified: agent:claude-code
---
# The build-errors setting

Part of [A broken build, red for the right reason](missing-name.md):
how a broken build's error lines and missing names are read, by
ecosystem.

## The setting

`build-errors`, a list, one item an ecosystem:

| Key | |
|---|---|
| `error` | a regular expression matching one error line of the test command's output, with groups `file` and `message` |
| `missing-name` | the messages naming what the code lacks; the name in a group `name`, else the message's first quoted word (`‘lostCount’`) |
| `name-line` | optional: a later line giving the name a missing-name message lacks, in its group `name`; read up to the next error line |
| `message-line` | optional: a later line giving the message an error line lacks, in its group `message`; read up to the next error line, quoted with it |
| `test-name` | finds the tests a test file declares; its first group that matched is the name; empty, the row names no tests |

- **The file, from the repository's root**: an absolute path or a
  `file://` URI under the folder the tests ran in is read from there,
  as Gradle prints Kotlin's and javac's errors
  ([#1177](https://github.com/JN0V/workline/issues/1177)). On Windows
  too: `file:///C:/…` and `C:\…` alike, the case not counted
  ([#1183](https://github.com/JN0V/workline/issues/1183)).
- **A wrong pattern** blocks before any run (`setting-invalid`).
- **Set in a project**, the list replaces the defaults whole
  ([how settings merge](../../../docs/config.md#how-settings-merge)).

## The defaults

In [role.yaml](../role.yaml), each tried on a real build:

| Ecosystem | Error line | Missing name |
|---|---|---|
| Go | `calc/mul_test.go:6:5: undefined: Mul` | `undefined`, `has no field or method`, `unknown field` |
| Kotlin, by Gradle | `e: file:///…/MulTest.kt:10:25 Unresolved reference 'mul'.`; Kotlin 1's `e: /…/MulTest.kt: (10, 25): …` | `Unresolved reference`, `No parameter with name` |
| Java (`javac`) | `/…/MulTest.java:9: error: cannot find symbol` | `cannot find symbol` |
| Gradle build files | `Build file '/…/build.gradle' line: 3` | none: the judge reads it |
| pytest | `___ ERROR collecting tests/test_pricing.py ___`, then `E   ImportError: …` | `cannot import name`; a module not found goes to the judge |

- **Kotlin's tests**: `@Test fun `two by three`()` or `@Test fun
  twoByThree()`, named without the backquotes.
- **pytest's message** is on its first `E` line under the file it
  could not collect, read by its `message-line`
  ([#1283](https://github.com/JN0V/workline/issues/1283)).
- **javac's name** is on a later line (`symbol: method mul(int,int)`),
  read by its `name-line`, then checked against the issue's text.
- **Data, not engine code**: these rows stay in role.yaml until the
  language packs take them
  ([#1230](https://github.com/JN0V/workline/issues/1230)).
- **A path with a space** is read, Kotlin 1's and javac's plain paths
  too.

## Other languages

The same rule; their error shapes, from each compiler's documentation,
not tried yet.

| Language | Missing name | Shape |
|---|---|---|
| TypeScript (`tsc`) | `TS2304 Cannot find name`, `TS2339 Property … does not exist on type` | one line: `file(line,col): error TS2304: …` |
| C# | `CS0103` (a name), `CS1061` (a member) | one line: `file(line,col): error CS0103: …` |
| Rust | `E0425` (a value), `E0599` (a method), `E0433` (a path) | two lines: `error[E0425]: …`, then ` --> file:line:col` |
| Java by Maven | `cannot find symbol` | one line: `[ERROR] /…/X.java:[9,25] cannot find symbol` |

- **One-line shapes** (TypeScript, C#, Maven): set `build-errors` with
  their patterns. A message whose first quoted word is the missing name
  (`Cannot find name 'lostCount'`) needs no group `name`.
- **Something else quoted first** needs a group `name`. C#'s CS1061,
  `'Stats' does not contain a definition for 'lostCount'`, quotes the
  type first: match it with `definition for '(?P<name>\w+)'`.
- **Several lines** (Rust): one line does not hold both the file and
  the message, so these are not read yet. `cargo
  --message-format=json` would give both in one record.

## The lint

The tests pass the project's lint before the Draft opens
([hand-over](hand-over.md)). A lint that type-checks, as
golangci-lint does, reports the same missing name
([#988](https://github.com/JN0V/workline/issues/988)):

- **Not counted**: a lint finding read by `build-errors` as a missing
  name, or as one of the build's own errors in a test of the change,
  which the proof reads or the judge does. golangci-lint puts the
  compiler's lines in its message (`calc/mul_test.go:8:5: undefined:
  Mul`), at the package's first file: each line must be one.
- **Still counted**: any other finding, a format check's too.
- **Seen later**: golangci-lint runs no other linter on a package that
  does not build. An unchecked error in the test shows only once the
  code exists, on the builder's merge request.
