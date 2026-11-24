---
sources: [roles/tester/code/green.go, roles/tester/code/runcache.go, roles/tester/code/frozen.go, roles/tester/code/renamed.go, roles/tester/code/tester.go]
checked: e81df7b34
verified: agent:claude-code
---
# How the green proof runs

Part of [the green proof](green-proof.md), and of [the tester](../README.md).

## The base once

The merge base of the record's base and the head, to tell the
builder's own new tests (required green too) from the base's.

- **Its test ids alone are read**, so a run of the same tree is
  reused ([#1338](https://github.com/JN0V/workline/issues/1338)):
  - the ids a run found are kept in the main clone's
    `.git/workline/test-ids/`, keyed by the tree's hash, the command
    and the report;
  - kept only when the command exited 0, its report read: the whole
    suite ran;
  - written by each run of a clean tree: the tester's red proof on
    the base, the head's first run here (merged up to date, its tree
    is the next base);
  - found, the base is not run again, said on the run's error output;
    the head always runs.
  - Scoped to the tests' folders, the key is the command as run on
    the base, its folders those the base holds: the Draft's red
    proof wrote it.
  - A fresh clone keeps none: the base runs.
- **No report and exit 0 on the base** is an empty run, as Gradle's
  `NO-SOURCE` for a module with no test yet; the head's runs still
  need their report ([#2091](https://github.com/JN0V/workline/issues/2091)).
- **In CI**, the templates carry the folder from the tester's run to
  each pull request's guard, and to its next push, where no other
  branch writes ([#1407](https://github.com/JN0V/workline/issues/1407)):
  GitHub's cache, GitLab's artifacts
  ([GitHub](../../../docs/github/jobs.md),
  [GitLab](../../../docs/gitlab/review-jobs.md)); never a run where the
  developer ran. A file of another tree, command or report is not read.

## The head, and again when red

- **The head once**, in one fresh checkout. Each run is
  `test-command`, or `draft-command` on the folders when it is scoped
  (the folders, two bullets down), its report read as JUnit XML,
  stopped at `suite-max-seconds`, its last lines said
  ([#1425](https://github.com/JN0V/workline/issues/1425)).
- **Again only when a test is red**: one more run in the same
  checkout tells flaky (green in the other run) from red (red in
  both). Never a third: not "until it passes".
- **The tests' folders alone**, when `draft-command` names
  `{packages}`: the base and the head run it on the folders holding
  the tester's test files, said by `proof-scope`. The pull request's
  own test job runs the whole suite. Unset, naming no `{packages}`, or
  with test files in no folder, `test-command` runs whole
  ([ADR-0055, amended 2026-11-23](../../../docs/adr/0055-the-tester-writes-the-acceptance-tests-before-the-code.md#amendment-2026-11-23-the-green-proof-on-the-tests-folders-once)).
- **Why one checkout**: a second run there reuses the runner's build
  cache. On a copy of workline, a Go package: 4.5 s the first run,
  1 s the next.
- **A test that a run before it breaks** (a file it leaves, a port it
  holds) fails on a person's second run too: it counts as flaky.
