---
sources: [ci/gitlab/workline.gitlab-ci.yml, internal/forge/gitlab.go, cmd/workline, Dockerfile]
checked: d477467
verified: agent:claude-code
---
# GitLab: pipelines a tool starts

The GitLab template ([ci.md](ci.md#gitlabcom)) runs on merge request
events and pipeline schedules. Where an internal tool starts the jobs
instead — the trigger API, the pipelines API, another pipeline — this page
does the same with a plain script. The commands themselves:
[triggers.md](triggers.md). **Not tried yet** on a triggered pipeline:
what follows is the template's commands under other rules.

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
`merge_request_event`, so none of the template's jobs run. The tool passes
what to do as variables. The names below are this page's; the engine reads
none of them:

| Variable | Value |
|---|---|
| `WORKLINE_TASK` | `merge-request`, `garden`, `product-owner`, `release`, `sample` |
| `MR_IID`, `MR_SOURCE_BRANCH`, `MR_BASE_SHA` | for `merge-request`: from the merge request (`diff_refs.base_sha`); the pipeline's `ref` is the source branch |

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

`.gitlab-ci.yml`:

```yaml
stages: [test, deploy]
variables: {GIT_DEPTH: 0, WORKLINE_VERSION: v0.13.0, WORKLINE_RUNS_DIR: $CI_PROJECT_DIR/.workline-runs}
.workline:
  image: ghcr.io/jn0v/workline:$WORKLINE_VERSION
  rules: [{if: '$CI_PIPELINE_SOURCE =~ /^(api|trigger|pipeline|parent_pipeline)$/ && $WORKLINE_TASK'}]
workline:judge:
  extends: .workline
  stage: test
  script: [sh ci/workline-job.sh judge]
  artifacts:
    when: always
    paths: [.workline-runs/, line.json, sample.json]
    reports: {codequality: gl-code-quality.json}
workline:apply:
  extends: .workline
  stage: deploy
  needs: [workline:judge]
  when: always
  script: [sh ci/workline-job.sh apply]
```

`ci/workline-job.sh`, any runner with the engine, git and jq:

```sh
#!/bin/sh
set -eu
step=$1; ai=none
if [ "$step" = judge ]; then                      # reads the forge, writes nothing
  [ -z "${WORKLINE_GITLAB_READ_TOKEN:-}" ] || export GITLAB_TOKEN="$WORKLINE_GITLAB_READ_TOKEN"
  if [ -n "${CLAUDE_CODE_OAUTH_TOKEN:-}" ]; then
    npm install -g --silent @anthropic-ai/claude-code && ai=claude
  fi
else                                              # writes, with no AI key
  export GITLAB_TOKEN="$WORKLINE_GITLAB_TOKEN"; unset CLAUDE_CODE_OAUTH_TOKEN ANTHROPIC_API_KEY
  git config user.name workline; git config user.email "workline@noreply.$CI_SERVER_HOST"
  git remote set-url origin "https://oauth2:$WORKLINE_GITLAB_TOKEN@$CI_SERVER_HOST/$CI_PROJECT_PATH.git"
fi
case "$WORKLINE_TASK" in garden|product-owner|release) git checkout -q -B "$CI_COMMIT_REF_NAME";; esac
status=0
case "$step:$WORKLINE_TASK" in
  judge:merge-request)
    git fetch --quiet origin "$MR_BASE_SHA"
    workline route merge-request --ai "$ai" --no-apply --push-to-merge-request --forge gitlab \
      --target "merge-request:$MR_IID" --branch "$MR_SOURCE_BRANCH" \
      --input "range=$MR_BASE_SHA..$CI_COMMIT_SHA" --code-quality gl-code-quality.json --json > line.json || status=$? ;;
  judge:garden)
    workline route schedule --ai "$ai" --no-apply --forge gitlab --open-merge-request --json > line.json || status=$? ;;
  judge:product-owner)
    workline run-role product-owner --event schedule --ai "$ai" --no-apply --forge gitlab --json > po.json || status=$?
    jq '{pending: [."run-dir"]}' po.json > line.json ;;
  judge:release)
    workline route release --ai "$ai" --no-apply --forge gitlab --open-merge-request --json > line.json || status=$? ;;
  judge:sample)
    workline sample --out sample.json || status=$?; [ "$status" != 1 ] && status=0 ;;
  apply:sample) [ ! -s sample.json ] || workline sample --apply sample.json --forge gitlab ;;
  apply:*) [ ! -f line.json ] || workline apply --line line.json || status=$? ;;
esac
exit "$status"
```

The judging job exits with the verdict (1 block, 2 for a person, 3 the
agent or forge unreachable: make 3 a warning with `allow_failure:
{exit_codes: [3]}`); the applying job still runs (`when: always`) and
applies what passed. On `release`, a judging job that does not pass holds
the release; the docs' fix goes to a merge request of its own (untried
outside a release tool's merge request).

## Tokens and variables

| Variable | Holds | Masked | Protected |
|---|---|---|---|
| `WORKLINE_GITLAB_TOKEN` | a project (or group) access token, Developer, `api` and `write_repository` ([triggers.md](triggers.md#what-each-job-needs)) | yes | only if every triggered pipeline runs on a protected branch |
| `WORKLINE_GITLAB_READ_TOKEN` | a token with `read_api`, Reporter: gardening, the product owner and the reviewer read the forge while judging; unset, `CI_JOB_TOKEN`, which reads no issue | yes | the same |
| `CLAUDE_CODE_OAUTH_TOKEN` | `claude setup-token`, a Claude subscription; or set `ANTHROPIC_API_KEY` | yes | the same |

A merge request's source branch is rarely protected: a protected variable
never reaches its pipeline. The trigger token acts as the user who owns it.
GitLab gives every variable to every job of the pipeline: the script keeps
the write token out of the judging job's commands, not out of its reach.
The template does the same, and gives its gardening judge the write token.

The Code Quality report shows on the merge request when the pipeline is
the merge request's own; a pipeline started on its branch by a tool may not
be shown there (untried). The findings are in the job log either way, and
the documentalist's and the reviewer's comments on the merge request do not
depend on it.
