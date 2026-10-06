---
sources: [ci/gitlab/workline.gitlab-ci.yml, internal/forge/gitlab.go, cmd/workline, Dockerfile]
checked: 0efb09c
judged: 02da247
verified: agent:claude-code
---
# GitLab: pipelines a tool starts

The GitLab template ([ci.md](ci.md#gitlabcom)) runs on merge request
events and pipeline schedules. Where an internal tool starts the jobs
instead — the trigger API, the pipelines API, another pipeline — this page
does the same with a plain script. The commands themselves:
[triggers.md](triggers.md). Tried on a pipeline the API started
(2026-10-06): the judging job of `garden`, without an agent, and its
summary; the other tasks and the applying job, not yet. What follows is
the template's commands under other rules.

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
| `WORKLINE_TASK` | `merge-request`, `garden`, `product-owner`, `release`, `sample`, `follow` (when the default branch moved: the release fix rebuilt on it, ADR-0034), `import` |
| `MR_IID`, `MR_SOURCE_BRANCH`, `MR_BASE_SHA` | for `merge-request`: from the merge request (`diff_refs.base_sha`); the pipeline's `ref` is the source branch |
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

`.gitlab-ci.yml`:

```yaml
stages: [test, deploy]
variables: {GIT_DEPTH: 0, WORKLINE_VERSION: v0.17.0, WORKLINE_RUNS_DIR: $CI_PROJECT_DIR/.workline-runs}
.workline:
  image: ghcr.io/jn0v/workline:$WORKLINE_VERSION
  rules: [{if: '$CI_PIPELINE_SOURCE =~ /^(api|trigger|pipeline|parent_pipeline)$/ && $WORKLINE_TASK'}]
  after_script:                     # the summary: at the end of the log, linked from the job's page
    - '[ ! -s workline-summary.md ] || cat workline-summary.md'
    - printf '{"workline":[{"external_link":{"label":"workline summary","url":"%s/artifacts/file/workline-summary.html"}}]}\n' "$CI_JOB_URL" > workline-annotations.json
  artifacts:
    when: always
    paths: [workline-summary.html, workline-summary.md]
    reports: {annotations: workline-annotations.json}
workline:judge:
  extends: .workline
  stage: test
  script: [sh ci/workline-job.sh judge]
  artifacts:
    paths: [workline-summary.html, workline-summary.md, .workline-runs/, line.json, sample.json]
    reports: {codequality: gl-code-quality.json, annotations: workline-annotations.json}
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
step=$1; ai=none; sum="--summary workline-summary.md --summary workline-summary.html"   # what the job's page shows
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
      --input "range=$MR_BASE_SHA..$CI_COMMIT_SHA" --code-quality gl-code-quality.json $sum --json > line.json || status=$? ;;
  judge:garden)
    workline route schedule --ai "$ai" --no-apply --forge gitlab --open-merge-request $sum --json > line.json || status=$? ;;
  judge:product-owner)
    workline run-role product-owner --event schedule --ai "$ai" --no-apply --forge gitlab $sum --json > po.json || status=$?
    jq '{pending: [."run-dir"]}' po.json > line.json ;;
  judge:release)
    workline route release --ai "$ai" --no-apply --forge gitlab --open-merge-request $sum --json > line.json || status=$? ;;
  judge:import)                                   # the issues it would open; apply:* opens them
    workline issues import "$WORKLINE_IMPORT" --ai "$ai" --forge gitlab $sum --json > line.json || status=$? ;;
  judge:sample)
    workline sample --out sample.json $sum || status=$?; [ "$status" != 1 ] && status=0 ;;
  apply:sample) [ ! -s sample.json ] || workline sample --apply sample.json --forge gitlab $sum ;;
  judge:follow) ;;                                # no agent: all in the applying job
  apply:follow) workline follow --base "$CI_DEFAULT_BRANCH" --forge gitlab || status=$? ;;
  apply:*) [ ! -f line.json ] || workline apply --line line.json $sum || status=$? ;;
esac
exit "$status"
```

`--summary` needs the first release after v0.16.0
([ci.md](ci.md#what-a-job-shows)). The judging job exits with the verdict (1 block, 2 for a person, 3 the
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
be shown there (untried). The findings are in each job's summary either
way — `workline-summary.html`, linked from the job's page, and the end of
its log ([ci.md](ci.md#what-a-job-shows)) — and the documentalist's and the
reviewer's comments on the merge request do not depend on it.
