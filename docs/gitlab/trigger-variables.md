---
sources:
  - docs/gitlab/without-template.md#ciworkline-jobsh
  - defaults/ci/gitlab/workline.gitlab-ci.yml#.workline.rules
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:base.rules
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:propose.rules
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:apply.rules
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:warn:child.rules
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:warn.rules
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:warn:apply.rules
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:merge.rules
  - defaults/ci/gitlab/workline.gitlab-ci.yml#.workline:garden.rules
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:garden:apply.rules
  - defaults/ci/gitlab/workline.gitlab-ci.yml#.workline:objection.rules
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:objection:apply.rules
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:issue.rules
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:reconcile.rules
  - defaults/ci/gitlab/workline.gitlab-ci.yml#.workline:develop.rules
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:develop:apply.rules
  - defaults/ci/gitlab/workline.gitlab-ci.yml#.workline:review.rules
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:review:apply.rules
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:release:dry-run.rules
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:release:dry-run:post.rules
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:curator.rules
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:curator:propose.rules
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:curator:apply.rules
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:release.rules
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:main.rules
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:main:issue.rules
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:follow.rules
  - defaults/ci/gitlab/workline.gitlab-ci.yml#.workline:sample.rules
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:objection:propose.script
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:review:propose.script
  - defaults/ci/gitlab/workline.gitlab-ci.yml#workline:develop:propose.script
checked: e2e165c2d
verified: agent:claude-code
reader: 735aaf4c142e
---
# How a started pipeline knows what to do

Part of [GitLab: pipelines a tool starts](../gitlab-trigger.md): the
variables a tool, or the webhook's URL, passes.

GitLab sets `CI_PIPELINE_SOURCE`: `trigger` (a trigger token), `api` (the
pipelines API), `pipeline` (another project's `trigger:` job),
`parent_pipeline` (a child pipeline); never `schedule` nor
`merge_request_event`.

So none of the template's jobs run, but:

- in its own child of a merge request's pipeline, what only warns;
- the line of states (`workline:issue`);
- the tester's answer to an objection, the developer's ready issue and
  its review round, below;
- `workline:release` on a tag.

The tool, or
the webhook's URL ([the webhook](../gitlab-trigger.md#a-label-or-a-comment-with-gitlab-alone)), passes
what to do as variables. The names below are this page's; the engine reads
none of them:

## The variables

| Variable | Value |
|---|---|
| `WORKLINE_TASK` | `merge-request`, `garden`, `objection`, `develop`, `review`, `product-owner`, `release`, `sample`, `follow` (when the default branch moved: the [merge policy](../merge-policy.md), when set), `main` (after a merge, `MAIN_BEFORE` its tip before: [main after a merge](../main-after-merge.md)), `import` |
| `MR_IID`, `MR_SOURCE_BRANCH`, `MR_TARGET_BRANCH`, `MR_BASE_SHA` | for `merge-request`: from the merge request (`diff_refs.base_sha`); the pipeline's `ref` is the source branch; the tool cancels the one still judging an older head first ([#359](https://github.com/JN0V/workline/issues/359)) |
| `MR_READY`, `MR_DRAFT` | for `merge-request`: `MR_DRAFT`, `true` or `false`, each run, `true` asking no agent; `MR_READY` when it becomes ready (opened not a draft, or an `update` whose `changes.draft` went false), a pipeline the template never gets: every reviewer lens once ([how](../../roles/reviewer/docs/ready.md)) |
| `MR_IID`, `MR_SOURCE_BRANCH` with `WORKLINE_TASK=objection` | the tester's Draft: a note opened by `workline:objection` on a merge request from `workline/tester/…` (the note webhook); the pipeline's `ref`, the default branch. The template's jobs `workline:objection:*` run it: `workline test answer --merge-request "$MR_IID"` ([tester](../../roles/tester/README.md#frozen-tests-and-objections)) |
| `WORKLINE_TASK=issue` | the issue webhook's pipeline: `workline:issue` reads `$TRIGGER_PAYLOAD` ([the line of states](line-of-states.md)) |
| `WORKLINE_TASK=reconcile` | a pipeline schedule, hourly: `workline:reconcile` |
| `ISSUE_IID` with `WORKLINE_TASK=develop` | the issue whose step is due, from `workline:issue` or `workline:reconcile`, or a tool; none: the first ready issue nobody took. The template's jobs `workline:develop:*` run `workline route issue --target issue:<n>`, the role its state wakes ([developer](../../roles/developer/docs/triggers.md)) |
| `MR_IID`, `MR_SOURCE_BRANCH` with `WORKLINE_TASK=review` | the developer's merge request: the reviewer's note on a merge request from `workline/developer/…`, or a person's (the note webhook). The template's jobs `workline:review:*` run `workline develop --merge-request "$MR_IID"` ([the review loop](../../roles/developer/docs/review-loop.md)) |
| `WORKLINE_IMPORT` | for `import`: the file moved to issues, as committed on the pipeline's `ref` |

## Passing them

A triggered pipeline has none of the `CI_MERGE_REQUEST_*` variables: the
tool reads them from the merge request and passes them. Passing variables
needs Settings → CI/CD → Variables, "Minimum role to use pipeline
variables", at or below the role of the token's user.

```sh
# a trigger token (Settings → CI/CD → Pipeline trigger tokens)
curl -fsS -X POST --form token="$TRIGGER_TOKEN" --form ref=main \
  --form "variables[WORKLINE_TASK]=garden" "$GITLAB/api/v4/projects/$ID/trigger/pipeline"
# or from another project's pipeline
# workline: {trigger: {project: group/app, branch: main, strategy: depend}, variables: {WORKLINE_TASK: garden}}
```

## Tried

Tried on a pipeline the API started (2026-10-06):

- the proposing job of `garden`, without an agent, and its summary;
- `import`, both jobs, the engine of the branch and a planted answer;
- the other tasks, not yet.
