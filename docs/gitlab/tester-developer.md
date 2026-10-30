---
sources: [ci/gitlab/workline.gitlab-ci.yml, internal/forge/gitlab.go, cmd/workline, Dockerfile]
checked: aea0b546
verified: agent:documentalist
judged: e8a46f7
---
# The tester and the developer on GitLab

Part of [workline on GitLab CI](../ci-gitlab.md). GitLab starts no
pipeline on a label or a comment: a tool given the project's webhooks
starts them ([pipelines a tool starts](../gitlab-trigger.md)).

## The tester

- **Its scheduled run**: list `tester` in the `schedule` line; the
  gardening jobs run it. Install your test runner in their image.
- **An objection** on its Draft: GitLab starts no pipeline on a comment.
  A tool given the note webhook starts one on the default branch, with
  `WORKLINE_TASK=objection`, `MR_IID` and `MR_SOURCE_BRANCH`
  ([the variables](../gitlab-trigger.md#how-the-pipeline-knows-what-to-do));
  the jobs `workline:objection:*` answer it
  ([what each job holds](../objection-jobs.md)).

## The developer

- **Its scheduled run**: list `developer` in the `schedule` line; the
  gardening jobs run it. Install your test runner in their image.
- **Its environment**: redefine `.workline:setup` in your
  `.gitlab-ci.yml`, its script your test job's setup commands; its
  judges run it before the agent and the proof
  ([the project's environment](../../roles/developer/docs/triggers.md#the-projects-environment)).
- **An issue labelled ready**: GitLab starts no pipeline on a label. A
  tool given the issue webhook starts one on the default branch, with
  `WORKLINE_TASK=develop` and `ISSUE_IID`; the jobs `workline:develop:*`
  run the `issue` line on it
  ([the triggers](../../roles/developer/docs/triggers.md)).
- **A review round** on its merge request: the note webhook again,
  `WORKLINE_TASK=review`, `MR_IID` and `MR_SOURCE_BRANCH`; the jobs
  `workline:review:*` run it
  ([the review loop](../../roles/developer/docs/review-loop.md#what-wakes-it)).
- **Its bounds** on its merge request: the job `workline:bounds`, and
  what keeps a merge request from editing it out, by tier
  ([the bounds on GitLab](bounds.md)).
