---
sources: [ci/gitlab/workline.gitlab-ci.yml, internal/forge/gitlab.go, cmd/workline, Dockerfile]
checked: 99a14dc0
judged: bf016b7
verified: agent:claude-code
---
# GitLab: pipelines a tool starts

The GitLab template ([ci-gitlab.md](ci-gitlab.md)) runs on merge request
events and pipeline schedules. Where an internal tool starts the jobs
instead — the trigger API, the pipelines API, another pipeline — this page
does the same with a plain script: the template's commands under other
rules. The commands themselves: [triggers.md](triggers.md).

Tried on a pipeline the API started (2026-10-06):

- the judging job of `garden`, without an agent, and its summary;
- `import`, both jobs, the engine of the branch and a planted answer;
- the other tasks, not yet.

## Before CI: on your machine

Everything runs locally first. `forge: gitlab` in `.workline/config.yaml`
and `glab auth login` (or `GITLAB_TOKEN` and `GITLAB_HOST`) let `workline
run-role product-owner --event schedule --ai none` read your issues from a
laptop; `forge: local` keeps issues and merge requests in the clone,
nothing pushed. The hooks, `workline docs` and `workline review` need no
forge.

## What Premium adds

- **Blocks links**: the product owner's `depend` writes GitLab's
  `is_blocked_by` link; on Free, GitLab refuses it and the issue's body
  gets `Blocked by #12.` instead. Both are read back.
- **Project and group access tokens** on gitlab.com (every tier on a
  self-managed instance): a bot user, the right identity for workline.
  A group token can serve several projects; untried with workline.

## How the pipeline knows what to do

GitLab sets `CI_PIPELINE_SOURCE`: `trigger` (a trigger token), `api` (the
pipelines API), `pipeline` (another project's `trigger:` job),
`parent_pipeline` (a child pipeline) — never `schedule` nor
`merge_request_event`, so none of the template's jobs run but in its own
child of a merge request's pipeline, what only warns, and the tester's
answer to an objection, the developer's ready issue and its review round, below. The tool passes
what to do as variables. The names below are this page's; the engine reads
none of them:

| Variable | Value |
|---|---|
| `WORKLINE_TASK` | `merge-request`, `garden`, `objection`, `develop`, `review`, `product-owner`, `release`, `sample`, `follow` (when the default branch moved: the [merge policy](merge-policy.md), when set), `main` (after a merge, `MAIN_BEFORE` its tip before: [main after a merge](main-after-merge.md)), `import` |
| `MR_IID`, `MR_SOURCE_BRANCH`, `MR_TARGET_BRANCH`, `MR_BASE_SHA` | for `merge-request`: from the merge request (`diff_refs.base_sha`); the pipeline's `ref` is the source branch; the tool cancels the one still judging an older head first ([#359](https://github.com/JN0V/workline/issues/359)) |
| `MR_READY`, `MR_DRAFT` | for `merge-request`: `MR_DRAFT`, `true` or `false`, each run, `true` asking no agent; `MR_READY` when it becomes ready (opened not a draft, or an `update` whose `changes.draft` went false), a pipeline the template never gets: every reviewer lens once ([how](../roles/reviewer/docs/ready.md)) |
| `MR_IID`, `MR_SOURCE_BRANCH` with `WORKLINE_TASK=objection` | the tester's Draft: a note opened by `workline:objection` on a merge request from `workline/tester/…` (the note webhook); the pipeline's `ref`, the default branch. The template's jobs `workline:objection:*` run it: `workline test answer --merge-request "$MR_IID"` ([tester](../roles/tester/README.md#frozen-tests-and-objections)) |
| `ISSUE_IID` with `WORKLINE_TASK=develop` | the issue labelled `workline:ready` (the issue webhook); none: the first ready issue nobody took. The template's jobs `workline:develop:*` run `workline route issue --target issue:<n>` ([developer](../roles/developer/docs/triggers.md)) |
| `MR_IID`, `MR_SOURCE_BRANCH` with `WORKLINE_TASK=review` | the developer's pull request: the reviewer's note on a merge request from `workline/developer/…`, or a person's (the note webhook). The template's jobs `workline:review:*` run `workline develop --merge-request "$MR_IID"` ([the review loop](../roles/developer/docs/review-loop.md)) |
| `WORKLINE_IMPORT` | for `import`: the file moved to issues, as committed on the pipeline's `ref` |

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

## The jobs, without the template

A `.gitlab-ci.yml` and a script that run the same commands, for a project
that does not include the template: [the jobs, without the template](gitlab/without-template.md).

## Tokens and variables

| Variable | Holds | Masked | Protected |
|---|---|---|---|
| `WORKLINE_GITLAB_TOKEN` | a project (or group) access token, Developer, `api` and `write_repository` ([triggers.md](triggers-outputs.md#gitlab-tokens)) | yes | only if every triggered pipeline runs on a protected branch |
| `WORKLINE_GITLAB_READ_TOKEN` | a token with `read_api`, Reporter: gardening, the product owner, the reviewer and the weekly sample read the forge while judging; unset, `CI_JOB_TOKEN`, which reads no issue | yes | the same |
| the agent's credentials | whichever agent you chose; for Claude, `CLAUDE_CODE_OAUTH_TOKEN` (`claude setup-token`, a Claude subscription) or `ANTHROPIC_API_KEY` | yes | the same |
| `WORKLINE_SIGNING_KEY` | where the project refuses unsigned commits: the bot's SSH private key, type File ([signed commits](gitlab/signed-commits.md)) | no: GitLab masks no value of several lines | the same |

A merge request's source branch is rarely protected: a protected variable
never reaches its pipeline. The trigger token acts as the user who owns it.
GitLab gives every variable to every job of the pipeline: the script keeps
the write token out of the judging job's commands, not out of its reach.
The template does the same, and gives its gardening judge the write token.
It also scopes the tokens to the environment `workline`, which the jobs
running your tools never declare ([the environment](gitlab/environment.md)).
Here the judging job runs the gates beside the agent's token; scoped
there, both jobs above declare `environment: {name: workline, action:
prepare}`, or they get no token.

The Code Quality report shows on the merge request when the pipeline is
the merge request's own; a pipeline started on its branch by a tool may not
be shown there (untried). The findings are in each job's summary either
way — `workline-summary.html`, linked from the job's page, and the end of
its log ([ci.md](ci.md#what-a-job-shows)) — and the documentalist's and the
reviewer's comments on the merge request do not depend on it.
