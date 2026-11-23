---
sources:
  - docs/gitlab/without-template.md#gitlab-ciyml
  - docs/gitlab/without-template.md#ciworkline-jobsh
  - defaults/ci/gitlab/workline.gitlab-ci.yml#variables
  - defaults/ci/gitlab/workline.gitlab-ci.yml#.workline:environment
  - defaults/ci/gitlab/workline.gitlab-ci.yml#.workline:bot
  - defaults/ci/gitlab/workline.gitlab-ci.yml#.workline:garden
  - defaults/ci/gitlab/workline.gitlab-ci.yml#.workline:summary
  - internal/forge/gitlab.go#gitlab.connect
checked: 3bab32400
verified: agent:claude-code
reader: 0f3e75c013d8
---
# Tokens and variables of a started pipeline

Part of [GitLab: pipelines a tool starts](../gitlab-trigger.md): what
the jobs a tool starts hold, and where.

| Variable | Holds | Masked | Protected |
|---|---|---|---|
| `WORKLINE_GITLAB_TOKEN` | a project (or group) access token, Developer, `api` and `write_repository` ([triggers.md](../triggers-outputs.md#gitlab-tokens)) | yes | only if every triggered pipeline runs on a protected branch |
| `WORKLINE_GITLAB_READ_TOKEN` | a token with `read_api`, Reporter: gardening, the product owner, the reviewer and the weekly sample read the forge while judging; unset, `CI_JOB_TOKEN`, which reads no issue | yes | the same |
| the agent's credentials | whichever agent you chose; for Claude, `CLAUDE_CODE_OAUTH_TOKEN` (`claude setup-token`, a Claude subscription) or `ANTHROPIC_API_KEY` | yes | the same |
| `WORKLINE_SIGNING_KEY` | where the project refuses unsigned commits: the bot's SSH private key, type File ([signed commits](signed-commits.md)) | no: GitLab masks no value of several lines | the same |

- A merge request's source branch is rarely protected: a protected variable never reaches its pipeline.
- The trigger token acts as the user who owns it.
- GitLab gives every variable to every job of the pipeline: the script keeps the write token out of the proposing job's commands, not out of its reach.
- The template does the same, and gives its gardening propose job the write token.
- It also scopes the tokens to the environment `workline`, which the jobs running your tools never declare ([the environment](environment.md)).
- Here the proposing job runs the gates beside the agent's token. Scoped there, the tokens reach only jobs that declare the environment: add `environment: {name: workline, action: prepare}` to both jobs of [the script](without-template.md), or they get no token.

The Code Quality report shows on the merge request when the pipeline is
the merge request's own; a pipeline started on its branch by a tool may not
be shown there (untried). The findings are in each job's summary either
way — `workline-summary.html`, linked from the job's page, and the end of
its log ([ci.md](../ci.md#what-a-job-shows)) — and the documentalist's and the
reviewer's comments on the merge request do not depend on it.
