---
sources:
  - defaults/ci/github/workline.yml#on
  - defaults/ci/github/workline.yml#jobs.propose.steps
  - defaults/ci/github/workline-apply.yml#jobs.queue
  - internal/engine/queue.go#queueGroup
  - .github/workflows/workline.yml#on
checked: e2e165c2d
verified: agent:claude-code
---
# A merge queue on GitHub

Part of [a merge queue or a merge train](../merge-queue.md).

- **Where**: an organisation's repository, public, or private on
  Enterprise Cloud
  ([changelog](https://github.blog/changelog/2023-07-12-pull-request-merge-queue-is-now-generally-available/)).
  A user's repository has none, workline's own included: its
  `workline.yml` has no `merge_group`.
- **The trigger**: [workline.yml](../../defaults/ci/github/workline.yml) runs on
  `merge_group` too. Without it, the checks the queue requires never
  report on its commit, and it waits until its timeout
  ([GitHub's docs](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/configuring-pull-request-merges/managing-a-merge-queue)).
- **The checks**: `propose` reads the gates' verdict on the group's
  commit from the `gates` job, and gives it: the check the ruleset
  requires, as on a pull request.
- **No agent installed**: `propose` installs one on a pull request only,
  never on a group.
- **With the [merge policy](../merge-policy.md)**: the base requires
  `workline/accepted` from the App, on the group's commit too.
  [workline-apply.yml](../../defaults/ci/github/workline-apply.yml)'s `queue` job
  sets it there as the head of the pull request the group ends with
  holds it; a head with no yes fails the group.
- **Set up**: Settings → Rules, on the base: "Require merge queue", and
  the checks required as without a queue.
- **The engine**: [from v0.38.0](../../CHANGELOG.md#0380-2026-10-20).
