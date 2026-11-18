---
sources:
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:main
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:base
  - defaults/ci/gitlab/workline.gitlab-ci.yml#.workline:kept
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:gates
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:main:issue
checked: c46a93d6c
verified: agent:claude-code
---
# Main after a merge, on GitLab

Part of [main after a merge](../main-after-merge.md).

- **`workline:main`**, on a push to the default branch, holds no token,
  as `workline:gates`: it declares no environment, and unsets them
  still ([the environment](environment.md)). No Code Quality
  report there: the merge requests' widget compares with it, and old
  findings would read as fixed.
- **Main's own artifacts** hold the findings, never a cache: a merge
  request's pipeline can write a cache entry another reads
  (`cache:unprotect`), not the default branch's job's artifacts.
  - `workline:main` keeps `.workline-kept/` as its artifact, 30 days: the
    last 20 commits analysed, carried from its last successful pipeline.
  - `workline:base`, first in each pipeline, reads them,
    `jobs/artifacts/$CI_DEFAULT_BRANCH/raw/...?job=workline:main`: the
    merge base's `<commit>.json` on a merge request, the last 19 on the
    default branch, carried. Its artifact goes to `workline:gates` or
    `workline:main`, which never hold the token that read it.
  - It reads with `WORKLINE_GITLAB_TOKEN`, no tool run, nothing checked
    out; else with the job's token, said in its log, which gitlab.com's
    Free tier refuses there (404, tried).
- **Not found** (expired, a red pipeline's, a base older than those 20):
  the tools run again, said in the job's log and the summary. GitLab
  gives a branch's artifacts from its last successful pipeline only.
- **The issue** and the comments: `workline:main:issue`, with
  `WORKLINE_GITLAB_TOKEN`.
