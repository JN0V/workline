---
sources:
  - roles/tester/role.yaml
  - roles/tester/code
  - internal/kit/testmark
  - defaults/ci/github/workline-objection.yml
  - defaults/ci/github/workline-answer.yml
  - defaults/ci/github/workline-gardening.yml
checked: e81df7b34
verified: agent:claude-code
---
# Settings

Part of [the tester](../README.md). Under `roles: {tester: {settings: …}}`
in `.workline/config.yaml` ([config reference](../../../docs/config.md));
defaults from [role.yaml](../role.yaml).

## The test command and its report

| Key | Default | |
|---|---|---|
| `test-command` | none | runs the project's tests and writes a JUnit XML report; `{report}` is its path. The developer proves its code with it too |
| `draft-command` | none: `test-command` | proves a Draft red, and its pull request green, on the folders holding its tests alone, `{packages}`, not the whole suite ([#979](https://github.com/JN0V/workline/issues/979), [the green proof](green-proof-runs.md#the-head-and-again-when-red)) |
| `module-files` | none | the file marking a build module, by name alone: `build.gradle.kts` or `pom.xml`. `{packages}` is then the modules a change touches; with a build file at the root, a file outside any module runs the whole suite. The command turns a folder into its task ([#1176](https://github.com/JN0V/workline/issues/1176)) |
| `test-report` | `junit.xml` | where the report is written, from the root: a path, or a pattern, `**` spanning folders, each file it names read. Removed before each run, but a file git tracks, in a nested repository, or through a folder linked outside the root, as Bazel's `bazel-testlogs/*/*/test.xml`: kept, read only if the run rewrites it into a JUnit report ([#1175](https://github.com/JN0V/workline/issues/1175), [#1185](https://github.com/JN0V/workline/issues/1185), [#1189](https://github.com/JN0V/workline/issues/1189)) |

## Where the tests live

| Key | Default | |
|---|---|---|
| `test-paths` | none: the packs' | the test files, as patterns; every role reads them, the one definition of a test file ([ADR-0065](../../../docs/adr/0065-code-only-what-converges-language-knowledge-in-packs.md#3-one-definition-of-a-test-file)). Unset: the listed [packs'](../../../docs/packs.md) with the generic pack's, else the generic pack's alone: `test/`, `tests/`, `Tests/`, `__tests__/` folders, `test_*`, `*_test.*`, `*_spec.*`, `*.test.*`, `*.spec.*`, said by `workline doctor`. The only files `test write` may add; a new one beside the Scope is not listed beyond it by the developer ([its bounds](../../developer/docs/bounds.md#what-it-checks)) |
| `test-homes` | none: the packs' | where a test's code lives, for the developer's Scope: `[{tests: "**/src/test/kotlin/**", code: "**/src/main/kotlin/**"}]`, each `**` filling the next; unset, the listed [packs'](../../../docs/packs.md), then the generic ones: the folder holding a `test/` or `tests/` folder; in none, the test's own folder |

## A broken build

| Key | Default | |
|---|---|---|
| `build-errors` | Go, Kotlin, Java, Gradle, pytest | how a broken build's errors and missing names read, by ecosystem ([page](missing-name.md)) |
| `judge-read-rounds`, `judge-read-lines-max` | 2, 400 | what the judge of a broken build may read of the base before it answers ([page](build-judge.md#the-judge)) |

## What the agent gets and may spend

| Key | Default | |
|---|---|---|
| `code-lines-max` | 1500 | lines of the code the issue names and of the tests given to the agent, all files together, fewer when long lines would pass the context budget; a longer file gives the pieces the issue names in it, inside its declarations; the files the test and the Scope include or import, two steps deep, as declarations ([the code picker](../../../docs/spec/code-picker.md)); a fifth kept for a test beside when the code would fill it; files past it are named with their size |
| `ai-max-tokens`, `ai-max-seconds` | 300000, 1200 | what one `test write` or `test answer` run may spend, and wait on its agent |
| `ai-max-turns` | 40 | the turns of an agent that reads the project itself (`claude`), told in its task and enforced by the engine (`claude --max-turns`); reached before it answered: asked again once, then a machine error the line retries, never "no test possible" ([#1194](https://github.com/JN0V/workline/issues/1194)) |
| `objection-exchanges` | 3 | the tester's answers in an objection's thread before the product owner settles it |

## The curator and the time limit

| Key | Default | |
|---|---|---|
| `curator-*`, `suite-max-seconds`, `coverage-report` | 2 runs, 600 s | the curator's measures ([page](curator.md#settings)) and removals ([page](removals.md#settings)); each run of the tests (red-first, green, the developer's proof; not the curator's) stops at `suite-max-seconds`, the whole Draft's proof at three times it, the retry told the change's test files with no report by then and the last lines ([#1425](https://github.com/JN0V/workline/issues/1425)): its test command and the processes it started are killed on Linux and macOS, the command alone on Windows; a build daemon that detaches, as Gradle's, lives on ([#1197](https://github.com/JN0V/workline/issues/1197)) |

## The command, by ecosystem

| Runner | `test-command` |
|---|---|
| Go | `gotestsum --junitfile {report} ./...` |
| pytest | `python -m pytest --junitxml={report}`: the root on the import path, as a project not installed needs |
| Jest | `JEST_JUNIT_OUTPUT_FILE={report} npx jest --reporters=jest-junit` |
| cargo nextest | `cargo nextest run --profile ci`, `test-report` set to its JUnit path |
| Maven | `mvn test`, `test-report: "**/target/surefire-reports/*.xml"` |
| Gradle | `./gradlew test`, `test-report: "**/build/test-results/**/*.xml"` |
