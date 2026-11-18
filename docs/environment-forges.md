---
sources:
  - internal/forge/gitlab.go#gitlab.connect
  - internal/forge/github.go#github
  - internal/forge/github_merge.go#github.me
  - internal/forge/merge.go#github.mergeToken
  - internal/engine/merge.go#mergePolicy
  - internal/forge/own.go#github.own
  - internal/forge/own.go#gitlab.own
  - internal/forge/gitlab.go#gitlab.editors
  - internal/line/payload.go#PayloadEnv
  - internal/line/payload.go#readPayload
  - internal/line/payload.go#readNote
  - internal/forge/local.go#InCI
checked: e2e165c2d
verified: agent:claude-code
---
# The forge's variables

Part of [the environment](environment.md): the tokens and CI variables
the forges are reached with.

| Variable | |
|---|---|
| `GITLAB_TOKEN` | the GitLab forge's token; else glab's, if glab is set up; else `CI_JOB_TOKEN`, which may read but not write |
| `CI_API_V4_URL`, `CI_PROJECT_ID` (or `CI_PROJECT_PATH`) | the GitLab instance and project, set by GitLab CI; elsewhere `GITLAB_HOST` and the `origin` remote |
| `GH_TOKEN` | read by `gh`, which the GitHub forge calls |
| `WORKLINE_APP`, `WORKLINE_APP_ID` | in GitHub Actions, the GitHub App's slug and id: who keeps the [merge policy](merge-policy.md)'s verdicts and holds its yes. No slug: the job's own token, nothing turned on. The id alone, in a job holding no App token: whose comments keep the records a role reads back ([ADR-0051](adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md)) |
| `WORKLINE_GITLAB_TOKEN` | in GitLab CI, the bot's token: its user, as GitLab says, is workline's own account, whose notes keep the records a role reads back, even in a job reading with another token; it may ask GitLab's GraphQL who last edited them, as a job's own token may not |
| `TRIGGER_PAYLOAD` | in GitLab CI, set by GitLab on a pipeline the issue webhook started: workline reads from it the issue the event names: a state's label put on, the reviewer's comment on a spec in refine, or a person's comment or body edit on an issue on the line; any other event runs nothing ([GitLab alone](gitlab/line-of-states.md)) |
| `CI`, `GITHUB_ACTIONS`, `GITLAB_CI` | any set: `forge: local` refuses its writes, the clone being thrown away |
