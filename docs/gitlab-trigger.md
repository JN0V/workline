---
sources: [ci/gitlab/workline.gitlab-ci.yml, internal/forge/gitlab.go, cmd/workline, Dockerfile]
checked: 87a81d4a
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
child of a merge request's pipeline, what only warns. The tool passes
what to do as variables. The names below are this page's; the engine reads
none of them:

| Variable | Value |
|---|---|
| `WORKLINE_TASK` | `merge-request`, `garden`, `product-owner`, `release`, `sample`, `follow` (when the default branch moved: the [merge policy](merge-policy.md), when set), `main` (after a merge, `MAIN_BEFORE` its tip before: [main after a merge](main-after-merge.md)), `import` |
| `MR_IID`, `MR_SOURCE_BRANCH`, `MR_TARGET_BRANCH`, `MR_BASE_SHA` | for `merge-request`: from the merge request (`diff_refs.base_sha`); the pipeline's `ref` is the source branch; the tool cancels the one still judging an older head first ([#359](https://github.com/JN0V/workline/issues/359)) |
| `MR_READY`, `MR_DRAFT` | for `merge-request`: `MR_DRAFT`, `true` or `false`, each run, `true` asking no agent; `MR_READY` when it becomes ready (opened not a draft, or an `update` whose `changes.draft` went false), a pipeline the template never gets: every reviewer lens once ([how](../roles/reviewer/docs/ready.md)) |
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

### `.gitlab-ci.yml`

```yaml
stages: [test, deploy]
variables: {GIT_DEPTH: 0, WORKLINE_VERSION: v0.45.0, WORKLINE_RUNS_DIR: $CI_PROJECT_DIR/.workline-runs}  # x-release-please-version
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
    paths: [workline-summary.html, workline-summary.md, .workline-runs/, line.json, sample.json, workline-verdict.json]
    reports: {codequality: gl-code-quality.json, annotations: workline-annotations.json}
workline:apply:
  extends: .workline
  stage: deploy
  needs: [workline:judge]
  when: always
  resource_group: workline-follow   # one at a time: follow and the merge policy push
  script: [sh ci/workline-job.sh apply]
```

### `ci/workline-job.sh`

