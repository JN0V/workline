---
sources:
  - defaults/ci/github/workline.yml#on
  - defaults/ci/github/workline.yml#concurrency
  - defaults/ci/github/workline.yml#jobs.propose
  - defaults/ci/github/workline-apply.yml#jobs.apply.if
  - defaults/ci/github/workline-apply.yml#jobs.apply.concurrency
  - defaults/ci/github/workline-apply.yml#jobs.verdict.if
  - defaults/ci/github/workline-apply.yml#jobs.follow.if
  - defaults/ci/github/workline-fork.yml#jobs.comment.if
  - defaults/ci/github/workline-warn.yml#concurrency
  - defaults/ci/github/workline-warn.yml#jobs.warn.concurrency
  - tests/conformance/cases/reviewer/review-applied-when-a-push-lands-before-it.yaml
checked: e2e165c2d
verified: agent:claude-code
judged-in-parts: 06fe2e1
---
# The `propose` job on GitHub

Part of [what the GitHub jobs may do](jobs.md): what `propose` reads, when
it runs, and its locks.

- `propose` reads the code and the pull request (`pull-requests: read`: the
  reviewer reads its own summary comment, to review only new commits,
  the one the App wrote, told by `WORKLINE_APP_ID`
  ([ADR-0051](../adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md)));
  it writes only code scanning's SARIF, without the documentalist's and
  the reviewer's findings: a conversation each to resolve
  ([#412](https://github.com/JN0V/workline/issues/412)).
- `propose` reads issues, checks, statuses and actions too: on a release
  tool's pull request, what holds the release
  ([what holds a release](../release-holds.md)).
- `propose` runs on a pull request opened, pushed to, reopened, and marked
  ready (`ready_for_review`). A draft asks no agent, `workline/verdict`
  pending until it is ready; opened ready or marked ready, the reviewer
  reads every lens once, on the whole change
  ([more](../../roles/reviewer/docs/ready.md)). An edit runs nothing: after
  fixing a refused title, re-run `propose` or push
  ([the squash title](../../roles/committer/docs/squash-title.md)).
- `propose` reads and keeps the tests' ids the tester's guard reads, so it
  does not run the base again, in the Actions cache: its pull request's
  own, else the default branch's, which `workline-develop.yml` writes
  after the tester's run alone
  ([the green proof](../../roles/tester/docs/green-proof-runs.md#the-base-once)).
- **One `propose` per pull request at a time** (`concurrency`): a push
  cancels the run still on the head before; the newer run reviews its
  commits too, and `apply`, `verdict` and the fork's comment skip
  a cancelled run, `follow` one of a pull request ([#359](https://github.com/JN0V/workline/issues/359)).
  A cancelled run on ready leaves every lens to the next run.
- **One review at a time**: `propose` waits while `apply` writes the review
  before it, then reviews only what is new; `apply` writes it even once
  the branch moved on ([#375](https://github.com/JN0V/workline/issues/375)).
- **The lenses that warn, apart**: `warn`, its own run and lock, never
  fails by what they find ([apart](../../roles/reviewer/docs/apart.md)); `verdict` and `follow` run after `propose` alone
  ([ADR-0050](../adr/0050-each-act-has-its-moment-and-a-wait-budget.md#2-an-act-over-its-budget)).
