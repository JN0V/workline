# ADR-0055: The tester writes the acceptance tests before the code, and keeps the corpus small

- **Status:** proposed
- **Date:** 2026-10-28
- **Decided by:** the maintainer, on 2026-10-28: build the tester before
  the developer, which depends on it; the [five choices](#for-the-person)
  as recommended, choice 2 amended: an objection is talked over
- **Builds on:** [ADR-0018], [ADR-0027], [ADR-0039], [ADR-0042],
  [ADR-0043], [ADR-0050], [ADR-0051]; principles 1, 2, 4, 5, 7, 11, 12, 13
- **Touches:** ADR-0054 ([#647]): the developer starts from the
  tester's branch, and objects to a test rather than edit it
- **Research:** [tests-first-and-corpus.md](../research/tests-first-and-corpus.md),
  [acceptance-test-disputes.md](../research/acceptance-test-disputes.md)

## Context

- **Whoever builds a change writes the tests that prove it**, and can
  bend them to the code. With a developer role both are one agent, and
  agents delete or edit failing tests
  ([research](../research/tests-first-and-corpus.md#agents-and-the-tests-they-can-edit)).
- **The issue already holds the tests' source**: acceptance criteria as
  named Given / When / Then scenarios, a person's ([ADR-0039]); a test
  plan, the builder's.
- **workline's own practice is the model**: a conformance case first,
  in `pending.txt` until it passes ([ADR-0027]).
- **The maintainer asked, on [#264](https://github.com/JN0V/workline/issues/264)**:
  red then green, a bug shown by a red test first; a corpus that does
  not grow forever; subsets per module run locally, a wider set in CI.

## Decision

### Three jobs, built in this order

1. **Tests first**: a `ready` issue's acceptance tests, red on the base.
2. **Corpus curator**: time, redundancy, flaky tests, subsets; removals
   with their proof.
3. **Adapt after a change**, with the reason, once the developer exists.

### Job 1: what it takes, and when

- **Trigger**: an issue labelled `workline:ready`, a person's act or the
  product owner's under the rules a person wrote ([ADR-0018]).
- **The moment**: a scheduled run after the product owner's and the spec
  review's, before the developer's ([ADR-0050] §3, the night column);
  `workline test --issue <n>` on a machine. Nobody waits: a token and
  time cap, no wait budget.
- **Input**: the acceptance criteria, the test plan, the Scope's paths
  and the tests beside them; the project's test command and report.
- **One test per scenario**, in the project's own framework, named
  after it, its Given / When / Then as the test's comment (Gherkin only
  where the project has `.feature` files). A scenario no test can check
  (a person's look) is said in the pull request, not faked.
- **The agent is a choice**: none, `claude`, `cmd:<any command>`; with
  none, the issue waits for its tests (judgement, principle 5).

### Where the tests land

- **The first commit of the work branch** `workline/issue-<n>`, a Draft
  pull request `Refs #<n>`, `## Tests` listing each scenario and its
  test. No pending marker: `main` never runs them red, in any language.
- **Whoever builds next pushes on that branch**, the developer
  ([#647]) or a person; the same pull request then says `Closes #<n>`
  and merges by the project's policy ([ADR-0042]), no queue of its own.

### The proof, red first

- **The engine runs, no AI**: the project's test command on the base
  plus the new tests, then reads the JUnit XML report (pytest,
  jest-junit, gotestsum, cargo nextest, Surefire, behave write it).
- **Each new test must fail on the base**, by an assertion or a missing
  name, not a broken build of other tests. One that passes is refused:
  it proves nothing new (inverting TestGen-LLM's pass filter).
- **No command, no report**: `setting-missing` or `report-missing`, blocking.
- **Green, on the closing pull request**: each test passes three runs
  out of three (TestGen-LLM's flakiness filter, cheaper).

### Guards

- **Frozen to the builder**: the tester's state records each test
  file's hash; another's commit that changes or deletes one is a
  blocking finding, `tester-test-changed`. No label lifts it: only the
  tester's commit moves a hash ([below](#a-test-that-does-not-fit)).
  Tests stay readable: hiding them holds no better (EvilGenie).
- **Writes test paths only**: the project's `test-paths` setting, by
  default each ecosystem's names (`*_test.go`, `test_*.py`,
  `*.test.ts`, `tests/`). Any other file refused at `apply`.
- **Never weakens an existing test without proof**: removing a test or
  an assertion is refused unless the pull request shows the corpus
  keeps its coverage and its killed mutants, before and after.
- **Budget caps**: `ai-max-tokens`, 20 min a run, one test a scenario
  plus two, `max-added-seconds` (30 s) to the suite; the Draft says which.

### A test that does not fit

A team's way ([research](../research/acceptance-test-disputes.md)):
the one held to the test explains, the one who wrote it decides.

- **The objection**: the builder comments on the test's line, opened by
  `workline:objection`: the test, its scenario, the kind (wrong
  interface, over-specified, contradicts another scenario, untestable
  as written, the need itself) and why. The Draft stays red.
- **The answer**: a fresh tester run, woken by it ([ADR-0050] §2),
  reads the scenario, the objection, the code only for an interface
  claim; replies *kept* with the reason, or *rewritten*, its own commit
  `Refs` the thread. Either wakes a fresh developer run.
- **The need itself**: a wrong or ambiguous scenario goes to the
  product owner, which amends the issue ([ADR-0038]); the tester adapts.
- **Three exchanges a thread**: still open after the second, the five
  whys; past the third, a person decides (`human`), the tester writes it. The
  review's shape ([#647]: five rounds, the five whys at three), shorter:
  one test against one scenario, not a whole change.
- **Not a cheat path**: the tester judges the scenario's meaning, not
  the builder's convenience; a rewrite still checks the scenario's Then
  and fails on the base (red-first); the reviewer reads it.
- **Five whys, a shared skill** ([defects become guards](../spec/role-adapting.md#defects-become-guards)):
  `skills/five-whys/SKILL.md`, also for a bug's red test (why no catch?).

### Job 2: the corpus curator

- **Weekly, off every path**, measures without AI ([ADR-0051]).
- **Measures**: each test's time from the JUnit report against
  `suite-max-seconds`; tests whose covered lines are a subset of
  another's; mutation on the lines changed since the last run, with the
  project's tool (Stryker, PIT, mutmut, cargo-mutants, Gremlins).
- **The AI judges a candidate**: a real duplicate, or a behaviour the
  measure cannot see (a bug's regression with its scenario stays).
- **Output**: a pull request that removes or merges, measures before
  and after, by the merge policy; never a lost killed mutant. No
  coverage or mutation tool: an issue proposing one, pinned
  ([inspector-coverage.md](../research/inspector-coverage.md)).
- **Subsets**: the project's own selection (Go's cache, pytest-testmon,
  Jest `--findRelatedTests`, Nx affected), proposed once in an issue:
  a subset on a machine, wider on a pull request, all on a release.
- **Flaky**: the ecosystem's skip and an issue naming it, never deleted.

### What it never does

- Write or change product code, CI, a runner's config, a gate.
- Merge, approve, mark its Draft ready, lift its own finding.
- Weaken, skip or delete a test without the proof above.
- Take an issue that is not `ready`, or invent a scenario the issue
  does not hold: a missing one is asked of the product owner.

## For the person

Five choices, each with the recommendation (chosen 2026-10-28, choice 2
amended) and its alternative:

1. **Where red tests live**: the work branch, in the pull request that
   makes them green. *Alternative*: merged first behind strict pending
   markers ([per ecosystem](../research/tests-first-and-corpus.md#a-failing-test-that-keeps-main-green)),
   only a skip in Go and Rust.
2. **Kept from the builder**: visible, frozen, an objection talked
   over with the tester (amended). *Alternative*: hidden holdout tests
   — a private store, a second runner, no better against an agent.
3. **Format**: the project's own test framework. *Alternative*: Gherkin
   `.feature` files — a runner and step code in every project.
4. **Red proof**: the runner's JUnit XML report. *Alternative*: one run
   per test by a name filter (`-run`, `-k`).
5. **Curator's output**: removal pull requests with before / after
   proof, by the merge policy. *Alternative*: an issue a person acts on —
   safer, and the corpus grows meanwhile.

## Build plan

Each step: conformance cases first; a try told under `## Tried` ([ADR-0043]).

1. **Red-first check, no AI**: `roles/tester`; settings `test-command`,
   `test-report`, `test-paths`; `workline test check --base <ref>`.
   Cases: a new test red passes; one green refused; no report or no
   command blocks.
2. **Tests from the criteria**: `pre` writes `task.md`; a `test-file`
   intention; `apply` holds the paths, runs step 1, opens the Draft.
   Cases: one test a scenario; a product file refused; agent none.
3. **Frozen guard, objections**: a change blocks; an objection wakes
   the tester; kept, or rewritten red; a need to the product owner.
4. **Moment and hand-over**: the scheduled run; the developer starts
   from the branch. Cases: one issue a run, an open Draft skipped.
5. **Green proof**: three runs on the closing pull request.
6. **Curator measures, no AI**; 7. **redundancy** with its proof;
   8. **five whys skill**, then **job 3** once the developer is built.

## Amendment 2026-10-28: step 2 as built

- **No `test-file` intention**: the catalogue's `patch`, held to
  `test-paths` by `duties.writes`; `post` refuses a file the base holds.
- **The command**: `workline test write --issue <n>`; the branch
  `workline/tester/issue-<n>`, as every role's branch is named.
- **`post` proves, before `apply`**: the tests committed on the base in
  a checkout apart, the red-first check run there; not red, nothing is
  applied nor opened.

## Amendment 2026-10-28: step 3 as built

- **The record**: each test file's git blob, in workline's own comment
  on the Draft (ADR-0051's records), written when it opens and by the
  tester's runs only; never a commit's trailer, which anyone writes.
  The guard runs on `merge-request` where the project routes the
  tester; none to read blocks, `tester-record-missing`.
- **The form**: one field a line after `workline:objection`: `test:`,
  `scenario:`, `kind:`, `why:`. One it cannot read is told the form.
- **No trigger yet**: `workline test answer --merge-request <n>` takes
  the first objection waiting; what runs it on a comment comes with
  step 4's moment, in CI templates beside the scheduled run.
- **The rewrite**: the engine commits it on the role's own branch, on
  top, without force; another's branch is never written to (ADR-0051).
- **The need itself**: no agent; the issue back to `to-refine`, its
  `ready` taken off (docs/spec/routing.md), said on it and in the thread.
- **A person decides**: the run passes, having said so (a `human`
  verdict applies nothing); the merge request labelled
  `objection-person` (default `workline:human`), or an `@login`
  mentioned. The next comment of a person of the project is their
  decision; the tester follows it.
- **The five whys** are said due at the third answer; the skill is
  step 8. A fresh developer run is woken once the developer exists.

## Amendment 2026-10-28: step 4 as built

- **The moment**: the tester on `schedule`, opt-in in the project's
  routing, after the product owner and the reviewer. One issue a run:
  the first in the backlog's order offered to whoever builds (ready, no
  open blocker, no parts) whose Validation names scenarios, its Draft
  not open. None: said, a pass.
- **Needs `--open-merge-request`**: the open Drafts are the engine's
  list of the role's branches; without it, `no-merge-request` blocks.
- **Two jobs** (principle 6): `route schedule --no-apply` proves the
  tests red with no write token; `apply` opens the Draft and its record.
  `test write` and `test answer` take `--no-apply` too.
- **The objection's trigger**: on GitHub, `pull_request_review_comment`
  runs a workflow with no token, which only notices a
  `workline:objection` comment of a person of the project or an App on
  a `workline/tester/` branch; its end wakes a `workflow_run` workflow
  from the default branch, which answers, then applies apart. On
  GitLab, no pipeline starts on a comment: a tool given the note webhook
  starts the template's `objection` jobs.
- **The Draft's tip is checked out** in both jobs: the test is read,
  and a rewrite committed, there. The apply job runs only git and the
  forge's API; its engine is set up before the switch.
- **The hand-over**: the branch, the Draft (`Refs`, never `Closes`), its
  `## Tests` table and the record;
  [the contract](../../roles/tester/docs/hand-over.md). No label: the
  branch is the data. The builder claims the issue by assignment
  ([ADR-0054]); the tester assigns nobody.
- **Not in workline's own line yet**: in Go, a test of a name the code
  lacks got no Draft (step 2's limit), and most of its issues add one;
  lifted by the amendment below.

## Amendment 2026-10-28: a missing name, told from a broken build

- **The gap**: in a compiled language, a test calling a name the base
  lacks breaks its package's build. The report then holds only the
  runner's record of it (gotestsum's `TestMain`), no new test:
  `test write` opened no Draft, though step 1 meant such a test red.
- **Borrowed**: "it does not compile" is TDD's first red. The compiler
  says which name is missing, in a shape each ecosystem fixes: Go's
  `file:line:col: undefined: X` (as `go vet` prints it, and gotestsum
  under `=== Errors`), tsc's `TS2304`, C#'s `CS0103`, rustc's `E0425`,
  javac's `cannot find symbol`.
- **The rule**: a build that broke on the base with the change's tests
  is red for the right reason only when every error line of the test
  command's output lies in a test file the change adds or changes, and
  names what the base lacks. Any other error blocks (`build-broken`).
- **The names**: with the tests alone, read from the test files
  (`file:name`); beside code, from the head's report, which must build
  and run them green.
- **Per ecosystem, by setting**: `build-errors`, each an error-line
  pattern, the missing-name messages and the test-name pattern; Go's is
  the default. One-line shapes (tsc, C#) are a setting away; rustc's and
  javac's span several lines, not read yet
  ([the page](../../roles/tester/docs/missing-name.md)).

## Consequences

- **The builder gets its target first**; only the tester moves it, on
  a traced objection; a person only past three exchanges.
- **Not covered**: a human test campaign, UI tests beyond the project's
  runner, observability; `unsafe` and CVE audits go to the security role
  ([#204](https://github.com/JN0V/workline/issues/204)).

[#647]: https://github.com/JN0V/workline/pull/647
[ADR-0018]: 0018-the-product-owner.md
[ADR-0027]: 0027-derived-blocks-are-regenerated-on-the-default-branch.md
[ADR-0038]: 0038-the-product-owner-proposes-on-the-issue-a-person-answers-there.md
[ADR-0039]: 0039-issue-sections-are-read-by-meaning-the-kind-is-the-forges.md
[ADR-0042]: 0042-the-engine-merges-by-the-projects-policy.md
[ADR-0043]: 0043-a-try-is-told-in-its-pull-request.md
[ADR-0050]: 0050-each-act-has-its-moment-and-a-wait-budget.md
[ADR-0054]: 0054-the-developer-takes-a-ready-issue-to-a-pull-request.md
[ADR-0051]: 0051-a-check-sits-where-its-catch-is-worth-its-cost.md

## Amendment 2026-10-28: step 5 as built

- **Where**: the tester on `merge-request`, after the frozen guard, on
  its branch once the pull request is marked ready: the builder marks
  it so when it closes the issue. A Draft's tests are meant red; the
  proof waits.
- **What runs**: the base once, to tell the new tests (the head's
  reports hold them, the base's does not); then the head three times,
  one after the other. Never again until green.
- **One fresh checkout for the three**, not three: the runner's build
  cache. On a copy of workline, a Go package: the proof about 10 s,
  not 22 s. A test that a run before it breaks is flaky too.
- **Go's result cache off** (`-count=1` in `GOFLAGS`): it would replay
  the first pass. A random order is the project's `test-command`'s,
  `-shuffle=on` in Go
  ([per ecosystem](../../roles/tester/docs/green-proof.md)).
- **Findings**: one red of three is `test-flaky`, blocking, the runs
  named; red in all three, `test-red-on-head`; a flaky test the base
  had warns, not the change's (principle 8).

## Amendment 2026-10-28: step 6 as built

- **The command**: `workline test measure --history <file>`, no agent,
  on `HEAD` in a checkout apart; the suite `curator-runs` times (2), to
  tell a flaky test on one commit. Red or flaky warns, never blocks.
- **The history**: a JSON file the CI passes from one run to the next as
  an artifact (GitHub 90 days, GitLab a branch's latest): no store of
  its own. Missing, the first measure, said. A test gone longer than
  `curator-window` (12 measures) is dropped.
- **Where a person reads it**: the job's summary, not a sticky issue.
  Nothing in it asks a person yet; step 7's pull requests will.
- **Its moment**: a `curator` job beside the gardening's, opt-in
  (`WORKLINE_CURATOR=on`), not a step of the `schedule` line: that line
  holds the agent's token; the measure needs none, nor a write token.
- **Coverage**: the command's own report, Go's profile or LCOV
  (`coverage-report`); total and least covered packages.
- **Not built**: per-test coverage, a run per test: twelve times the
  package's time on `internal/tester` (3.6 s, not 0.3 s), 11 min more on
  the whole corpus at that rate; duplicates by name or
  shape (a name says little, a shape needs a parser per language): step
  7 measures redundancy by covered lines. Mutation: only whether a tool
  exists for the language and is set up.
- **On workline**: 1,877 tests, 250 s of suite time; the measure 7.5
  min for two runs with coverage; no test over 2.7 s; coverage 33.9 %
  (the engine the conformance cases run is not profiled); a test red
  on the environment (`WORKLINE_AI` set), found.

## Amendment 2026-10-28: step 7 as built

- **The command**: `workline test curate --history <file>`, the
  tester's `curate` event; one folder of tests a run, the largest that
  fits `code-lines-max`, not weighed within `curator-window` measures
  (kept in the history) nor held by an open curator pull request.
- **The judgement**: the agent reads the folder's test files and each
  test's measures; proposes at most `curator-removals-max` (5) tests
  removed or merged, a note each naming the test that keeps its check.
- **The proof, no agent**: one checkout of `HEAD`, the touched folders
  only (`{packages}`), before and after the patches: green both times;
  no covered line lost; no killed mutant lost, the language's tool on
  each folder (`curator-mutation-command`, Gremlins' JSON or the
  mutation testing report schema); within `curator-proof-seconds`.
  Not per-test coverage: the folder's covered lines before and after
  are the same proof, at two runs' cost.
- **Departs from choice 5's "a drop with a stated reason"**: none is
  accepted; a covered line or a killed mutant lost is no pull request.
- **Kept out**: a test file the tester's open Draft records (frozen),
  a name a `curator-keep` file lists (workline: the conformance
  `pending.txt`), files outside `curator-paths` (workline: the
  conformance cases, whose built engine the profile does not see).
  A tester's test merged with an issue still open is not told apart.
- **Mutation required** where the language has a tool: unset, the
  curator does not run; `none`, coverage alone, said in the pull
  request.
- **The pull request**: not a Draft, opened by the bot from the weekly
  curator job (the gardening's `apply` on GitHub, `workline:curator:apply` on
  GitLab), merged by the merge policy ([ADR-0042]).
- **On workline**: three folders of its own unit tests, nothing
  redundant found (14 to 25 thousand tokens, under 10 s each); a
  duplicate planted in `internal/scale`, found and proven in 32 s:
  20 tests to 19, coverage 94.7 % and 141 of 173 mutants killed, both
  unchanged.

## Amendment 2026-10-28: step 8 as built, the five whys

- **One shared skill**: `skills/five-whys/SKILL.md` at the repository's
  root, in the Agent Skills format, shipped in the binary; any role
  declares it in `context.skills`, its `pre` names it in the run's
  `in/skills` when due ([role contract](../spec/role-contract.md#shared-skills),
  [research](../research/root-cause.md)). The `skills/` folder of a
  role holds only its own copy of one.
- **Not the agent's choice**: the role's code says when it is due; the
  body is given whole after the task, not picked from a description.
- **Its form, checked**: one `note` opening with `Five whys`; the
  problem, the whys numbered (five at most, each quoting the material),
  the root cause, an optional second (`Also:`), one action of a kind
  (`issue`, `test`, `doc`, `code`, `decision`); 1,500 characters at
  most. Read by the engine's `whys` package.
- **The tester, its first user**: the third answer to a thread loads it;
  the analysis is posted under its answer, kept or rewritten, for the
  builder's next turn or the person. Missing (`five-whys-missing`) or
  malformed (`five-whys-unreadable`), the answer is refused and the
  agent asked again. It replaces the warning `five-whys-due`.
- **Drawn by the auditor**: its own kind, `whys` ("five whys given"),
  from the answer's hidden line (`"whys": true`): does each because
  follow from the thread, is the cause one the project can change?
- **The developer** loads it the same way at review round three
  ([ADR-0054]), when it is built; the defect ledger
  ([role-adapting.md](../spec/role-adapting.md#defects-become-guards))
  likewise, when a defect is recorded.

## Amendment 2026-10-29: the curator removes, never merges

- **Decided by** the maintainer, on 2026-10-29, after the evaluation
  ([#672](https://github.com/JN0V/workline/pull/672)): on two tests that
  each checked their own branch, the curator merged them into one, five
  runs out of five, and the proof passed it.
- **The curator removes only**: never a test rewritten, nor two merged
  into one; step 7's "removed or merged" no longer holds. A test stays
  tied to its issue's scenario, and the proof covers a removal only.
- **Held by the engine**: `removal-writes-code` blocks, nothing proposed,
  when a touched file writes anything. A Go file is read by its
  declarations: each one left is one of the base's, unchanged but for
  whitespace. Any other file is read by its lines: the base's, in order,
  some gone. That is weaker, since deleted lines can still join two tests.

## Amendment 2026-10-29: in workline's own line

- **Routed** ([#264](https://github.com/JN0V/workline/issues/264)):
  `schedule`, last, one ready issue a night, its tests a Draft, the
  role's own settings; `merge-request`, after the gate and before the
  reviewer, its frozen guard and green proof on its own branches.
  Weekly, the docs' run skips it.
- **The command**: every package but `tests/evaluation`, whose tests
  measure agents, not the code; the conformance suite in, though it is
  most of the run (about 230 s of 240 s on a laptop). One run of the
  whole suite, never a second command beside it.
- **The report**: gotestsum, pinned, run by `go run` as the curator's
  commands are; the gardening's judge job leaves Go set up, and
  workline.yml's judge on `workline/tester/` branches. A check holds it
  (`TestOwnTesterHasGoWhereItRuns`).
- **Conformance cases are acceptance tests**: a new case under
  `tests/conformance/cases/` is a subtest of the suite's report, red on
  the base while the engine lacks what it says; its scenario named in a
  `# scenario:` comment. Never listed in `pending.txt`: a pending case is
  skipped, which the check refuses as green.
- **A case the suite cannot read** fails too, red for no reason:
  workline's `build-errors` adds a line for it, never a missing name,
  so the check says `build-broken`.
