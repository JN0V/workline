---
sources: [ci/github, .github/workflows/workline.yml, .github/workflows/workline-apply.yml, .github/workflows/workline-gardening.yml, .github/workflows/workline-sample.yml, .github/workflows/workline-answer.yml, .github/workflows/workline-develop.yml, .github/actions/engine/action.yml, .github/workflows/release.yml, .github/workflows/release-please.yml, release-please-config.json, .goreleaser.yaml, cmd/workline]
checked: eebe4270
verified: agent:claude-code
judged: 220b6dc
---
# What the GitHub jobs may do

Part of [workline on GitHub Actions](../ci-github.md): each job's rights, its locks, and what it writes.

- `judge` reads the code and the pull request (`pull-requests: read`: the
  reviewer reads its own summary comment, to review only new commits,
  the one the App wrote, told by `WORKLINE_APP_ID`
  ([ADR-0051](../adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md)));
  it writes only code scanning's SARIF, without the documentalist's and
  the reviewer's findings: a conversation each to resolve
  ([#412](https://github.com/JN0V/workline/issues/412)).
- `judge` reads issues, checks, statuses and actions too: on a release
  tool's pull request, what holds the release
  ([what holds a release](../release-holds.md)).
- `judge` runs on a pull request opened, pushed to, reopened, and marked
  ready (`ready_for_review`). A draft asks no agent, `workline/verdict`
  pending until it is ready; opened ready or marked ready, the reviewer
  reads every lens once, on the whole change
  ([more](../../roles/reviewer/docs/ready.md)). An edit runs nothing: after
  fixing a refused title, re-run `judge` or push
  ([the squash title](../../roles/committer/docs/squash-title.md)).
- On a merge queue's group (`merge_group`), `gates` runs the gates on
  the merged result and `judge` gives their verdict: no agent, nothing
  applied; with the merge policy, `queue` sets the App's yes on the
  group's commit ([a merge queue](../merge-queue.md)).
- **One `judge` per pull request at a time** (`concurrency`): a push
  cancels the run still on the head before; the newer run reviews its
  commits too, and `apply`, `verdict`, `follow` and the fork's comment skip
  a cancelled run ([#359](https://github.com/JN0V/workline/issues/359)).
  A cancelled run on ready leaves every lens to the next run.
- **One review at a time**: `judge` waits while `apply` writes the review
  before it, then reviews only what is new; `apply` writes it even once
  the branch moved on ([#375](https://github.com/JN0V/workline/issues/375)).
- **The lenses that warn, apart**: `warn`, its own run and lock, never
  fails by what they find; `verdict` and `follow` run after `judge` alone
  ([ADR-0050](../adr/0050-each-act-has-its-moment-and-a-wait-budget.md#2-an-act-over-its-budget)).
- `apply`, `follow` and each pair's second job hold the write token, the
  App's when there is one, and no AI key, in the environment
  `workline-app`. `follow`'s merge into a pull request runs its checks
  again only with the App's token; the merge policy acts only with it,
  and needs "Allow auto-merge" (Settings → General).
- `apply` reads the pull request's run as data, only the agents'
  proposals ([more](../commands.md#workline-apply)), whatever `judge`
  concluded: one the reviewer holds still gets its comment
  ([#226](https://github.com/JN0V/workline/issues/226)). Its job is in
  Actions, "workline apply"; the check is `judge`'s.
- `apply` posts the reviewer's findings on their lines, a pull request
  review (`pull-requests: write`); a token without it leaves them in the
  summary comment, said
  ([more](../../roles/reviewer/docs/lines.md)).
- `apply` writes the documentalist's findings as the annotations of the
  check `workline findings`, the [gates'](../gates-in-ci.md) as those of
  `workline gates` (`checks: write`; an App's **Checks**): beside their
  lines, none a conversation; a refusal, said.
- The product owner's issues: read when judging, write when applying —
  the gardening template gives both.
- A GitHub call that fails for a moment is tried again first, a write
  only when GitHub did not take it ([which](calls.md)).
- **The sample**: the judge is the repository variable `WORKLINE_JUDGE`,
  or the `sample.judge` setting; its reading job reads the issues and pull
  requests, for every role's acts. For another week than the last whole
  one, run the workflow by hand (Actions → workline sample → Run workflow)
  with its `week` input (`2026-W40`).
- **The summary**: each job's summary page (`$GITHUB_STEP_SUMMARY`), the
  applying job's saying what it did and what failed; a fork's pull request
  gets the judge's as a comment
  ([what a job shows](../ci.md#what-a-job-shows)).
