---
sources:
  - defaults/ci/gitlab/workline.gitlab-ci.yml#.workline:summary
  - defaults/ci/gitlab/workline.gitlab-ci.yml#.workline:tally
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:propose
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:apply
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:merge
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:follow
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:release
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:release:dry-run:post
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:main
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:main:issue
checked: e81df7b34
verified: agent:claude-code
---
# What a GitLab job shows

Part of [what the GitLab template does](template.md).

- **The summary** ([what a job shows](../ci.md#what-a-job-shows)):
  `workline-summary.html`, linked from the job's page (its annotations)
  and, for a merge request, from the merge request's page (`expose_as`);
  `workline-summary.md` printed at the end of the log. The applying
  job adds what it applied to the proposing job's. The HTML opens through
  GitLab Pages, on by default on gitlab.com; without Pages, the link
  downloads it.
  - **A job that fails says why**: its log ends with a line
    `ERROR: workline ended with status N: …` and the first 2000
    characters of its summary.
  - **Kept as artifacts** a day, but:
    - a week: `workline:merge`, `workline:follow`, `workline:release`,
      `workline:release:dry-run:post`, `workline:main:issue`;
    - two weeks: each propose job and apply, whose agent's calls the weekly
      sample tallies;
    - 30 days: `workline:main`, the next base `workline:gates` reads.
  - Tried on gitlab.com (2026-10-06), the engine of the branch: a
    scheduled gardening job, the sample's read, and a gardening job
    started through the pipelines API each printed it, and the job's page
    linked the HTML, which Pages showed.
  - Not tried: a merge request's `expose_as` link, an applying job's page,
    a private project's Pages preview, a self-managed instance.
