# ADR-0073: tests that read the docs as they stand run weekly

- **Status:** proposed, until the maintainer merges it
- **Date:** 2026-11-23
- **Proposed by:** a contributor, from an independent analysis of the
  suite's wall time and red runs; the maintainer approved the move
- **Amends:** [ADR-0051](0051-a-check-sits-where-its-catch-is-worth-its-cost.md)
  cut 3 (no docs obligation per pull request)

## Context

- Cut 3 of ADR-0051 says: no zero-suspect rule on a pull request; one
  docs pull request a week; "nothing false stays past a week's docs
  PR".
- Some tests still enforce that rule on each pull request: they read this
  repository's docs as they stand and fail on a stale `checked` stamp,
  a source that moved, a name the code lost.
- Measured on a loaded machine (see the pull request's `## Tried`):
  the `TestUserPages*` tests dominate `cmd/workline`, minutes against
  seconds for the rest of the package.
- The analysis behind this decision counted 30 of 65 red runs on
  non-draft pull requests in 2.5 days, and no code defect caught.

## Decision

- **Gated, not deleted**: the tests that read the docs as they stand
  call `docsTestsOnly` and skip unless `WORKLINE_DOCS_TESTS` is set:
  `TestUserPages*`, `TestEveryCheckedIsAnAncestorOfMain`,
  `TestMergeReleasePagesNotSuspect`, `TestMergeReleaseSourcesNarrowAndNamed`.
- **Kept in the required suite**: tests that build a synthetic repository
  to prove an engine behaviour (`TestDocsCheckRefusesCheckedNotAnAncestor`,
  `TestNarrowSourcesRelatedChange`). They catch engine defects, not stale docs.
- **Weekly**: the docs run of the gardening workflow runs the gated tests
  with `WORKLINE_DOCS_TESTS=1` and says a failure in one open issue,
  closed when a later run passes. Never a red pull request.
- **Unchanged**: the documentalist's own `suspect` report.

## Consequences

- A pull request's `test` no longer waits on the docs tests.
- A stale doc is found within a week, as cut 3 already accepts.
- Run them by hand before a release: `WORKLINE_DOCS_TESTS=1 go test ./cmd/workline/ ./tests/`.
