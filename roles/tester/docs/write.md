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
# Write an issue's tests

Part of [the tester](../README.md): what `workline test write` does,
from the issue to the Draft. What it refuses is in
[refusals of a write](write-findings.md).

```sh
workline test write --issue 42 --ai claude
```

## The issue and the run

- **The agent is a choice**: none, `claude`, `cmd:<any command>`
  (`--ai`, else the project's `ai` setting). With none, nothing is
  written: `no-agent` says so, and the issue waits for its tests.
- **The issue**: labelled `workline:needs-tests` (or `workline:ready`
  with no Draft), its Validation a list of
  named scenarios, as the product owner writes them
  ([writing an issue](../../product-owner/docs/writing.md)):
  `- **Old debt:** Given …, When …, Then …`; a `**Scenario: Old debt**`
  or `Scenario: Old debt` line with its steps below is read too.
- **The test command runs first**, on the base, a minute at most: one
  that fails with no report (an SDK not found, a tool missing) stops the
  run before the agent, its error lines quoted (`test-command-fails`);
  `workline doctor` runs it on HEAD the same way
  ([#1333](https://github.com/JN0V/workline/issues/1333)).

## What the agent reads

- **What the agent reads**: the scenarios, the Need, a person's
  Verification and the reviewer's findings on it, the Scope, the code
  the issue names, the tests beside them, the test command.
  - The code is found by [the code
    picker](../../../docs/spec/code-picker.md): the Scope's files, a
    folder's own, a pattern's matches; each definition of a name its
    title, Need or Scope writes as code; a long file by those pieces.
  - Long lines over the context budget: fewer lines given; still over,
    the issue says so and names `context.budget`, and goes to Blocked,
    where the product owner reads it at once to narrow its Scope
    ([routing](../../../docs/spec/routing.md#the-line-of-states)).
  - What was cut is said in the run (`input-cut`), with the setting that
    raises it.
- **With `claude`, it reads the project too**: a copy at the issue's
  base, read-only (Read, Glob, Grep; no shell), the picker's code its
  start; it finds the test helpers itself, and a helper it could not
  confirm is listed on the Draft as assumed. What it read is logged
  with the call
  ([ADR-0063 §8](../../../docs/adr/0063-every-open-issue-is-due-waiting-or-done.md#8-one-code-picker-and-tools-where-the-agent-has-them)).
  Another agent reads its task alone, and names the files it lacks
  (`needs:`), given at once.

## The tests it writes

- **One test a scenario, declared**: the agent says which test covers
  which scenario, a `covers` entry each; one test may cover several.
  - The id is the test's as the report names it; a parametrised test by
    its name for all its rows.
  - Each id must be one of the new tests red on the base, or, the build
    broken with no report, a test a new file declares, or that file
    alone where no `test-name` names its tests
    ([#1293](https://github.com/JN0V/workline/issues/1293)); nothing is
    read from a test's name.
  - A scenario no test can check gets a reason instead, listed in the
    Draft as not tested, for a builder to object to
    ([ADR-0065](../../../docs/adr/0065-code-only-what-converges-language-knowledge-in-packs.md#4-the-agent-says-which-test-covers-which-scenario)).
- **Where the project keeps its tests**: what `test-paths` names, never
  the code. A new file, or one already there, as the agent judges: to
  that one it only adds lines, none changed nor removed.
- **Proven red** by [the check](check.md), on the base with the tests alone.
- **In Go, a name the code lacks** breaks the package's build: no new
  test reported, `no-new-test` blocks; a bug's test, of code that
  exists, is proven ([status](status.md#missing)).
- **The Draft**: branch `workline/tester/issue-<n>`, `Refs #<n>`, a
  `## Tests` table of each scenario and its declared tests. A person builds on
  that branch; the [developer](../../developer/docs/pull-request.md), on top of it.
- **The issue's Verification**: written from the tests, each scenario
  and its test, as a tester writes the test plan; one a person wrote is
  kept ([how](verification.md)).
