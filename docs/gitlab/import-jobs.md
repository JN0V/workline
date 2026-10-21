<!-- workline
sources: [ci/gitlab/workline.gitlab-ci.yml, cmd/workline]
checked: c2e40104
judged-in-parts: e8a46f7
-->
# Importing a file on GitLab: the two jobs

Beside the template, for a pipeline run by hand with the variable
`WORKLINE_IMPORT` set ([how](../ci-gitlab.md#the-product-owner)). Both
declare the environment the tokens are scoped to
([the environment](environment.md)):

```yaml
workline:import:judge:                     # the agent, a read token
  image: $WORKLINE_IMAGE:$WORKLINE_VERSION
  stage: test
  environment: {name: workline, action: prepare}   # its tokens' scope (environment.md)
  rules: [{if: $WORKLINE_IMPORT}]
  variables: {AGENT: claude}               # one choice: claude, claude:<model>, cmd:<command>
  script:
    - export GITLAB_TOKEN=$WORKLINE_GITLAB_READ_TOKEN
    - npm install -g --silent @anthropic-ai/claude-code   # Claude's CLI; another agent, its own
    - workline issues import "$WORKLINE_IMPORT" --ai "$AGENT" --forge gitlab --summary workline-summary.md --json > line.json
  artifacts: {when: always, paths: [line.json, .workline-runs/, workline-summary.md], expire_in: 1 day}
workline:import:apply:                     # the write token, no agent
  image: $WORKLINE_IMAGE:$WORKLINE_VERSION
  stage: deploy
  environment: {name: workline, action: prepare}
  needs: [workline:import:judge]
  rules: [{if: $WORKLINE_IMPORT, when: always}]
  script:
    - unset CLAUDE_CODE_OAUTH_TOKEN; export GITLAB_TOKEN=$WORKLINE_GITLAB_TOKEN
    - workline apply --line line.json --summary workline-summary.md
  artifacts: {when: always, paths: [workline-summary.md]}
```
