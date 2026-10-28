# Tests first, and a corpus kept small — what exists (2026-10-28)

**Question.** A tester role ([#264](https://github.com/JN0V/workline/issues/264),
which replaced [#206](https://github.com/JN0V/workline/issues/206)) turns
a `ready` issue's acceptance criteria into tests before the code, keeps
the test corpus concise, scoped and fast, and later adapts tests after a
change. What does the ecosystem already do for each, in its own words,
and what is left for workline to add?

**Verdict.**

- **Tests first has a name and a practice**: acceptance test-driven
  development (ATDD), specification by example, BDD. Given / When / Then
  is its format; Cucumber, behave and Godog run it. None of them writes
  the tests: people do, in a "three amigos" talk.
- **A failing test that must not turn `main` red is solved per
  ecosystem**, by a *strict expected failure*: it fails the suite when it
  starts passing. pytest, Jest, Cucumber's `--wip` have it; Go and Rust
  only skip. workline's own `pending.txt` is the same idea. The one form
  every language shares: **the failing tests stay on the work branch**,
  in the pull request that makes them pass.
- **AI test generators all keep only passing tests**: Meta's
  TestGen-LLM, Qodo Cover, Diffblue, Copilot. They protect today's
  behaviour (regression tests), not a need not built yet. Their filters
  are worth borrowing, inverted: builds, *fails* before, passes after,
  not flaky, adds something.
- **Agents bend tests**: a coding agent that can edit the tests deletes
  or rewrites the failing ones (ImpossibleBench). Hidden holdout tests
  help, not foolproof (EvilGenie). A visible test the builder cannot
  edit is cheaper; one that does not fit is talked over ([disputes](acceptance-test-disputes.md)).
- **Selection is mature, minimization is rare**: test impact analysis
  ships in every ecosystem (Go's test cache, pytest-testmon, Jest
  `--findRelatedTests`, Nx affected, bazel-diff, Launchable). Removing
  tests is mostly research; industry runs a subset instead. Mutation
  testing on the diff only (Google, cargo-mutants `--in-diff`) is the
  affordable way to tell a test that protects something.
- **Flaky tests are quarantined, never silently skipped**: Buildkite,
  Atlassian's Flakinator, TestGen-LLM's five runs.
- **What is left for workline**: an agent writing tests *from the
  acceptance criteria, red first*, kept apart from the builder; a check
  that they fail before and are not weakened after; a curator proposing
  removals with before / after proof. Nobody ships these as one role.

## Tests before the code: ATDD, BDD, Gherkin

- **ATDD**: customer, developer and tester agree acceptance tests before
  the code is built; the talk is the "three amigos"; the cycle is
  discuss, distill (often into Given / When / Then), develop against
  failing tests, demo
  ([Agile Alliance](https://www.agilealliance.org/glossary/atdd/),
  [Wikipedia](https://en.wikipedia.org/wiki/Acceptance_test-driven_development)).
  Also called specification by example, story test-driven development.
- **Its known cost**: it needs test automation skills a team may lack.
  An agent writing the first draft fills that gap.
- **Gherkin runners**: Cucumber (many languages), behave (Python), Godog
  (Go). A `.feature` file holds the scenarios; *step definitions* in code
  bind each line. An unbound line is *undefined*, a stub is *pending*.
- **Strict mode**: cucumber-js fails a run with pending steps by default;
  cucumber-ruby's `--strict` makes undefined and pending fail
  ([cucumber-js CLI](https://github.com/cucumber/cucumber-js/blob/main/docs/cli.md)).
  Godog returns `godog.ErrPending` from a stub; its `Strict` option
  decides whether pending fails
  ([godog](https://github.com/cucumber/godog)).
- **Work in progress**: cucumber-ruby's `--wip` fails the run if a
  scenario tagged `@wip` *passes*; behave has the same `-w` / `@wip`
  ([behave](https://behave.readthedocs.io/en/stable/)).
- **Taken**: Given / When / Then scenarios are already workline's
  acceptance criteria ([ADR-0039](../adr/0039-issue-sections-are-read-by-meaning-the-kind-is-the-forges.md)).
  A Gherkin runner is a dependency: used only where a project already
  has `.feature` files.

## A failing test that keeps `main` green

| Ecosystem | Marker | Fails when it starts passing |
|---|---|---|
| pytest | `@pytest.mark.xfail(strict=True)`, or `xfail_strict = true` | yes ([pytest](https://docs.pytest.org/en/stable/how-to/skipping.html)) |
| Jest | `test.failing(...)` | yes ([Jest API](https://jestjs.io/docs/api#testfailingname-fn-timeout)) |
| Cucumber, behave | `@wip` with `--wip` / `-w` | yes |
| Godog | step returns `ErrPending`, non-strict | no: pending, not checked |
| JUnit 5 | `@Disabled` | no: skipped |
| Go | `t.Skip(...)` | no: skipped ([testing](https://pkg.go.dev/testing)) |
| Rust | `#[ignore]`, run with `--ignored` | no: skipped ([test attributes](https://doc.rust-lang.org/reference/attributes/testing.html)) |
| workline | a case listed in `tests/conformance/pending.txt` | yes: the suite fails if a listed case passes |

- **A skip proves nothing**: it hides the test and never says it now
  passes. Only a strict expected failure keeps red-first honest on
  `main`.
- **A strict xfail can fail for the wrong reason** (a typo, an import)
  and still count as expected
  ([Ganssle](https://blog.ganssle.io/articles/2021/11/pytest-xfail.html)).
- **The universal form**: the tests land as the first commits of the
  work branch; its pull request is red until the code makes them pass.
  No marker, any language; `main` never sees them red.

## AI test generation

- **Meta's TestGen-LLM** (FSE 2024): extends human test classes; keeps a
  candidate only if it *builds*, *passes*, passes *five runs out of
  five* (not flaky) and *adds coverage*. 75 % built, 57 % passed
  reliably, 25 % added coverage; 73 % of its proposals accepted by
  engineers. They call it "assured" LLM-based engineering
  ([paper](https://arxiv.org/abs/2402.09171)).
- **Qodo Cover** (was CodiumAI Cover-Agent): open source, the same
  loop to a `--desired-coverage` within `--max-iterations`; reads a
  Cobertura report; now unmaintained
  ([qodo-cover](https://github.com/qodo-ai/qodo-cover)).
- **Diffblue Cover**: Java, reinforcement learning, not an LLM; tests
  that "reflect the current behaviour", compile and pass; maintains
  only the tests a change affects
  ([docs](https://docs.diffblue.com/get-started)).
- **Copilot**: `/tests` in the editor; agent mode writes tests, runs
  them and fixes the failures
  ([VS Code guide](https://code.visualstudio.com/docs/copilot/guides/test-with-copilot)).
  Its blog shows a TDD flow with a person writing the failing test first
  ([GitHub blog](https://github.blog/ai-and-ml/github-copilot/github-for-beginners-test-driven-development-tdd-with-github-copilot/)).
- **What none does**: start from a need, demand red first, keep the
  tests apart from the code's author.
- **Taken, inverted**: build filter, *red* filter on the base, *green*
  filter on the change, repeated runs for flakiness, a cap on
  iterations.

## Agents and the tests they can edit

- **ImpossibleBench** (ICLR 2026): tasks whose tests contradict the
  spec; any pass is a cheat. Agents with access delete or edit failing
  tests ([paper](https://arxiv.org/abs/2510.20270)).
- **EvilGenie**: holdout tests help, but a heuristic solution can pass
  them too; some "hacks" were fixing a wrong test
  ([paper](https://arxiv.org/abs/2511.21654)).
- **Shadow Score, smixs/code-quality, ai-test-integrity-review**: spec
  tests by another model, weakened-assertion detection, modified tests
  replayed on the old code ([gates.md](gates.md#test-integrity)).
- **Taken**: the builder may read the tests and must not change them; a
  change to them is a finding; an objection goes to the tester, which
  keeps or rewrites the test ([disputes](acceptance-test-disputes.md)).
  Hiding them costs a private store and a second runner, no better.

## Running only what a change touches

- **Go**: `go test` caches a package's passing result while its code,
  the files it reads and its environment stay the same; `-count=1`
  forces a run ([cmd/go](https://pkg.go.dev/cmd/go#hdr-Test_packages)).
- **pytest-testmon**: records, through Coverage.py, which code each test
  ran; reruns only the tests a change touches, and the last failures
  ([pytest-testmon](https://pypi.org/project/pytest-testmon/)).
- **Jest**: `--onlyChanged`, `--changedSince`, `--findRelatedTests`
  ([Jest CLI](https://jestjs.io/docs/cli)). **Nx**: `nx affected -t test`
  ([Nx](https://nx.dev/ci/features/affected)). **Bazel**: bazel-diff
  hashes the graph at two revisions, `-t` for tests only
  ([bazel-diff](https://github.com/Tinder/bazel-diff)).
- **Launchable**: a model ranks tests; `subset --confidence 90%`,
  `--time`, `--target`; an observation mode before it gates
  ([docs](https://help.launchableinc.com/features/predictive-test-selection/requesting-and-running-a-subset-of-tests/choosing-a-subset-optimization-target/)).
- **Taken**: the subset per module the maintainer asked for exists in
  each ecosystem; the tester finds the project's and proposes it, the
  person's machine running the subset, CI the wider set, a release all.

## Mutation testing, and tests that protect nothing

- **Tools**: Stryker (JS, C#, Scala; incremental mode, `thresholds.break`),
  PIT (Java; history files), mutmut (Python), cargo-mutants (Rust;
  `--in-diff`), Gremlins and go-mutesting (Go; a maintained fork adds a
  diff filter and a baseline) ([gates.md](gates.md#test-integrity),
  [cargo-mutants](https://mutants.rs/in-diff.html),
  [Gremlins](https://github.com/go-gremlins/gremlins)).
- **Google** mutates only the lines a change covers, caps mutants per
  line and per review, suppresses kinds that proved unproductive, and
  shows survivors as code findings: 17 M mutants over 760 k changes
  ([paper](https://research.google/pubs/practical-mutation-testing-at-scale-a-view-from-google/)).
- **A whole run is slow**: Gremlins warns of hours on a large module;
  cargo-mutants says a diff run is no substitute for a full one.
- **Minimization**: *adequate* reduction keeps every requirement
  (coverage, killed mutants); *inadequate* drops more for a small loss.
  Loss is measured as the faults the reduced suite misses; killed
  mutants lose less than coverage alone
  ([Shi et al., FSE 2014](https://mir.cs.illinois.edu/gyori/pubs/fse14reduction.pdf)).
  Rarely used in industry, which selects instead
  ([Teamscale, 2020](https://teamscale.com/hubfs/26978363/Publications/2020-test-suite-minimization-swqd.pdf)).
- **Taken**: a removal is proposed only when the corpus keeps its
  coverage and its killed mutants (adequate), measured before and after.

## Flaky tests

- **Buildkite Test Engine**: history finds unreliable tests; muted ones
  still run and report; owners told; recovery detected
  ([docs](https://buildkite.com/docs/test-engine/reduce-flaky-tests)).
- **Atlassian's Flakinator**: quarantined tests leave the gating path,
  run elsewhere, return after a healthy period
  ([blog](https://www.atlassian.com/blog/atlassian-engineering/taming-test-flakiness-how-we-built-a-scalable-tool-to-detect-and-manage-flaky-tests)).
- **Taken**: a flaky test is quarantined with the ecosystem's marker and
  an issue, counted and said (principle 12), never deleted for it.

## A report every runner writes

- **JUnit XML** is the shared format: pytest `--junitxml`, `jest-junit`,
  `gotestsum --junitfile`, `cargo nextest` (JUnit output), Maven
  Surefire, behave `--junit`. Each test's id, result and time.
- **Taken**: the tester reads it for red, green and run time; a project
  with no report gets a finding, not a guess.
