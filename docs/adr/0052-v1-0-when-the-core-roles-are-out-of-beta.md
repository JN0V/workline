# ADR-0052: v1.0 when the core roles are out of beta

- **Status:** accepted; V1 accepted on 2026-11-11
  ([amendment](#amendment-2026-11-11-1266-v1-accepted)); its freeze
  lifted on 2026-11-14 ([amendment](#amendment-2026-11-14-the-freeze-is-lifted));
  v1.0 is the V1 accepted, its other criteria after 1.0, on 2026-11-15
  ([amendment](#amendment-2026-11-15-v10-is-the-v1-accepted))
- **Date:** 2026-10-26
- **Decided by:** the maintainer, on 2026-10-26
- **Builds on:** [ADR-0036] (a role's status is earned by written
  criteria), [ADR-0051] (a check sits where its catch is worth its cost)
- **Research:** [catch-and-cost.md](../research/catch-and-cost.md)
- **Tracked by:** the GitHub milestone
  [v1.0](https://github.com/JN0V/workline/milestone/1)
- **Amended by:** [ADR-0068](0068-the-line-steps-back-on-its-own-drift.md#9-the-old-brakes-hold-until-replaced)
  (step 1 built): the "until then" freeze held until ADR-0068's step 1 passed its
  test (nobody bypasses `main`, the guard guarded); lifted on
  2026-11-14, after [#1463] and [#1464]
  ([amendment](#amendment-2026-11-14-the-freeze-is-lifted)). Its step
  7 now decides the bar after 1.0
  ([amendment](#amendment-2026-11-15-v10-is-the-v1-accepted))

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

## Amendment 2026-11-11 ([#1266]): V1 accepted

- **Decided by:** the maintainer, on 2026-11-11, after round 3 of the
  acceptance ([its report][round 3]) and the fix of [#1370] in [#1375].
- **What V1 is**: the product owner, the tester and the developer take a
  need to a merged change alone, on any repository ([#1266], defined by
  the maintainer on 2026-11-09). Its Validation is the acceptance round:
  three real repository copies (Kotlin, C++, Python), a small bug, a big
  bug and a feature on each.
- **The three rounds**, one pinned engine each:

  | Round | Reviewed, green | Merged | Silent stalls |
  |---|---|---|---|
  | 1 | 5 of 9 | 5 | 3 |
  | 2 | 9 of 9 | 7 | 2 |
  | 3 | 9 of 9 | 9 | 0 |

- **Round 3 passes every criterion**: 6 bugs merged by the policy with
  no person's yes, 3 features after a yes, no fix specific to one
  language, the tester right the first time on 3, 2 and 3 of 3.
- **After V1, until v1.0**: the Decision above still rules (the
  maintainer, 2026-11-12; lifted on 2026-11-14, below): bug fixes and work that moves a role out of
  beta only, no new role. The inspector on GitLab and the upkeep
  measure (criteria 2 and 3) are part of it.
- **After v1.0**: optimisation, token savings and new roles, the next
  step [#1266] names for after V1.

## Amendment 2026-11-14: the freeze is lifted

- **Decided by:** the maintainer, on 2026-11-14.
- **[ADR-0068]'s step 1 passed its test**
  ([§6](0068-the-line-steps-back-on-its-own-drift.md#6-nobody-bypasses-on-main)):
  - nobody bypasses `main` ([#1458], built in [#1463]): a real
    `gh pr merge --admin` without `workline/accepted` was refused; a
    labelled pull request then merged through workline's App;
  - the guard guarded ([#1459], built in [#1464]): the files that
    define oversight and authority listed in `CODEOWNERS`, merged on
    the person's label only.
- **The "until then" list no longer rules**: the line takes its work
  by its own policy, a guarded file on the person's label
  ([ADR-0068 §9](0068-the-line-steps-back-on-its-own-drift.md#9-the-old-brakes-hold-until-replaced)).
- **What still stands**: the v1.0 bar until ADR-0068's step 7;
  [ADR-0063]'s standing stop until its step 3 ([#1461]), lifted on
  2026-11-14 ([its amendment](0063-every-open-issue-is-due-waiting-or-done.md#amendment-2026-11-14-the-standing-stop-is-lifted)).

## Amendment 2026-11-15: v1.0 is the V1 accepted

- **Decided by:** the maintainer, on 2026-11-15.
- **v1.0 is the V1 accepted in [#1266]**: the product owner, the tester
  and the developer, with the reviewer and the committer and judge they
  rely on, take a need to a merged change on any repository
  ([above](#amendment-2026-11-11-1266-v1-accepted)).
- **Why**: V1 is the stage the line set out to close; the Decision's
  three criteria measure what comes after it.
- **Moved after 1.0**, to the bar [ADR-0068's step 7](0068-the-line-steps-back-on-its-own-drift.md#next-each-decided-when-reached)
  decides for after 1.0 (a 1.x or 2.0 bar):
  - all eight roles out of beta (criterion 1 and its amendment);
  - the inspector validated on a real GitLab (criterion 2);
  - upkeep under 10% of tokens for 4 weeks, by [#610] (criterion 3).
- **What 1.0 still requires**:
  1. **The maintainer signs "validated on real use"** (criterion 2 of
     [ADR-0036]) in the status.md of the product owner, the tester,
     the developer and the reviewer. Their other criteria to leave
     beta may stay open at 1.0.
  2. **[#1556] fixed**: the release's yes survives `main` moving, so
     the release pull request can be accepted.
  3. **The small items below**, fixed before 1.0 or after it, as
     listed.
  4. **The release**: a `Release-As: 1.0.0` footer reaches `main`
     through a pull request, then the maintainer's label on the
     release.
- **The small items**, judged against V1's goal (the chain alone, on
  any repository, a person's word never lost):

  | Issue | Side | Why |
  |---|---|---|
  | [#1291] | before | a tester's test lost unfrozen, silently |
  | [#1355] | before | an issue stalls in refine, silently |
  | [#1356] | before | a person's answer brings the same hold back |
  | [#1358] | before | a person's comment on a Draft unanswered at merge |
  | [#1364] | before | a person's long answer cut |
  | [#1382] | before | the developer skips an issue on GitHub |
  | [#1381] | after | GitLab parity, with the inspector's GitLab bar |
  | [#1319] | after | language packs: V1 passed on three languages without |
  | [#1346] | after | rare edge cases: a deleted comment, an upgrade |
  | [#1357] | after | a judge's miss, caught at a feature's yes |
  | [#1361] | after | the documentalist's noise, outside the chain |
  | [#1379] | after | the auditor's measure, outside the chain |
  | [#264] | after | its wider scenarios; its red-first part is the tester's signature |

[#1266]: https://github.com/JN0V/workline/issues/1266
[#1556]: https://github.com/JN0V/workline/issues/1556
[#1291]: https://github.com/JN0V/workline/issues/1291
[#1355]: https://github.com/JN0V/workline/issues/1355
[#1356]: https://github.com/JN0V/workline/issues/1356
[#1358]: https://github.com/JN0V/workline/issues/1358
[#1364]: https://github.com/JN0V/workline/issues/1364
[#1382]: https://github.com/JN0V/workline/issues/1382
[#1381]: https://github.com/JN0V/workline/issues/1381
[#1319]: https://github.com/JN0V/workline/issues/1319
[#1346]: https://github.com/JN0V/workline/issues/1346
[#1357]: https://github.com/JN0V/workline/issues/1357
[#1361]: https://github.com/JN0V/workline/issues/1361
[#1379]: https://github.com/JN0V/workline/issues/1379
[round 3]: https://github.com/JN0V/workline/issues/1266#issuecomment-6803875764
[#1370]: https://github.com/JN0V/workline/issues/1370
[#1375]: https://github.com/JN0V/workline/pull/1375

[#117]: https://github.com/JN0V/workline/issues/117
[#264]: https://github.com/JN0V/workline/issues/264
[its status.md]: ../../roles/reviewer/docs/status.md
[ADR-0036]: 0036-a-roles-status-is-earned-by-written-criteria.md
[ADR-0051]: 0051-a-check-sits-where-its-catch-is-worth-its-cost.md
[#610]: https://github.com/JN0V/workline/issues/610
[ADR-0063]: 0063-every-open-issue-is-due-waiting-or-done.md
[ADR-0068]: 0068-the-line-steps-back-on-its-own-drift.md
[#1458]: https://github.com/JN0V/workline/issues/1458
[#1459]: https://github.com/JN0V/workline/issues/1459
[#1461]: https://github.com/JN0V/workline/issues/1461
[#1463]: https://github.com/JN0V/workline/pull/1463
[#1464]: https://github.com/JN0V/workline/pull/1464
