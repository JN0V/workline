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
# Check a change

Part of [the tester](../README.md).

```sh
workline test check --base origin/main
```

- **Exit codes**: 0 pass, 1 block, 64 misuse; `--json`, `--repo <dir>`.
- **The head is `HEAD` as committed**: changes not committed are not read.
- **On a pull request's CI job**: the same line, the base its target
  branch; it needs the history down to the merge base (`fetch-depth: 0`
  on GitHub).

## What the check says

### What blocks

| Finding | Blocks | When |
|---|---|---|
| `test-green-on-base` | yes | a new test passes, or is skipped, on the base; `workline test check` says it of a Draft's guard too, which writing an issue's tests lets pass beside a red test ([write](write.md)) |
| `test-red-on-head` | yes | the change holds code, and a new test fails, is skipped, or did not run on the head |
| `setting-missing` | yes | `test-command` or `test-report` is not set |
| `test-command-fails` | yes | writing an issue's tests: the test command, run on the base before the agent, failed with no report |
| `setting-invalid` | yes | `test-report` leaves the repository's root, a `module-files` name holds a `/`, or `build-errors` does not compile |
| `report-missing` | yes | a run wrote no report where `test-report` says, or not JUnit XML; its error lines quoted. With the change's tests and no error a row reads, the agent is asked once more ([missing name](missing-name.md)) |
| `build-broken` | yes | the build broke on the base with the change's tests, and the judge says it is not by what the issue adds, its line quoted ([broken build](missing-name.md)); writing an issue's tests, after the agent is asked once more with the error, then a machine error the line runs again ([the record](../../../docs/spec/routing.md#once-a-state-the-record)) |


### What warns or informs

| Finding | Blocks | When |
|---|---|---|
| `tests-not-built` | no, warns | tests the base ran are not in the report of the base with the change's tests: their build broke there |
| `suite-record` | no, informs | the change's tests alone: a new case is the runner's record of its suite, not a test, not frozen |
| `no-new-test` | no, warns | the change touches test files, but no new test is in the report |
| `red-unproven` | no, warns; `human` with no agent | the build broke on the base past the fast path, and the judge did not settle it: the Draft opens marked, the developer starts ([broken build](build-outcomes.md#red-unproven)) |

### Cases

- **A change with no new test** passes, and says so.
- **The change's tests alone** (no code yet, the tester's own change):
  only red on the base is asked; green is the code's job. When they
  break the build, the runner's record of it (gotestsum's `TestMain`)
  is not a new test: `tests-not-built` says so.
- **A runner's record of a suite** (PlatformIO's `native:<suite>`): a
  case red by an error, a name none of the change's test files holds,
  beside the suite's tests. Not a test, never frozen: `suite-record`
  says so
  ([#1325](https://github.com/JN0V/workline/issues/1325)).
- **A broken build is read, then judged**: each compiler error a name
  the base lacks and the issue names is red at once; any other goes to
  the judge, whose yes is red and no `build-broken`
  ([how](missing-name.md)).
