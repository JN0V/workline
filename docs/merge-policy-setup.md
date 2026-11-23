---
sources:
  - defaults/ci/github/workline-merge.yml#on
  - defaults/ci/github/workline-apply.yml#jobs.verdict
  - defaults/ci/github/workline-apply.yml#jobs.queue
  - defaults/ci/github/workline-apply.yml#jobs.follow
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:merge
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:follow
checked: 3f432ab14
verified: agent:claude-code
---
# Set up the merge policy

Part of [the merge policy](merge-policy.md): what each forge needs before
`workline follow` can turn auto-merge on.

## Per forge

- **GitHub**: the [templates](ci-github.md) — `workline-apply.yml`'s
  `verdict` and `follow` jobs run after `propose`, `queue` after a merge
  queue's group, from the default branch;
  `workline-merge.yml` runs when a label is put or taken off, or a draft
  made, and hourly; on a demotion issue's close; at `enterprising`, on a
  candidate's yes or close too.
  Then:
  - Settings → General → "Allow auto-merge";
  - the [GitHub App](ci-github-app.md), with **Commit statuses** read and
    write besides its other permissions, its key in the environment
    `workline-app`; a merge made with the job's own token starts no
    workflow on `main` — nothing is turned on then, said;
  - a ruleset on the base requiring the checks, and `workline/accepted`
    from the App; "Require branches to be up to date" left off; its
    bypass list empty ([nobody bypasses `main`](merge-policy-no-bypass.md)).
- **GitLab**: the [template](ci-gitlab.md)'s `workline:merge` job, after
  the propose job; the project bot's token; `workline:follow` on the hourly
  reconcile schedule too. Merged results pipelines on
  (Premium): without them a merge request's pipeline tests its branch
  alone. A label starts no pipeline there: it
  waits for that merge request's own next pipeline.
- **Another forge** (`cmd:`): the operations `merge-state`, `auto-merge`,
  `guarded`, `accept`, `keep-verdict` and `verdict`
  ([forge command](spec/forge-command.md)); at `normal`, more
  ([set up](merge-policy-reverts.md#set-up)).
- **The local forge** (`forge: local`): nothing to set up. It has no
  auto-merge, so `workline follow` merges itself what the policy says
  yes to, in the clone, nothing pushed
  ([the line with no forge](local-line-merge.md)).
