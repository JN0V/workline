---
sources:
  - defaults/ci/gitlab/workline.gitlab-ci.yml#.workline
  - defaults/ci/gitlab/workline.gitlab-ci.yml#.workline:bot-token
  - defaults/ci/gitlab/workline.gitlab-ci.yml#.workline:claude
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:judge
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:apply
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:merge
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:garden:apply
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:follow
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:main
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:main:issue
  - internal/builtin/documentalist/documentalist.go#preRun.sayShallow
checked: e2e165c2d
verified: agent:claude-code
judged: e8a46f7
---
# What the GitLab template does

Part of [workline on GitLab CI](../ci-gitlab.md): what each job of
[the template](../../defaults/ci/gitlab/workline.gitlab-ci.yml) does, and why.

- **A merge request's jobs**: the gates, the reviewer's lenses that only
  warn, the tests' ids ([a merge request's jobs](review-jobs.md)).
- **The whole history** (`GIT_DEPTH: 0`, over the project's "Git shallow
  clone" setting, 20 commits on a new project): the documentalist compares
  each doc's sources since its `checked`, the release looks for its last
  tag. A job set too shallow gets `shallow-clone`, never a pass.
- **A fork's merge request** runs in the fork, which has neither variable:
  judged without an agent, nothing applied or commented
  ([its gates](../gates-fork.md)).
- **A judge that fails**: the applying jobs still run (`when: always`),
  the pipeline failed by the judge's job.
  - Gardening: what steps passed before one blocked is applied
    ([#142](https://github.com/JN0V/workline/issues/142)).
  - A role that blocks still posts its comments, the reviewer's as the
    documentalist's ([`on-block`](../spec/role-contract.md#one-run)).
  - An agent out of reach (3) warns (`allow_failure: {exit_codes: [3]}`),
    as on GitHub: a person judges what it would have; a GitLab call is
    tried again first ([which](calls.md)).
- **`workline:follow`**, each push to the default branch and the hourly reconcile schedule, its bot's token,
  no agent: the [merge policy](../merge-policy.md), when set, also after
  each judge (`workline:merge`).
- **`workline:main`**, each push to the default branch, no token, no
  agent: the gates, their findings kept in its artifacts as the next base
  `workline:gates` reads, and the whole repository's checks;
  `workline:main:issue`, the bot's token, keeps what it found in one issue
  ([main after a merge](../main-after-merge.md#gitlab)).
- **`workline:garden:apply`** then rebuilds gardening's earlier merge
  requests, or closes those with nothing left to merge
  ([ADR-0034, amended](../adr/0034-the-release-fix-follows-its-base.md#amendment-2026-10-12-gardenings-merge-requests-follow-too-or-are-closed)).
- **The summary** each job shows, and how long it is kept
  ([what a job shows](job-summary.md)).
- **The records** the reviewer and the documentalist read back from
  their comments ([the records](records.md)).
