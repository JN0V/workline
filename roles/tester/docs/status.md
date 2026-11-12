# Tester — where it stands (2026-11-12)

**In one line:** `workline test write --issue <n>` has the agent write
a test for each named scenario of a ready issue, proves them red on the
base with the red-first check (`workline test check`, no agent), and
opens them as a Draft; it keeps them from the builder, and answers an
objection to one (`workline test answer`); a scheduled run takes one
ready issue a night, an objection wakes its answer; on the closing pull
request, each new test passes three runs out of three; a weekly job
measures the corpus, no agent (`workline test measure`), and proposes
a few redundant tests removed, proven with no agent (`workline test
curate`). Steps 1 to 7 of
[ADR-0055's plan](../../../docs/adr/0055-the-tester-writes-the-acceptance-tests-before-the-code.md#build-plan);
beta; in workline's own line since
[#264](https://github.com/JN0V/workline/issues/264), woken by an issue's
state since [#764](https://github.com/JN0V/workline/pull/764); it wrote
the tests of the V1 acceptance round's 9 issues on three real
repository copies, right the first time on 8 of 9 in round 3
([round 3](https://github.com/JN0V/workline/issues/1266#issuecomment-6803875764)).

A try is told in its pull request's `## Tried`
([ADR-0043](../../../docs/adr/0043-a-try-is-told-in-its-pull-request.md)):
the first, the check on a copy of a real Go module with gotestsum, on
[#649](https://github.com/JN0V/workline/pull/649); then a real agent's
tests for a real bug, on [#652](https://github.com/JN0V/workline/pull/652);
the scheduled pick and the objection's chain on GitHub, stub agents, on
[#656](https://github.com/JN0V/workline/pull/656); a real agent's Go
tests of a name the base lacks, on a copy of workline, on
[#659](https://github.com/JN0V/workline/pull/659); the curator's
measures of workline's own corpus, on
[#665](https://github.com/JN0V/workline/pull/665); a real agent's
removal of a duplicate test, proven on a copy of workline, on
[#668](https://github.com/JN0V/workline/pull/668); the issue's
Verification written from a real agent's tests, a person's kept, on
[#1277](https://github.com/JN0V/workline/pull/1277#tried).

## To leave beta

Beta until the three criteria of
[ADR-0036](../../../docs/adr/0036-a-roles-status-is-earned-by-written-criteria.md)
hold:

1. **Nothing buildable in Missing**: open. The plan's eight steps are
   built; [Missing](#missing) keeps gaps the V1 rounds found, each an
   open issue.
2. **The main issue validated on real use**: awaits the maintainer.
   [#264](https://github.com/JN0V/workline/issues/264), on workline and
   on a copy of a real repository: a bug shown by a red test first, then
   green.
   - **The evidence**: V1 round 3, 6 bugs on Kotlin, C++ and Python
     copies, each red on its base in the tester's Draft, then green and
     merged by the policy
     ([round 3](https://github.com/JN0V/workline/issues/1266#issuecomment-6803875764));
     on workline, its Drafts opened by the line, such as
     [#1306](https://github.com/JN0V/workline/pull/1306) on 2026-11-10.
   - **What the maintainer would sign**: that these meet #264's
     Validation. #264 itself is blocked on a wider spec (test plans,
     corpus growth) its line found no test for; that spec is not this
     criterion's.
3. **AI verdicts measured**: awaits the maintainer: whether the
   measures below, with no second model, are enough. On real use: right
   the first time on 8 of 9 issues in each of rounds 1 and 3 and 9 of 9
   in round 2
   ([round 1](https://github.com/JN0V/workline/issues/1266#issuecomment-6779607472),
   [round 2](https://github.com/JN0V/workline/issues/1266#issuecomment-6796923928),
   [round 3](https://github.com/JN0V/workline/issues/1266#issuecomment-6803875764)),
   recorded in
   [ADR-0052's amendment](../../../docs/adr/0052-v1-0-when-the-core-roles-are-out-of-beta.md#amendment-2026-11-11-1266-v1-accepted).
   Seven evaluation cases with known
   answers (`tests/evaluation/cases/tester/`): `test write` (a bug, a
   name the base lacks, a scenario no test can check), `test answer` (a
   right objection, a wrong one), `test curate` (a true duplicate, a
   near-duplicate of another branch). On Sonnet, 30 of 35 runs passed
   every check; the miss, 5 of 5: the curator merged the near-duplicate
   rather than keeping it
   ([tried](https://github.com/JN0V/workline/pull/672)). The curator
   now removes only, told and held by the engine: 3 of 3 kept both
   ([tried](https://github.com/JN0V/workline/pull/674)). Left: another
   model, more runs over time. Its acts
   are observable: the auditor's weekly sample draws them
   ([ADR-0046](../../../docs/adr/0046-the-auditor-re-checks-every-roles-acts.md#amendment-2026-10-28-the-testers-row)).

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
- **A broken build, judged**: a build that broke on the base is red
  at once when each compiler error lies in the change's tests, names
  what the base lacks, and the issue names it; any other goes to the
  judge with the raw facts: yes red, no `build-broken`, anything else
  red unproven, the Draft opened and marked
  ([the page](missing-name.md),
  [#1227](https://github.com/JN0V/workline/issues/1227); tried on the
  Kotlin/Android copy and in Go, on
  [#1252](https://github.com/JN0V/workline/pull/1252#tried)). Go's,
  Kotlin's, javac's, Gradle's and pytest's collection errors read by
  default (`build-errors`; Kotlin's tried on an Android project,
  [#1181](https://github.com/JN0V/workline/pull/1181#tried); pytest's
  on the Python copy, [#1283](https://github.com/JN0V/workline/issues/1283)).
  A change's test file that does not build on the head, but for a
  missing name, is `build-broken`, never the code's red; on the base,
  a module pytest cannot import is the judge's, its no `build-broken`.
- **Tests from the criteria** (`workline test write --issue <n>`): one
  test a scenario; a ready issue only, with named
  scenarios; a new test file or one already there, lines added only;
  which test covers which scenario declared by the agent, each id found
  among the tests red on the base, or a reason for a scenario none can
  check ([#1229](https://github.com/JN0V/workline/issues/1229)); a
  broken build with no `test-name`: the test file the change adds,
  declared by its file, the file alone proven
  ([#1293](https://github.com/JN0V/workline/issues/1293));
  proven red before the Draft opens; no agent, nothing written.
- **An answer that does not read**, asked again once with the reader's
  error; each refusal written on the issue, at night too
  ([#761](https://github.com/JN0V/workline/issues/761)); a real agent's
  slip asked again, on [#776](https://github.com/JN0V/workline/pull/776).
- **A test green on the base**, asked again once, told which test
  passes there: the answer keeps its scenario as today, or the run
  blocks; a feature's every scenario kept so goes back to the product
  owner; trial #56's real answer replayed with its green test
  ([#1156](https://github.com/JN0V/workline/issues/1156)).
- **Real answers replayed** from the DomoticsCore trial's round 5, with
  the covers entries they predate: one test declared for two scenarios
  counts for both
  ([#1147](https://github.com/JN0V/workline/issues/1147)); prose before
  the list, or a `file:` beside a patch, no longer refuses the answer
  ([#1148](https://github.com/JN0V/workline/issues/1148)); a new test the
  project's runner skips is said, its filter named, to the next run too
  ([#1149](https://github.com/JN0V/workline/issues/1149)).
- **Tests that do not lint**, asked again once with each finding, its
  file and line from the repository's root; still found, blocked, the
  finding said on the issue
  ([#964](https://github.com/JN0V/workline/issues/964)); tried with the
  real golangci-lint on a copy of workline, on
  [#966](https://github.com/JN0V/workline/pull/966#tried). The lint's
  error naming the test's missing name not counted
  ([#988](https://github.com/JN0V/workline/issues/988),
  [the page](missing-name.md#the-lint)); tried with the real
  golangci-lint on a copy of workline, on
  [#996](https://github.com/JN0V/workline/pull/996#tried).
- **A Draft proven on its tests' folders**, `draft-command` naming
  `{packages}`, on the base and with the tests; a folder the base lacks
  runs nothing there
  ([#979](https://github.com/JN0V/workline/issues/979)); tried on a
  copy of workline, 697 s to 33 s, on
  [#991](https://github.com/JN0V/workline/pull/991#tried).
- **A test in a folder the base lacks**, written whole: its folder
  made, never through a link
  ([#989](https://github.com/JN0V/workline/issues/989)); tried on a
  copy of workline, on
  [#996](https://github.com/JN0V/workline/pull/996#tried).
- **Frozen to the builder**, no agent: the record of each test file's
  blob in workline's own comment on the Draft; a test changed or
  deleted by another blocks (`tester-test-changed`), a forged trailer
  too; a record a person wrote is never read.
- **Tests added to a file already there**, where the project keeps
  them: only lines added, `test-line-changed` else; the base's tests of
  that file never taken for new ones; the guard freezes the tester's
  lines, others added beside them pass
  ([#1259](https://github.com/JN0V/workline/issues/1259)); tried on a
  Python copy, on [#1290](https://github.com/JN0V/workline/pull/1290#tried).
- **Objections** (`workline test answer --merge-request <n>`): kept
  with the reason; rewritten on the branch, `Refs` the thread, still red
  on the base, else refused; the need itself back to the product owner;
  a form it cannot read said; the third answer gives the five whys,
  the shared skill, checked and posted with it (step 8);
  past it, the product owner settles it, then the issue is blocked
  ([ADR-0059](../../../docs/adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#6-disagreements-end-at-the-product-owner-then-blocked)).
- **A test missing** ([#1265](https://github.com/JN0V/workline/issues/1265)):
  a review's test finding on code, sent by the developer; the tester
  adds the test in any of its tests, whole or a diff, or says why none
  is needed, its answer said on the reviewed pull request too. Cases
  `tester/objection-missing-test-added`,
  `objection-missing-test-added-as-a-diff`,
  `objection-missing-none-needed`,
  `objection-missing-two-files-refused`. Tried: its pull request's
  `## Tried`.
- **Its moment** (step 4): on `schedule`, one ready issue a run, the
  first in the backlog's order with named scenarios and no open Draft;
  none waiting, said; `--open-merge-request` needed
  (`no-merge-request`). Judged with no write token, applied apart
  (`--no-apply`, then `workline apply`).
- **The objection's trigger** (step 4): GitHub's
  `workline-objection.yml` and `workline-answer.yml`, GitLab's
  `workline:objection:*` jobs on a pipeline a note webhook starts.
- **The green proof** (step 5), no agent: on the pull request closing
  the issue, marked ready, the developer's among them, the record read
  from the tester's Draft ([#1281](https://github.com/JN0V/workline/issues/1281)); the base once, then the head three times in one fresh
  checkout; a new test red in some runs is flaky and blocks
  (`test-flaky`), red in all is `test-red-on-head`; a flaky test the
  base had warns; a Draft waits ([the page](green-proof.md)). Each
  test id the record keeps must run: one missing blocks,
  `frozen-test-missing` ([#1255](https://github.com/JN0V/workline/issues/1255));
  one named from its file, or kept as its file alone, beside other new
  tests, the judge's.
- **The curator's measures** (step 6), no agent: `workline test
  measure` runs the suite twice on `HEAD`; each test's time, red and
  flaky ones, kept across measures in a history the CI passes as an
  artifact; slowest tests, never-red ones, coverage, whether a mutation
  tool exists; said in the job's summary. Opt-in in the gardening
  templates ([the page](curator.md)).
- **The curator's removals** (step 7): `workline test curate` gives
  the agent one folder of tests and their measures; it proposes a few
  removed, a reason each, never a test rewritten nor merged
  (`removal-writes-code`); the engine proves it with no agent,
  the folder's tests before and after: green, no covered line and no
  killed mutant lost (Gremlins pinned on Go); frozen and kept tests
  refused; a pull request, not a Draft, from the weekly curator job
  ([the page](removals.md)).
- **Observable** ([ADR-0046](../../../docs/adr/0046-the-auditor-re-checks-every-roles-acts.md#amendment-2026-10-28-the-testers-row)): its acts drawn in the auditor's
  weekly sample, one in ten a kind — its tests, its answers kept or
  rewritten, its curator's removals; the record keeps the scenarios
  read, each answer a hidden line (commit, model, scenario): tester's
  `write-one-test-a-scenario`, `objection-kept`, `objection-rewritten`;
  auditor's `sample-draws-the-testers-acts`,
  `verdict-reads-the-testers-acts`. Tried on
  [#669](https://github.com/JN0V/workline/pull/669).
- **The hand-over** to whoever builds:
  [the contract](hand-over.md).
- **A Draft closed with its fix**, no agent: `workline follow`, on each
  push to the base and each night, closes the Draft of a closed issue,
  naming the pull request that closed it, and takes the states' labels
  off closed issues; cases `tester/draft-closed-once-its-fix-merges`,
  `engine/closed-issue-loses-its-state`
  ([#999](https://github.com/JN0V/workline/issues/999)). GitLab runs the
  same command on its default branch's pipeline: built, not tried there.
  Tried on a copy of workline and read only on its live Drafts, on
  [#1001](https://github.com/JN0V/workline/pull/1001#tried).
- **A bug reproduced first**
  ([ADR-0060](../../../docs/adr/0060-the-person-is-on-the-loop.md#4-a-person-validates-by-the-issues-kind)):
  every test green on the base, no Draft, its reporter asked, then
  Blocked as not reproduced by the line
  ([how](hand-over.md#a-bug-not-reproduced)). Cases
  `tester/bug-green-on-base-asks-its-reporter`,
  `routing/bug-not-reproduced-blocked-after-the-delay`. Tried on
  [#876](https://github.com/JN0V/workline/pull/876#tried).
- **Its code by the shared picker**
  ([the code picker](../../../docs/spec/code-picker.md)): what the
  issue names, the files its agent asked for first, a test kept room
  for. Cases `tester/picker-*`. Tried on
  [#1106](https://github.com/JN0V/workline/pull/1106#tried).
- **What its test builds against, as declarations**: the includes and
  imports two steps deep, the test's first, bodies left out; any
  language, no parser. Cases `tester/picker-imports-as-declarations-*`,
  `tester/picker-includes-as-declarations-cpp`. Tried on
  [#1108](https://github.com/JN0V/workline/pull/1108#tried): the
  copy's #36 still writes no test.
- **Read tools under `claude`**
  ([ADR-0063, amended](../../../docs/adr/0063-every-open-issue-is-due-waiting-or-done.md#amendment-2026-11-05-the-testers-read-tools-first)):
  a read-only copy of the base, Read, Glob, Grep, 40 turns; what it
  read logged; an unconfirmed helper said as assumed on the Draft;
  `cmd:` and no agent keep `needs:`. Cases
  `tester/write-claude-reads-the-code-itself`,
  `tester/write-cmd-agent-reads-its-task-alone`. Tried on
  [#1110](https://github.com/JN0V/workline/pull/1110#tried): #36's
  tests written on the real harness; a second run, the copy's pattern
  reading `const struct`, reaches a Draft, red on a missing name, in
  one call; symlinks out of the copy denied; Go #1081 still passes.
- **In the line**: an issue labelled `workline:needs-tests` wakes the
  tester, no job of the project's own
  ([#764](https://github.com/JN0V/workline/pull/764)); on a repository
  with no forge, `workline route reconcile` runs it
  ([#1287](https://github.com/JN0V/workline/pull/1287)), as in the V1
  rounds
  ([round 3](https://github.com/JN0V/workline/issues/1266#issuecomment-6803875764)).
- **Objections on the local forge**: a Draft's line threads listed and
  answered at the next reconcile
  ([#1367](https://github.com/JN0V/workline/pull/1367)).
- **A runner's record of a suite never frozen**
  ([#1342](https://github.com/JN0V/workline/pull/1342)).
- **Settings** `test-command`, `test-report`, `test-paths`,
  `code-lines-max`, `objection-exchanges`
  ([README](../README.md#settings)).

## Missing

Gaps found on real use and in the V1 rounds
([round 3](https://github.com/JN0V/workline/issues/1266#issuecomment-6803875764)),
kept for 1.0:

- **A person's comment on a Draft** is read by no role
  (kept for 1.0, [#1358](https://github.com/JN0V/workline/issues/1358)).
- **"The existing suite guards it"** names no test, and was false 3 of 3
  times (kept for 1.0, [#1359](https://github.com/JN0V/workline/issues/1359)).
- **Frozen tests under a runner's own names** judged again on every head
  (kept for 1.0, [#1373](https://github.com/JN0V/workline/issues/1373)).
- **A test under a display name** may escape being frozen
  (kept for 1.0, [#1348](https://github.com/JN0V/workline/issues/1348)).
- **A new test hidden** by a same-named test in another file
  (kept for 1.0, [#1291](https://github.com/JN0V/workline/issues/1291)).
- **A missing name in Rust**: its errors span several lines, which
  `build-errors` does not read
  ([the page](missing-name.md#other-languages)); build errors move to
  language packs
  (kept for 1.0, [#1319](https://github.com/JN0V/workline/issues/1319)).

Parked, none seen in the V1 rounds:

- **The developer beside a frozen test**: its bounds still refuse any
  edit to a file the tester froze, a shared one whole, though the guard
  passes lines added beside the frozen ones
  (parked, [#1388](https://github.com/JN0V/workline/issues/1388)).
- **A case the conformance suite cannot read** is told by workline's
  own `build-errors` line, not by the role: another project's suite
  that fails a malformed test file alike reads as red
  (parked, [#1388](https://github.com/JN0V/workline/issues/1388)).
- **A Draft a person closed** is not remembered as refused: a later
  scheduled run may take its issue again
  (parked, [#1388](https://github.com/JN0V/workline/issues/1388)).
- **Objections on a forge command** (`cmd:`): it lists no thread, and
  `no-threads` blocks; the local forge lists them since
  [#1367](https://github.com/JN0V/workline/pull/1367)
  (parked, [#1388](https://github.com/JN0V/workline/issues/1388)).
- **Job 3**, a test adapted after a change, with the reason
  (parked, [#1388](https://github.com/JN0V/workline/issues/1388)).
- **A test's flakiness over time on pull requests**: each push to the
  ready pull request runs the three again; their runs are not fed to the
  curator's history, which reads the weekly measure only
  (parked, [#91](https://github.com/JN0V/workline/issues/91)).
- **The curator's removals**: a tester's test merged while its issue is
  still open is not told apart (only open Drafts' records are read); a
  mutation report other than Gremlins' or the schema's is not read
  (parked, [#91](https://github.com/JN0V/workline/issues/91)).
