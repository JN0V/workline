---
sources: [ci/gitlab/workline.gitlab-ci.yml, internal/forge/gitlab.go, cmd/workline, Dockerfile]
checked: 492192ca
verified: agent:claude-code
judged: bf016b7
---
# GitLab jobs without the template

Part of [pipelines a tool starts](../gitlab-trigger.md): the jobs, for a
project that does not include the template.

## `.gitlab-ci.yml`

```yaml
stages: [test, deploy]
variables: {GIT_DEPTH: 0, WORKLINE_VERSION: v0.56.0, WORKLINE_RUNS_DIR: $CI_PROJECT_DIR/.workline-runs}  # x-release-please-version
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

## `ci/workline-job.sh`

Any runner with the engine, git and jq; `ssh-keygen` too, to sign
commits (workline's image holds it). The agent is a choice: Claude
here; for another, install it instead and set `ai` to a `cmd:`
([usage](../usage.md)).

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
case "$WORKLINE_TASK" in garden|product-owner|release|develop) git checkout -q -B "$CI_COMMIT_REF_NAME";; esac
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
  judge:develop)
    workline route issue ${ISSUE_IID:+--target issue:$ISSUE_IID} --ai "$ai" --no-apply --forge gitlab --open-merge-request $sum --json > line.json || status=$? ;;
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

Not in this script: `objection` and `review`, on a role's own merge
request. Their judge and apply run as the template's
`workline:objection:*` and `workline:review:*` jobs do
([the commands](../triggers.md#on-a-roles-own-merge-request)).

## What the jobs do

- `--summary` needs the first release after v0.16.0
  ([ci.md](../ci.md#what-a-job-shows)).
- The judging job exits with the verdict: 1 block, 2 for a person, 3 the
  agent or forge unreachable. Make 3 a warning with `allow_failure:
  {exit_codes: [3]}`, as the template does.
- The applying job still runs (`when: always`) and applies what passed,
  and what a role that blocks names to say why, such as its comments
  ([`on-block`](../spec/role-contract.md#one-run)).
- On `garden`, the applying job then tends the merge requests gardening
  opened before: rebuilt on the branch, or closed when nothing is left to
  merge
  ([ADR-0034, amended](../adr/0034-the-release-fix-follows-its-base.md#amendment-2026-10-12-gardenings-merge-requests-follow-too-or-are-closed));
  the first release after v0.24.0.
- On `release`, a judging job that does not pass holds the release: the
  release's gates, never the docs
  ([ADR-0051](../adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md#cut-1-the-release-weekly)).
- The [merge policy](../merge-policy.md), with `merge:` in the target
  branch's `.workline/config.yaml`, workline 0.26 or later:
  - `judge:merge-request` writes the head's verdict (`--verdict`);
  - `apply:merge-request` then runs `workline follow --verdict` from the
    target branch, as the template's `workline:merge` does;
  - `apply:follow` runs the policy too, each time the default branch
    moves;
  - a label starts no pipeline: the merge waits for that merge request's
    next one. Untried from a triggered pipeline.
