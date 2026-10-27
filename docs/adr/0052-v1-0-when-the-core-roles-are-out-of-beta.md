# ADR-0052: v1.0 when the core roles are out of beta

- **Status:** accepted
- **Date:** 2026-10-26
- **Decided by:** the maintainer, on 2026-10-26
- **Builds on:** [ADR-0036] (a role's status is earned by written
  criteria), [ADR-0051] (a check sits where its catch is worth its cost)
- **Research:** [catch-and-cost.md](../research/catch-and-cost.md)
- **Tracked by:** the GitHub milestone
  [v1.0](https://github.com/JN0V/workline/milestone/1)

## Context

- **The line grew faster than it settled**: seven roles, most of them
  still beta by their own criteria ([ADR-0036]).
- **Upkeep ate the work**: 42% of merged PRs in W43, from 9% in W40;
  escapes stayed flat as checks came in
  ([catch-and-cost.md](../research/catch-and-cost.md)).
  [ADR-0051] cut checks to bring it down; the weekly measure of [#610]
  tells whether it did.
- **No version said "done"**: each release added, none closed a stage.

## Decision

v1.0 is reached when all three hold:

1. **The six core roles are out of beta**: committer, documentalist,
   reviewer, product owner, judge, auditor; each by the criteria of
   its `roles/<name>/docs/status.md` ([ADR-0036]). A role built once
   counts only while its status.md says built: a role can go back to
   beta.
2. **The inspector is validated on a real GitLab**, not only on GitHub;
   it is not one of the six.
3. **Upkeep stays under 10% of tokens for 4 weeks in a row**, by the
   weekly measure of [#610].

Until then, the line takes only:

- **Bug fixes.**
- **Work that moves a role toward leaving beta**: an item of its
  status.md's "Missing" or "To leave beta".

No new feature, no new role. The milestone v1.0 holds the issues that
count.

## Consequences

- **The planned roles wait**: security and the others
  ([the roles](../../README.md#the-roles)) start after v1.0; the
  developer is the one exception (amendment below).
- **An issue outside the milestone is parked**, unless it is a bug.
- **The measure must run**: without [#610]'s weekly rows, criterion 3
  cannot be met.

## Amendment 2026-10-27: the developer before v1.0

- **Decided by:** the maintainer, on 2026-10-27.
- **The developer ([#117]) is built before v1.0**, an exception to
  "no new role".
- **Why**: the reviewer's first criterion to leave beta, "Nothing
  buildable in Missing" ([its status.md]), waits for the developer's
  loop: the reviewer finds, the developer fixes. Without it, criterion
  1 cannot be met.
- **The developer is not one of the six core roles**: its own beta
  does not hold v1.0.
- **[#117] joins the milestone v1.0.**

[#117]: https://github.com/JN0V/workline/issues/117
[its status.md]: ../../roles/reviewer/docs/status.md
[ADR-0036]: 0036-a-roles-status-is-earned-by-written-criteria.md
[ADR-0051]: 0051-a-check-sits-where-its-catch-is-worth-its-cost.md
[#610]: https://github.com/JN0V/workline/issues/610
