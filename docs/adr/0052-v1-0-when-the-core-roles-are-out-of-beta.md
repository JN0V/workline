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

1. **The v1.0 roles are out of beta**: the six core roles (committer,
   documentalist, reviewer, product owner, judge, auditor), then the
   tester and the developer (amendment below); each by the criteria
   of its `roles/<name>/docs/status.md` ([ADR-0036]). A role built
   once counts only while its status.md says built: a role can go back
   to beta.
2. **The inspector is validated on a real GitLab**, not only on GitHub;
   it is not one of the six.
3. **Upkeep stays under 10% of tokens for 4 weeks in a row**, by the
   weekly measure of [#610].

Until then, the line takes only:

- **Bug fixes.**
- **Work that moves a role toward leaving beta**: an item of its
  status.md's "Missing" or "To leave beta".

No new feature, no role outside the v1.0 set. The milestone v1.0
holds the issues that count.

## Consequences

- **The other planned roles wait**: security and the others
  ([the roles](../../README.md#the-roles)) start after v1.0; the
  tester and the developer are in the v1.0 set (amendment below).
- **An issue outside the milestone is parked**, unless it is a bug.
- **The measure must run**: without [#610]'s weekly rows, criterion 3
  cannot be met.

## Amendment 2026-10-28: the tester and the developer are in v1.0

- **Decided by:** the maintainer, on 2026-10-28; replaces the
  amendment of 2026-10-27, which built the developer before v1.0 but
  left its beta out of criterion 1.
- **v1.0 holds eight roles out of beta**: the six core roles, the
  tester ([#264]) and the developer ([#117]).
- **Why**: the line from a need to merged code is not whole without
  the role that writes the acceptance tests and the role that writes
  the code. Without them, a person still writes both, and the line
  only checks.
- **The reviewer needs the developer too**: its first criterion to
  leave beta, "Nothing buildable in Missing" ([its status.md]), waits
  for the loop where the reviewer finds and the developer fixes.
- **[#117] and [#264] are in the milestone v1.0.**

[#117]: https://github.com/JN0V/workline/issues/117
[#264]: https://github.com/JN0V/workline/issues/264
[its status.md]: ../../roles/reviewer/docs/status.md
[ADR-0036]: 0036-a-roles-status-is-earned-by-written-criteria.md
[ADR-0051]: 0051-a-check-sits-where-its-catch-is-worth-its-cost.md
[#610]: https://github.com/JN0V/workline/issues/610
