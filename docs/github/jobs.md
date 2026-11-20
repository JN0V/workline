---
sources:
  - defaults/ci/github/workline-loop.yml#jobs.loop
  - .github/workflows/workline-apply.yml#jobs.wake
  - defaults/ci/github/workline-gardening.yml#jobs.propose.permissions
  - defaults/ci/github/workline-gardening.yml#jobs.apply.permissions
  - defaults/ci/github/workline-sample.yml#on
  - defaults/ci/github/workline-sample.yml#jobs.read.permissions
  - internal/sample/verdict.go#noJudge
  - defaults/ci/github/workline-fork.yml#jobs.comment
checked: e2e165c2d
verified: agent:claude-code
judged: 220b6dc
---
# What the GitHub jobs may do

Part of [workline on GitHub Actions](../ci-github.md): each job's rights, its locks, and what it writes.

- **`propose`**: what it reads, when it runs, and its locks
  ([the `propose` job](propose-job.md)).
- **The jobs that write**: `apply`, `follow` and each pair's second job
  ([the jobs that write](write-jobs.md)).
- **On a merge queue's group** (`merge_group`): what `gates`, `propose`
  and `queue` do there ([a merge queue](../merge-queue.md#github-a-merge-queue)).
- **The loop** (`workline-loop.yml`): `actions: write`, to start what
  `workline tick` lists and its own successor; contents, issues and pull
  requests read; no agent, no App token ([the loop](../tick.md#the-github-loop)).
- **`wake`**, workline's own, on a push to `main`: `issues: write`,
  `pull-requests: read`, `statuses: read`, the App's token when there is
  one; it prints where the line may drift and never fails
  ([the wake](../wake.md)).
- The product owner's issues: read when judging, write when applying —
  the gardening template gives both.
- A GitHub call that fails for a moment is tried again first, a write
  only when GitHub did not take it ([which](calls.md)).
- **The sample**: the propose job is the repository variable `WORKLINE_JUDGE`,
  or the `sample.judge` setting; its reading job reads the issues and pull
  requests, for every role's acts. For another week than the last whole
  one, run the workflow by hand (Actions → workline sample → Run workflow)
  with its `week` input (`2026-W40`).
- **The summary**: each job's summary page (`$GITHUB_STEP_SUMMARY`), the
  applying job's saying what it did and what failed; a fork's pull request
  gets the propose job's as a comment
  ([what a job shows](../ci.md#what-a-job-shows)).
