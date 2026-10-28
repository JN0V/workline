# Tester — where it stands (2026-10-28)

**In one line:** the red-first check, `workline test check --base
<ref>`, proves from the project's test runner's JUnit report that each
test a change adds fails on the base and, beside code, passes on the
head. No agent. Step 1 of
[ADR-0055's plan](../../../docs/adr/0055-the-tester-writes-the-acceptance-tests-before-the-code.md#build-plan);
beta, not in the line yet.

A try is told in its pull request's `## Tried`
([ADR-0043](../../../docs/adr/0043-a-try-is-told-in-its-pull-request.md)):
the first, the check on a copy of a real Go module with gotestsum, on
the pull request that built step 1
([#264](https://github.com/JN0V/workline/issues/264)).

## To leave beta

Beta until the three criteria of
[ADR-0036](../../../docs/adr/0036-a-roles-status-is-earned-by-written-criteria.md)
hold:

1. **Nothing buildable in Missing**: open; steps 2 to 8 below.
2. **The main issue validated on real use**: open.
   [#264](https://github.com/JN0V/workline/issues/264), on workline and
   on a copy of a real repository: a bug shown by a red test first, then
   green.
3. **AI verdicts measured**: open; no agent yet. Once step 2 is built,
   the tests it writes measured against the scenarios they cover.

## Built

Cases in `tests/conformance/cases/tester/`:

- **The red-first check**: a new test red on the base passes; one green
  there is refused (`test-green-on-base`); one red on the head beside
  code is refused (`test-red-on-head`); the change's tests alone are
  asked red only.
- **Fail loud**: no test command blocks (`setting-missing`), no report
  blocks (`report-missing`); a change with no new test says so.
- **Settings** `test-command`, `test-report`, `test-paths`
  ([README](../README.md#settings)).

## Missing

- **Step 2, tests from the criteria**: `pre` writes the task, a
  `test-file` intention, `apply` holds the test paths, runs this check,
  opens the Draft.
- **Step 3, frozen guard and objections**: `tester-test-changed`, the
  objection's thread.
- **Step 4, the moment**: the scheduled run, the developer's hand-over.
- **Step 5, green proof**: three runs out of three on the closing pull
  request; step 1 runs the head once.
- **Steps 6 to 8**: the curator's measures, redundancy with its proof,
  the five whys skill.
- **A missing name told from a broken build**: in Go, a missing name
  breaks the whole package's build; step 1 counts it red and warns
  (`tests-not-built`), never blocks on it.
- **Not in the line, nor in the CI templates**: a project runs the
  command in its own job.
