# Tester — where it stands (2026-10-28)

**In one line:** `workline test write --issue <n>` has the agent write
a test for each named scenario of a ready issue, proves them red on the
base with the red-first check (`workline test check`, no agent), and
opens them as a Draft; it keeps them from the builder, and answers an
objection to one (`workline test answer`); a scheduled run takes one
ready issue a night, an objection wakes its answer; on the closing pull
request, each new test passes three runs out of three. Steps 1 to 5 of
[ADR-0055's plan](../../../docs/adr/0055-the-tester-writes-the-acceptance-tests-before-the-code.md#build-plan);
beta, not in the line yet.

A try is told in its pull request's `## Tried`
([ADR-0043](../../../docs/adr/0043-a-try-is-told-in-its-pull-request.md)):
the first, the check on a copy of a real Go module with gotestsum, on
[#649](https://github.com/JN0V/workline/pull/649); then a real agent's
tests for a real bug, on [#652](https://github.com/JN0V/workline/pull/652);
the scheduled pick and the objection's chain on GitHub, stub agents, on
[#656](https://github.com/JN0V/workline/pull/656); a real agent's Go
tests of a name the base lacks, on a copy of workline, on
[#659](https://github.com/JN0V/workline/pull/659).

## To leave beta

Beta until the three criteria of
[ADR-0036](../../../docs/adr/0036-a-roles-status-is-earned-by-written-criteria.md)
hold:

1. **Nothing buildable in Missing**: open; steps 6 to 8 below.
2. **The main issue validated on real use**: open.
   [#264](https://github.com/JN0V/workline/issues/264), on workline and
   on a copy of a real repository: a bug shown by a red test first, then
   green.
3. **AI verdicts measured**: open. The tests `test write` writes,
   measured against the scenarios they cover: one try so far.

## Built

Cases in `tests/conformance/cases/tester/`:

- **The red-first check**: a new test red on the base passes; one green
  there is refused (`test-green-on-base`); one red on the head beside
  code is refused (`test-red-on-head`); the change's tests alone are
  asked red only.
- **Fail loud**: no test command blocks (`setting-missing`), no report
  blocks (`report-missing`); a change with no new test says so.
- **A build record told from a test**: with the tests alone, a red id
  in a suite that lost its base tests is the runner's record only when
  it is the suite's one case, with no classname or named `TestMain`.
- **A missing name told from a broken build**: a build that broke on
  the base is red only when each compiler error lies in the change's
  tests and names what the base lacks; any other blocks
  (`build-broken`). Go's errors by default (`build-errors`,
  [the page](missing-name.md)).
- **Tests from the criteria** (`workline test write --issue <n>`): one
  test a scenario, named after it; a ready issue only, with named
  scenarios; new test files only; a scenario said untestable in a note;
  proven red before the Draft opens; no agent, nothing written.
- **Frozen to the builder**, no agent: the record of each test file's
  blob in workline's own comment on the Draft; a test changed or
  deleted by another blocks (`tester-test-changed`), a forged trailer
  too; a record a person wrote is never read.
- **Objections** (`workline test answer --merge-request <n>`): kept
  with the reason; rewritten on the branch, `Refs` the thread, still red
  on the base, else refused; the need itself back to the product owner;
  a form it cannot read said; the five whys due at the third answer;
  past it, a person decides, and the tester follows them.
- **Its moment** (step 4): on `schedule`, one ready issue a run, the
  first in the backlog's order with named scenarios and no open Draft;
  none waiting, said; `--open-merge-request` needed
  (`no-merge-request`). Judged with no write token, applied apart
  (`--no-apply`, then `workline apply`).
- **The objection's trigger** (step 4): GitHub's
  `workline-objection.yml` and `workline-answer.yml`, GitLab's
  `workline:objection:*` jobs on a pipeline a note webhook starts.
- **The green proof** (step 5), no agent: on the tester's pull request
  marked ready, the base once, then the head three times in one fresh
  checkout; a new test red in some runs is flaky and blocks
  (`test-flaky`), red in all is `test-red-on-head`; a flaky test the
  base had warns; a Draft waits ([the page](green-proof.md)).
- **The hand-over** to whoever builds:
  [the contract](hand-over.md).
- **Settings** `test-command`, `test-report`, `test-paths`,
  `code-lines-max`, `objection-exchanges`, `objection-person`
  ([README](../README.md#settings)).

## Missing

- **Asked again**: a test refused (green on the base) is not asked of
  the agent again with the reason; the run blocks.
- **Adding to a test file already there**: refused for now; new files
  only.
- **Not in workline's own line yet**: a Go issue that adds a name now
  gets its Draft. Routing it on workline's schedule still needs its
  `test-command` set, gotestsum in the gardening's judge job, and a
  choice for the conformance cases many issues ask for, which run only
  in `./tests/conformance`; the objection workflows wait for a Draft.
- **A Draft a person closed** is not remembered as refused: a later
  scheduled run may take its issue again.
- **Objections on a forge command** (`cmd:`): it lists no thread, and
  `no-threads` blocks.
- **A test's flakiness over time**: each push to the ready pull request
  runs the three again; no history across pushes (the curator's, step 6).
- **Steps 6 to 8**: the curator's measures, redundancy with its proof,
  the five whys skill.
- **A missing name in Rust or Java**: their compilers' errors span
  several lines, which `build-errors` does not read
  ([the page](missing-name.md#other-languages)).
- **Not in the line, nor in the CI templates**: a project runs the
  command in its own job.