Any runner with the engine, git and jq; `ssh-keygen` too, to sign
commits (workline's image holds it). The agent is a choice: Claude
here; for another, install it instead and set `ai` to a `cmd:`
([usage](usage.md)).

```sh
#!/bin/sh
set -eu
step=$1; ai=none; sum="--summary workline-summary.md --summary workline-summary.html"   # what the job's page shows
if [ "$step" = judge ]; then                      # reads the forge, writes nothing
  [ -z "${WORKLINE_GITLAB_READ_TOKEN:-}" ] || export GITLAB_TOKEN="$WORKLINE_GITLAB_READ_TOKEN"
  if [ -n "${CLAUDE_CODE_OAUTH_TOKEN:-}" ]; then   # the agent: Claude here, one choice among others
    if [ "${MR_DRAFT:-}" = true ]; then ai=claude  # a draft asks none: nothing to install
    else npm install -g --silent @anthropic-ai/claude-code && ai=claude; fi
  fi
else                                              # writes, with no AI key
  export GITLAB_TOKEN="$WORKLINE_GITLAB_TOKEN"; unset CLAUDE_CODE_OAUTH_TOKEN ANTHROPIC_API_KEY
  git config user.name workline; git config user.email "workline@noreply.$CI_SERVER_HOST"
  if [ -n "${WORKLINE_SIGNING_KEY:-}" ]; then     # signed commits, an SSH key as a File variable
    [ -f "$WORKLINE_SIGNING_KEY" ] || { echo "workline: WORKLINE_SIGNING_KEY is not a file: set its type to File"; exit 1; }   # the key never printed
    chmod 600 "$WORKLINE_SIGNING_KEY"; [ -z "$(tail -c 1 "$WORKLINE_SIGNING_KEY")" ] || echo >> "$WORKLINE_SIGNING_KEY"
    git config gpg.format ssh; git config user.signingkey "$WORKLINE_SIGNING_KEY"; git config commit.gpgsign true
  fi
  git remote set-url origin "https://oauth2:$WORKLINE_GITLAB_TOKEN@$CI_SERVER_HOST/$CI_PROJECT_PATH.git"
fi
case "$WORKLINE_TASK" in garden|product-owner|release) git checkout -q -B "$CI_COMMIT_REF_NAME";; esac
status=0
case "$step:$WORKLINE_TASK" in
  judge:merge-request)
    git fetch --quiet origin "$MR_BASE_SHA"
    workline route merge-request --ai "$ai" --no-apply --forge gitlab \
      --target "merge-request:$MR_IID" --branch "$MR_SOURCE_BRANCH" \
      --input "range=$MR_BASE_SHA..$CI_COMMIT_SHA" ${MR_READY:+--input lenses=all} ${MR_DRAFT:+--input draft=$MR_DRAFT} \
      --code-quality gl-code-quality.json --verdict workline-verdict.json $sum --json > line.json || status=$? ;;
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
    workline sample --forge gitlab --out sample.json $sum || status=$?; [ "$status" != 1 ] && status=0 ;;
  apply:sample) [ ! -s sample.json ] || workline sample --apply sample.json --forge gitlab $sum ;;
  judge:follow) ;;                                # no agent: all in the applying job
  apply:follow) workline follow --base "$CI_DEFAULT_BRANCH" --forge gitlab $sum || status=$? ;;
  judge:main) workline route main --ai none --forge none --input "range=${MAIN_BEFORE:-}..$CI_COMMIT_SHA" $sum --json > line.json || status=$? ;;
  apply:main) [ ! -s line.json ] || workline apply --main line.json --forge gitlab $sum || status=$? ;;
  apply:merge-request)                            # then the merge policy, by the target branch's config
    [ ! -f line.json ] || workline apply --line line.json $sum || status=$?
    git fetch -q origin "$MR_TARGET_BRANCH"; git checkout -q -B "$MR_TARGET_BRANCH" FETCH_HEAD
    v=; [ ! -s workline-verdict.json ] || v="--verdict workline-verdict.json"
    if grep -qE '^merge:' .workline/config.yaml 2>/dev/null; then
      workline follow --base "$MR_TARGET_BRANCH" --forge gitlab $v $sum || status=$?
    fi ;;
  apply:garden)                                   # then its merge requests the branch moved under
    workline apply --line line.json $sum || status=$?
    workline follow --gardening --base "$CI_COMMIT_REF_NAME" --forge gitlab $sum || status=$? ;;
  apply:*) [ ! -f line.json ] || workline apply --line line.json $sum || status=$? ;;
esac
exit "$status"
```

### What the jobs do

- `--summary` needs the first release after v0.16.0
  ([ci.md](ci.md#what-a-job-shows)).
- The judging job exits with the verdict: 1 block, 2 for a person, 3 the
  agent or forge unreachable. Make 3 a warning with `allow_failure:
  {exit_codes: [3]}`, as the template does.
- The applying job still runs (`when: always`) and applies what passed,
  and what a role that blocks names to say why, such as its comments
  ([`on-block`](spec/role-contract.md#one-run)).
- On `garden`, the applying job then tends the merge requests gardening
  opened before: rebuilt on the branch, or closed when nothing is left to
  merge
  ([ADR-0034, amended](adr/0034-the-release-fix-follows-its-base.md#amendment-2026-10-12-gardenings-merge-requests-follow-too-or-are-closed));
  the first release after v0.24.0.
- On `release`, a judging job that does not pass holds the release: the
  release's gates, never the docs
  ([ADR-0051](adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md#cut-1-the-release-weekly)).
- The [merge policy](merge-policy.md), with `merge:` in the target
  branch's `.workline/config.yaml`, workline 0.26 or later:
  - `judge:merge-request` writes the head's verdict (`--verdict`);
  - `apply:merge-request` then runs `workline follow --verdict` from the
    target branch, as the template's `workline:merge` does;
  - `apply:follow` runs the policy too, each time the default branch
    moves;
  - a label starts no pipeline: the merge waits for that merge request's
    next one. Untried from a triggered pipeline.

## Tokens and variables

| Variable | Holds | Masked | Protected |
|---|---|---|---|
| `WORKLINE_GITLAB_TOKEN` | a project (or group) access token, Developer, `api` and `write_repository` ([triggers.md](triggers.md#gitlab-tokens)) | yes | only if every triggered pipeline runs on a protected branch |
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
