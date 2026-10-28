# ADR-0055: The tester writes the acceptance tests before the code, and keeps the corpus small

- **Status:** proposed
- **Date:** 2026-10-28
- **Decided by:** the maintainer, on 2026-10-28: build the tester before
  the developer, which depends on it; the five choices
  [below](#for-the-person) wait for the maintainer
- **Builds on:** [ADR-0018], [ADR-0027], [ADR-0039],
  [ADR-0042], [ADR-0043], [ADR-0050], [ADR-0051]; principles 1, 2, 4, 5,
  7, 11, 12, 13
- **Touches:** ADR-0054, the developer, proposed in
  [#647](https://github.com/JN0V/workline/pull/647): it starts from the
  tester's branch ([below](#where-the-tests-land))
- **Research:** [tests-first-and-corpus.md](../research/tests-first-and-corpus.md)

## Context

- **Whoever builds a change writes the tests that prove it**, and can
  bend them to the code. With a developer role both are one agent, and
  agents delete or edit failing tests
  ([research](../research/tests-first-and-corpus.md#agents-and-the-tests-they-can-edit)).
- **The issue already holds the tests' source**: acceptance criteria as
  named Given / When / Then scenarios, a person's ([ADR-0039]); a test
  plan, the builder's.
- **workline's own practice is the model**: a behaviour starts as a
  conformance case, listed in `pending.txt` until it passes; the suite
  fails if a listed case passes ([ADR-0027]).
- **The maintainer asked, on [#264](https://github.com/JN0V/workline/issues/264)**:
  red then green, a bug shown by a red test first; a corpus that does
  not grow forever; subsets per module run locally, a wider set in CI.
- **No tool writes tests from a need, red first**: AI generators keep
  only tests that pass today
  ([research](../research/tests-first-and-corpus.md#ai-test-generation)).

## Decision

### Three jobs, built in this order

1. **Tests first**: from a `ready` issue's acceptance criteria, the
   acceptance tests, before any code; red on the base.
2. **Corpus curator**: run time, redundancy, flaky tests, the subset a
   change needs; proposals to remove or merge, each with its proof.
3. **Adapt after a change**: a test a change made wrong, rewritten with
   the reason; after the developer exists.

### Job 1: what it takes, and when

- **Trigger**: an issue labelled `workline:ready`, a person's act or the
  product owner's under the rules a person wrote ([ADR-0018]).
- **The moment**: a scheduled run after the product owner's and the spec
  review's, before the developer's ([ADR-0050] §3, the night column);
  `workline test --issue <n>` on a machine. Nobody waits: no wait
  budget, a token and time cap instead.
- **Input**: the acceptance criteria, the test plan, the Scope's paths
  and the tests beside them; the project's test command and report.
- **One test per scenario**, named after it; the scenario's Given /
  When / Then kept as the test's comment. A scenario no test can check
  (a person's look) is said in the pull request, not faked.
- **The project's own framework**, in its language. Gherkin only where
  the project already has `.feature` files and a runner.
- **The agent is a choice**: none, `claude`, `cmd:<any command>`. With
  none, the role says the issue waits for its tests, writes nothing:
  writing a test from a need is judgement (principle 5).

### Where the tests land

- **The first commit of the work branch** `workline/issue-<n>`, a Draft
  pull request `Refs #<n>`, `## Tests` listing each scenario and its
  test. No pending marker: `main` never runs them red, in any language.
- **Whoever builds next pushes on that branch**: the developer
  (ADR-0054 starts from it when it exists) or a person; the same pull
  request then says `Closes #<n>` and merges by the project's policy
  ([ADR-0042]). The tests are not reviewed in a queue of their own.
- **Pending markers, only when tests must merge first** (choice 1's
  alternative): pytest `xfail(strict=True)`, Jest `test.failing`,
  Cucumber / behave `@wip` with `--wip`; where the ecosystem has only a
  skip (Go, Rust, JUnit), a list the engine checks, as `pending.txt`.

### The proof, red first

- **The engine runs, no AI**: the project's test command on the base
  plus the new tests, then reads the JUnit XML report (pytest,
  jest-junit, gotestsum, cargo nextest, Surefire, behave write it).
- **Each new test must fail on the base**, by an assertion or a missing
  name, not a broken build of other tests. One that passes is refused:
  it proves nothing new (inverting TestGen-LLM's pass filter).
- **No command, no report**: `setting-missing` or `report-missing`, said
  and blocking (principle 12).
- **Green, on the closing pull request**: each test passes three runs
  out of three (TestGen-LLM's flakiness filter, cheaper).

### Guards

- **Frozen**: the tester's state on the pull request records each test
  file and its hash. A later commit that changes or deletes one is a
  blocking finding, `tester-test-changed`; a person lifts it with a
  label, `workline:tests-changed` (one click, as every person's yes,
  [ADR-0038]). Tests stay readable by the builder: hiding them costs a
  second store and holds no better (EvilGenie).
- **Writes test paths only**: the project's `test-paths` setting, by
  default each ecosystem's names (`*_test.go`, `test_*.py`,
  `*.test.ts`, `tests/`). Any other file refused at `apply`.
- **Never weakens an existing test without proof**: removing a test or
  an assertion is refused unless the pull request shows the corpus
  keeps its coverage and its killed mutants, before and after.
- **Budget caps**: `ai-max-tokens` and 20 min a run; at most one test a
  scenario plus two; new tests add at most `max-added-seconds` (30 s by
  default) to the suite. Over a cap: the Draft says which.

### Job 2: the corpus curator

- **The moment**: weekly, off every path; the measures without AI
  ([ADR-0051]: a check sits where its catch is worth its cost).
- **Measures, no AI**: each test's time from the JUnit report, the
  suite's total against `suite-max-seconds`; tests whose covered lines
  are a subset of another's; mutation on the lines changed since the
  last run, with the project's tool (Stryker, PIT, mutmut,
  cargo-mutants, Gremlins), Google's way, not the whole tree.
- **The AI judges a candidate**: a real duplicate, or a distinct
  behaviour the measure cannot see (a bug's regression with its own
  scenario stays).
- **Output**: a pull request that removes or merges, with the measures
  before and after, by the project's merge policy; adequate only, never
  a lost killed mutant. A project with no coverage or mutation tool
  gets an issue proposing one, pinned, as the inspector does
  ([inspector-coverage.md](../research/inspector-coverage.md)).
- **Subsets**: the project's own selection (Go's cache, pytest-testmon,
  Jest `--findRelatedTests`, Nx affected, bazel-diff), proposed once in
  an issue: the subset on a machine, a wider set on a pull request, all
  on a release. workline builds no selector of its own.
- **Flaky**: a test that fails then passes on the same tree gets the
  ecosystem's skip and an issue naming it, counted in each run's
  report; never deleted for it.

### Five whys, a shared skill

- **One `skills/five-whys/SKILL.md`** (Agent Skills format), any role
  may load ([role-adapting.md](../spec/role-adapting.md#defects-become-guards)).
- **The tester loads it** when a person lifts `tester-test-changed`
  (why was the test wrong?) and when a bug's red test is written (why
  did no test catch it?). The answer goes to the defect ledger, not to
  a test.

### What it never does

- Write or change product code, CI, a runner's config, a gate.
- Merge, approve, mark its Draft ready, lift its own finding.
- Weaken, skip or delete a test without the proof above.
- Take an issue that is not `ready`, or invent a scenario the issue
  does not hold: a missing one is asked of the product owner.

## For the person

Five choices, each with the recommendation and its alternative:

1. **Where red tests live**: the work branch, in the pull request that
   makes them green. *Alternative*: merged first behind strict pending
   markers — reviewed alone, but a marker per ecosystem and a list
   where only a skip exists.
2. **Kept from the builder**: visible and frozen, a change a finding.
   *Alternative*: hidden holdout tests, run only by a gate — a private
   store, a second runner, no better against an agent.
3. **Format**: the project's own test framework. *Alternative*: Gherkin
   `.feature` files — the scenarios word for word, but a runner and step
   code in every project.
4. **Red proof**: the runner's JUnit XML report. *Alternative*: one run
   per test by a name filter (`-run`, `-k`) — no report needed, a run
   per test.
5. **Curator's output**: removal pull requests with before / after
   proof, by the merge policy. *Alternative*: an issue a person acts on —
   safer, and the corpus grows meanwhile.

## Build plan

Each step: its conformance cases first, `pending.txt` listing what does
not pass; a try on JN0V/workline-sandbox told under `## Tried`
([ADR-0043]).

1. **Red-first check, no AI**: `roles/tester`, settings `test-command`,
   `test-report`, `test-paths`; `workline test check --base <ref>` runs
   the suite and refuses a new test that passes. Cases: a new test red,
   passes; one green, refused; no report, blocks; no command,
   `setting-missing`.
2. **Tests from the criteria**: `pre` reads the acceptance criteria and
   the Scope's tests, writes `task.md`; a `test-file` intention;
   `apply` holds the test paths, runs step 1, pushes the branch, opens
   the Draft. Cases: one test a scenario; a product file refused; agent
   none says so; a missing scenario asked.
3. **Frozen guard**: the state block's hashes; `tester-test-changed`;
   the label lifts it. Cases: a changed test blocks; a deleted one too;
   the label lifts; a new test beside them passes.
4. **Moment and hand-over**: the scheduled run on `ready` issues; the
   developer starts from the branch (ADR-0054). Cases: routing, one
   issue a run, an open Draft skipped.
5. **Green proof**: three runs on the closing pull request; flaky said.
6. **Curator measures, no AI**: run times, budget, slowest tests; an
   issue proposing the project's subset and coverage tools.
7. **Redundancy**: per-test coverage, diff mutation, the AI's judgement,
   removal pull requests with proof.
8. **Five whys skill**, loaded by the tester first.
9. **Job 3, adapt after a change**: once the developer is built.

## Consequences

- **The developer gets its target before it starts**, and cannot move
  it without a person's click.
- **A person building by hand gets the same**: a Draft with red tests.
- **Projects without a JUnit report or a test command** get a blocking
  finding until they set one: the tester never guesses.
- **The corpus is measured weekly**; a removal always carries its proof.
- **Not covered**: a human test campaign plan, UI tests beyond the
  project's own runner, observability, `unsafe` and CVE audits (the
  security role, [#204](https://github.com/JN0V/workline/issues/204)).

[ADR-0018]: 0018-the-product-owner.md
[ADR-0027]: 0027-derived-blocks-are-regenerated-on-the-default-branch.md
[ADR-0038]: 0038-the-product-owner-proposes-on-the-issue-a-person-answers-there.md
[ADR-0039]: 0039-issue-sections-are-read-by-meaning-the-kind-is-the-forges.md
[ADR-0042]: 0042-the-engine-merges-by-the-projects-policy.md
[ADR-0043]: 0043-a-try-is-told-in-its-pull-request.md
[ADR-0050]: 0050-each-act-has-its-moment-and-a-wait-budget.md
[ADR-0051]: 0051-a-check-sits-where-its-catch-is-worth-its-cost.md
