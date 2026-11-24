---
sources:
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:base
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:gates
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:propose
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:warn:child
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:warn
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:warn:apply
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:merge
  - defaults/ci/gitlab/workline.gitlab-ci.yml#.workline:ids
checked: e81df7b34
verified: agent:claude-code
---
# A merge request's jobs on GitLab

Part of [what the GitLab template does](template.md).

- `workline:gates` runs your gates alone, before `workline:propose`, and
  writes what they found the change adds as its Code Quality report;
  install their tools in it ([gates in CI](../gates-in-ci.md)). No token:
  `workline:base`, before it, reads the merge base's findings
  ([the environment](environment.md)). In a merge train's
  pipeline, they are workline's only jobs; a merged results pipeline
  runs them all ([a merge train](../merge-queue.md#gitlab-a-merge-train)).
- **The reviewer's lenses that only warn** read apart from the verdict
  ([apart](../../roles/reviewer/docs/apart.md)):
  - `workline:propose` asks those that block (`--warn-apart`; without
    `WORKLINE_GITLAB_TOKEN`, every lens, said in its log and summary).
  - `workline:warn`, with the agent's token and the bot's, asks the
    others (`--warn-only`), and the inspector's triage given the gates'
    `gates.json` ([the triage in CI](../gates-triage.md));
    `workline:warn:apply` posts them, each in a comment of its own.
  - **Both in a child pipeline**, `workline:warn:child`, started after
    `workline:gates` whatever its verdict, with no `strategy`: "merge
    when the pipeline succeeds" waits for `allow_failure` jobs too, never
    for a child
    ([ADR-0051](../adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md#4-also)).
  - The child gets the merge request's `CI_MERGE_REQUEST_*` variables and
    the tokens scoped to `workline`. `workline:warn` reads `gates.json`
    from the merge request's pipeline with the bot's token: GitLab's
    `needs:pipeline:job` gives no failed job's artifacts.
  - `workline:merge` reads the verdict from `workline:propose` alone.
- **The tests' ids** the tester's guard reads, so it does not run the
  base again: `workline:base` reads them from the artifacts of the
  default branch's `workline:develop:propose` and of the merge request's
  own `workline:propose`, never from GitLab's cache, which every branch
  shares ([the green proof](../../roles/tester/docs/green-proof-runs.md#the-base-once)).
