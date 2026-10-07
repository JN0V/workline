# ADR-0036: A role's status is earned by written criteria

- **Status:** accepted
- **Date:** 2026-10-07
- **Builds on:** [ADR-0014](0014-checked-is-earned-by-what-was-read.md)
  (a fix that vouches, measured); principles 7, 13, 14

## Context

- Roles were marked `beta` or `built` by feel: README's table said the
  documentalist was `built`, its own page `beta`. The maintainer asked
  whether they were finished or not; no written rule answered.
- Others name maturity by stage, each stage with its promise:
  [Kubernetes feature stages](https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/#feature-stages)
  (alpha, beta, GA) and
  [semver pre-releases](https://semver.org/#spec-item-9) (`-beta`: may not
  be stable). We keep three stages and write what earns each one.

## Decision

- **planned**: an issue, no code.
- **beta**: released and running on real use, with known gaps in its
  status.md's "Missing".
- **built**: all three hold.
  1. Nothing buildable is left in its status.md's "Missing": only tries
     on real use, or items deferred and parked as issues.
  2. The maintainer has validated the role's main issue on real use (its
     Validation section).
  3. When it gives AI verdicts, they are measured (the reviewer's bench, the five
     concordant runs of [ADR-0014](0014-checked-is-earned-by-what-was-read.md)), the result recorded in status.md or
     tried.md.
- **Who moves a status**: the maintainer, in a pull request. Until then
  the role's status.md has a "To leave beta" list: each criterion still
  open, with a link. README's table and each role's page say the same
  word.

## Consequences

- Applied on 2026-10-07, every role is `beta`: none has a main issue
  validated yet. The committer, the judge and the auditor get a short
  status.md for their list.
- The documentalist goes back to `beta` in README's table, as its page
  said; the committer, the judge and the auditor leave `built`.
