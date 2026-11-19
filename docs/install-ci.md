---
sources:
  - defaults/ci/github/workline.yml#on
  - defaults/ci/github/workline.yml#env
  - defaults/ci/github/workline-apply.yml#on
  - defaults/ci/github/workline-warn.yml#on
  - defaults/ci/github/workline-fork.yml#on
  - defaults/ci/github/workline-gardening.yml#jobs.judge
  - defaults/ci/github/workline-sample.yml#jobs.read
  - defaults/ci/github/workline-sample.yml#jobs.write
  - defaults/ci/gitlab/workline.gitlab-ci.yml#.workline
  - defaults/ci/gitlab/workline.gitlab-ci.yml#.workline:environment
  - defaults/ci/gitlab/workline.gitlab-ci.yml#.workline:bot-token
  - defaults/ci/gitlab/workline.gitlab-ci.yml#.workline:claude
  - defaults/ci/gitlab/workline.gitlab-ci.yml#.workline:sample
  - internal/doctor/workflows.go#ciTemplates
  - internal/builtin/documentalist/documentalist.go#preRun.sayShallow
checked: e2e165c2d
verified: agent:claude-code
---
# Install in CI

Part of [install](install.md).

What the jobs are: [ci.md](ci.md); every detail on your forge's page. In short:

- **GitHub**: copy the workflows of [defaults/ci/github](../defaults/ci/github/) into
  `.github/workflows/` — [workline.yml](../defaults/ci/github/workline.yml) (each
  pull request, and main after each merge), [workline-warn.yml](../defaults/ci/github/workline-warn.yml) (what
  only warns, beside it), [workline-apply.yml](../defaults/ci/github/workline-apply.yml)
  (what they proposed applied, and `workline follow`;
  [each job](ci-github.md#set-up)), [workline-fork.yml](../defaults/ci/github/workline-fork.yml) (a
  fork's), [workline-gardening.yml](../defaults/ci/github/workline-gardening.yml),
  [workline-sample.yml](../defaults/ci/github/workline-sample.yml); the others come
  with what uses them: the tester, the developer and the line of states
  (its reconciler's [heartbeat](github/heartbeat.md)), the
  [merge policy](merge-policy.md), a release tool
  ([ci-github.md](ci-github.md)). **Before December 2, 2026, on a
  public repository**: an Actions event policy on the bounds and merge
  workflows allowing `pull_request_target`, `issues` and `schedule`, or no
  pull request merges
  ([what to set](github/pull-request-target.md)).
- **GitLab**: include
  [defaults/ci/gitlab/workline.gitlab-ci.yml](../defaults/ci/gitlab/workline.gitlab-ci.yml),
  then add pipeline schedules, gardening, the sample and, with the line
  of states, the hourly reconciler and the issue webhook
  ([ci-gitlab.md](ci-gitlab.md#gitlabcom); a [self-managed
  instance](ci-gitlab-self-managed.md)).
- **Another forge** (Gitea, Forgejo…) or [none](no-forge.md): [ci-other-forges.md](ci-other-forges.md).
- **A board**, GitHub's project or GitLab's: what to switch on so the
  roles move its cards, [boards](boards.md#what-you-switch-on).

What the templates need:

| | GitHub | GitLab |
|---|---|---|
| the agent, your choice | its credentials; for Claude, which the templates install: secret `CLAUDE_CODE_OAUTH_TOKEN` (`claude setup-token`) | the same as a variable, masked, not protected, scoped to the environment `workline` ([why](gitlab/environment.md)) |
| writing | a [GitHub App](ci-github-app.md): variable `WORKLINE_APP_ID`, secret `WORKLINE_APP_PRIVATE_KEY` in the environment `workline-app` | variable `WORKLINE_GITLAB_TOKEN`: a project access token, `api` and `write_repository`, scoped the same |
| the history | `fetch-depth: 0`, set | `GIT_DEPTH: 0`, set |
| the sample's judge | variable `WORKLINE_JUDGE` (optional) | the same, and `WORKLINE_TASK=sample` on its schedule |

- Without the agent's token, the templates run with no agent: every check
  still runs. For another agent, change their `--ai` to a `cmd:`.
- The whole history is needed: docs are compared since their `checked`,
  the release since its tag. A job cloning too shallow reports
  `shallow-clone`, never a pass.
- The templates run the release named by `WORKLINE_VERSION` at their top:
  update it for a newer one; `workline doctor` names any pinned older.
- A pipeline started by a tool rather than by the forge (GitLab's trigger
  API, another pipeline): [gitlab-trigger.md](gitlab-trigger.md). The bare
  command for each event, for any other trigger: [triggers.md](triggers.md).
