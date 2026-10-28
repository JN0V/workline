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
[ADR-0051]: 0051-a-check-sits-where-its-catch-is-worth-its-cost.md
