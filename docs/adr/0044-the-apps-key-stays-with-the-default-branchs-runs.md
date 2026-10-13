# ADR-0044: The GitHub App's key stays with the default branch's runs

- **Status:** accepted, as [#331] asks
- **Date:** 2026-10-13
- **Decided by:** the maintainer, its direction ([#331], ready
  2026-10-13), found by an independent review of
  [ADR-0042](0042-the-engine-merges-by-the-projects-policy.md)
- **Completes:** [ADR-0042](0042-the-engine-merges-by-the-projects-policy.md),
  its limit on GitHub; a prerequisite of its `normal` level
- **Builds on:** [ADR-0010](0010-docs-judged-on-the-merge-request-and-by-gardening.md) (judge
  then apply, two trust zones); principles 6, 7, 12, 13

## Context

- **A pull request runs its own workflow.** On GitHub, a pull request from
  a branch of the repository runs the copy of the workflow its branch
  holds, with the repository's secrets. Whoever can push a branch — an
  agent through a person's account — can change it, or add one on `push`.
- **The App's key is the merge policy's trust.** `follow` sets
  `workline/verdict` and `workline/accepted` as the App; the base requires
  `workline/accepted` from the App; at `normal`, workline's own pull
  requests, opened and pushed by the App, merge with no person's yes. With
  the key, a pull request's code could do all of it.
- **Until now** `apply`, `verdict` and `follow` ran in the pull request's
  own run, with the key.
- **What GitHub has** ([changelog, 2025-11-07](https://github.blog/changelog/2025-11-07-actions-pull_request_target-and-environment-branch-protections-changes/)):
  - an environment's secrets reach only the jobs naming it, and its branch
    rule is checked against the run's ref — `refs/pull/<n>/merge` for a
    pull request's own run since 2025-12-08;
  - `workflow_run`, `push` to the default branch, `schedule` and
    `pull_request_target` run the default branch's workflow;
  - a `workflow_run` job reading the first run's artifacts reads what that
    run's code wrote: data, never instructions.

## Decision

- **The key lives in the environment `workline-app`**, which only the
  default branch may deploy to; no repository secret of that name. The
  App's ID stays a repository variable.
- **No job of a workflow a pull request runs holds it.** workline.yml
  keeps `judge` alone, a read token; it warns when the key reaches it, a
  repository secret.
- **The jobs that write move to workline-apply.yml**, run by
  `workflow_run` after workline.yml, and by a push to the default branch:
  `apply`, `verdict` and `follow`, each in the environment.
  workline-merge.yml (`pull_request_target`), gardening, the sample and
  workline's release-please name it too.
- **The pull request's run is read as data**:
  - its artifacts downloaded outside the checkout; the base checked out;
  - the head is the event's (`workflow_run.head_sha`): the verdict kept
    only for it; nothing applied once the branch moved on;
  - `workline apply --from-merge-request <n> --head <sha>`: only the
    agents' proposals are read. The runs are found by name; the
    repository, roles, forge and merge request are the job's. A run whose
    role does not run on a merge request, or proposing what its role never
    does, is refused whole. The patches go on the head, never through a
    link. Whether it is a release's is the base's settings' word.
- **A check**: no job of a pull request's workflow names the key; each
  job using it runs the default branch's workflow, in the environment;
  those after a pull request's run take the event's head, read its
  artifacts outside the checkout, never check its head out
  (TestGitHubAppKeyOutOfPullRequestRuns).

## Not chosen

- **`pull_request_target` for the judge**: it would run the pull request's
  code with the repository's secrets, the known "pwn request".
- **`apply` with the job's own token, in the pull request's run**: its
  push starts no check until someone approves the workflows.
- **Required reviewers on the environment**: every pull request's `apply`
  would wait for a click.
- **`deployment: false` on the jobs**: no deployment recorded, but
  GitHub's docs do not say the branch rule still holds then.

## Consequences

- **A project copies one more workflow**, and moves the secret once
  ([ci-github.md](../ci-github.md#set-up)); the templates need the first
  release after v0.26.1.
- **Each job holding the key records a deployment** to `workline-app`, on
  the default branch.
- **`apply` is no longer a check of the pull request**: its job is under
  "workline apply" in Actions; what it does, a commit or a comment, is on
  the pull request.
- **A pull request adding workline-apply.yml gets no `apply`**:
  `workflow_run` starts only from the default branch's workflow.
- **Left**:
  - the verdict is the pull request's run's word: one changing its
    workflow can write a passing verdict for its own head, as it can pass
    its own `judge`. At `cautious`, the person's yes covers that head; at
    `normal`, a routine pull request's every push is the App's;
  - a release pull request's run: whoever can push to the release tool's
    branch can change it, and its docs fix goes to a pull request the App
    opens, a `docs` kind at `normal`. A ruleset letting only the App push
    to that branch closes it;
  - GitHub Enterprise Server before the 2025-12-08 change checks the
    environment's branch rule against the pull request's head branch;
  - `CLAUDE_CODE_OAUTH_TOKEN` stays a repository secret: an agent's quota
    in reach, not a write;
  - GitLab, unchanged: the project bot's token is in reach of whoever runs
    a pipeline ([ADR-0042](0042-the-engine-merges-by-the-projects-policy.md#built-2026-10-12-321)).

[#331]: https://github.com/JN0V/workline/issues/331
