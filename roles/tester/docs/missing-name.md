---
sources: [internal/tester/build.go, internal/tester/redfirst.go, internal/tester/tester.go, internal/tester/write.go, internal/tester/objection.go, internal/tester/junit.go, roles/tester/role.yaml]
checked: 90ca43c5
verified: agent:claude-code
---
# A missing name, told from a broken build

Part of [the tester](../README.md). A test written before its code
calls names the code lacks. In a compiled language the test's package
then does not build: the report holds no new test, only the runner's
record of the failed build.

```mermaid
flowchart LR
  run["<b>Tests on the base</b><br/>build fails"]
  errs["<b>Each error line</b><br/>of the output"]
  red["<b>Red</b><br/>what the change adds"]
  block["<b>Back to the tester</b><br/>build-broken"]
  run --> errs
  errs -- "each file of the change's tests:<br/>one error on what the issue adds,<br/>the others its knock-ons" --> red
  errs -- "any other" --> block
```

## The rule

- **No report**: a build that stops at compiling the tests, as Gradle
  does, writes no report. The compiler's lines are read first
  ([#1192](https://github.com/JN0V/workline/issues/1192)).
- **Red, for the right reason**, when each file of the change's tests
  holding an error holds one on what the change adds:
  - **a missing name** (`undefined: Mul`, `Unresolved reference 'mul'`)
    the issue's text names as a whole word, or its last part
    (`lostCount` for `MQTTStatistics::lostCount`)
    ([#1111](https://github.com/JN0V/workline/issues/1111));
  - **any other error** quoting a name the issue writes as a call
    (`restoreFromExisting(existingCount, nextIndex)` makes `Too many
    arguments for 'fun restoreFromExisting(…)'` red), or a name the
    base lacks that the issue names (`'nextIndex' overrides nothing`).
    With no issue known, a name the base lacks counts unless a test of
    the change declares it (`class FakeCalc`). Quoting none, the words
    the test calls count (`call to Add`, `Add(` in the test).
- **Knock-ons**, in such a file, are not counted: a type the compiler
  could not infer (`Cannot infer type`), a missing name the test reaches
  as a member (`grid.vertical`), or a type it does not call
  (`GridType.THIRDS`). Any other error quoting nothing (a parse error,
  K1's `Type mismatch: inferred type is String`) or quoting a value
  (Go's `cannot use "Kitchen" … as int value`) is the test's own.
- **Anything else goes back to the tester** (`build-broken`): a syntax
  error, a wrong import, a wrong call on what the base has (`call to
  Add` where the issue only names `Add`), an error in a code file, a
  helper the test calls and the issue never names
  (`stubSetBrokerLink`). The check quotes the compiler's lines. Writing
  an issue's tests, the agent is asked once more with them.
- **No report and no error a row reads**: `report-missing`, its error
  lines quoted, not the build tool's last ones (`BUILD FAILED`). On the
  tests' run, the agent is asked once more; on the base alone, final.
- **The tests' names**, when nothing built them: read from the test
  files, as `file:name` (`calc/mul_test.go:TestMulTwoByThree`).
- **Beside code**, the head builds and runs those tests: they must pass
  there, as any new test. A name the change's code touches counts as
  one the issue writes.
- **No error line read**, in a language with no patterns set: as before,
  `tests-not-built` warns, and `test write` opens no Draft.

## The setting

`build-errors`, a list, one item an ecosystem:

| Key | |
|---|---|
| `error` | a regular expression matching one error line of the test command's output, with groups `file` and `message` |
| `missing-name` | the messages naming what the code lacks; the name in a group `name`, else the message's first quoted word (`‘lostCount’`) |
| `name-line` | optional: a later line giving the name a missing-name message lacks, in its group `name`; read up to the next error line |
| `test-name` | finds the tests a test file declares; its first group that matched is the name; empty, the row names no tests |

- **The file, from the repository's root**: an absolute path or a
  `file://` URI under the folder the tests ran in is read from there,
  as Gradle prints Kotlin's and javac's errors
  ([#1177](https://github.com/JN0V/workline/issues/1177)). On Windows
  too: `file:///C:/…` and `C:\…` alike, the case not counted
  ([#1183](https://github.com/JN0V/workline/issues/1183)).
- **A wrong pattern** blocks before any run (`setting-invalid`).
- **Set in a project**, the list replaces the defaults whole.

## The defaults

In [role.yaml](../role.yaml), each tried on a real build:

| Ecosystem | Error line | Missing name |
|---|---|---|
| Go | `calc/mul_test.go:6:5: undefined: Mul` | `undefined`, `has no field or method`, `unknown field` |
| Kotlin, by Gradle | `e: file:///…/MulTest.kt:10:25 Unresolved reference 'mul'.`; Kotlin 1's `e: /…/MulTest.kt: (10, 25): …` | `Unresolved reference`, `No parameter with name` |
| Java (`javac`) | `/…/MulTest.java:9: error: cannot find symbol` | `cannot find symbol` |
| Gradle build files | `Build file '/…/build.gradle' line: 3` | none: it blocks |

- **Kotlin's tests**: `@Test fun `two by three`()` or `@Test fun
  twoByThree()`, named without the backquotes.
- **javac's name** is on a later line (`symbol: method mul(int,int)`),
  read by its `name-line`, then checked against the issue's text.
- **A path with a space** is read, Kotlin 1's and javac's plain paths
  too.
- **JUnit 5 on Gradle** reports `two by three()`: the `()` is dropped,
  the name as its file writes it
  ([#1178](https://github.com/JN0V/workline/issues/1178)).

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
  name in a test of the change. golangci-lint puts the compiler's lines
  in its message (`calc/mul_test.go:8:5: undefined: Mul`), at the
  package's first file: each line must be a missing name.
- **Still counted**: any other finding, a format check's too.
- **Seen later**: golangci-lint runs no other linter on a package that
  does not build. An unchecked error in the test shows only once the
  code exists, on the builder's merge request.

## Limits

- **A name not read**: a message with no group `name` and no quoted
  word counts as red, as before: the issue's text is not checked.
- **Case counts**: `lostcount` in the issue does not name `lostCount`.
- **A common word**: a made-up name that is also a word of the issue's
  prose (`get`, `init`, `lost`) counts as red, since a whole word
  matches.
- **`workline test check`** on a pull request knows no issue: any
  missing name counts; another error counts by the base or the
  change's code alone.
- **The base's words**: "a name the base lacks" is a whole-word search
  of the base's files; a word in a comment there counts as held.
- **Go stops at ten errors** a package (`too many errors`): the errors
  past the tenth are not read. `-gcflags=-e` in the test command shows
  them all.
- **A new package's import** (`package … is not in std`) is not counted
  as a missing name: it blocks.
