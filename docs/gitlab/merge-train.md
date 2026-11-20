---
sources:
  - defaults/ci/gitlab/workline.gitlab-ci.yml#.workline
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:base
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:gates
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:propose.rules
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:apply.rules
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:warn:child.rules
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:merge
checked: e2e165c2d
verified: agent:claude-code
---
# A merge train on GitLab

Part of [a merge queue or a merge train](../merge-queue.md).

- **Where**: Premium and Ultimate, merged results pipelines on (Settings
  → Merge requests,
  [GitLab's docs](https://docs.gitlab.com/ci/pipelines/merge_trains/)).
- **A train's pipeline** (`CI_MERGE_REQUEST_EVENT_TYPE` is
  `merge_train`): [the template](../../defaults/ci/gitlab/workline.gitlab-ci.yml)
  runs `workline:gates` alone, against the target's tip, after
  `workline:base`, which reads that tip's findings main kept. A gate
  that blocks fails the pipeline, and the train drops the merge request.
- **A merged results pipeline** (`merged_result`) is the merge
  request's own, on each push: every job runs there, as on any other.
- **The merge policy**: `workline:merge` runs in the merge request's own
  pipelines, never in the train's; GitLab needs no status of
  workline's there.
