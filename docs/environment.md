---
sources: [internal/role/config.go, internal/role/role.go, cmd/workline, internal/engine/engine.go, internal/forge/gitlab.go, internal/hooks]
checked: b6085f38
verified: agent:claude-code
judged: e343388
---
# The environment

The variables workline reads, and those a role's `pre` and `post` receive;
part of [the configuration](config.md).

| Variable | |
|---|---|
| `WORKLINE_AI` | the agent the git hook uses |
| `WORKLINE_ROLES` | a folder of roles, as `--roles` |
| `WORKLINE_MODELS_SEEN` | the file keeping the last model that answered each model asked (default: `models-seen.yaml` in your cache folder, `~/.cache/workline/` on Linux) |
| `WORKLINE_JUDGE` | the agent reading the weekly sample (`workline sample`), before the `sample.judge` setting; the judge of a role's run (a reviewer's findings, a documentalist's split); and the evaluation's judge |
| `WORKLINE_RUNS_DIR` | where runs are kept, e.g. a folder a CI artifact carries to the job that applies |
| `GITLAB_TOKEN` | the GitLab forge's token; else glab's, if glab is set up; else `CI_JOB_TOKEN`, which reaches no issue |
| `CI_API_V4_URL`, `CI_PROJECT_ID` (or `CI_PROJECT_PATH`) | the GitLab instance and project, set by GitLab CI; elsewhere `GITLAB_HOST` and the `origin` remote |
| `GH_TOKEN` | read by `gh`, which the GitHub forge calls |
| `WORKLINE_APP`, `WORKLINE_APP_ID` | in GitHub Actions, the GitHub App's slug and id: who keeps the [merge policy](merge-policy.md)'s verdicts and holds its yes. No slug: the job's own token, nothing turned on. The id alone, in a job holding no App token: whose comments keep the records a role reads back ([ADR-0051](adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md)) |
| `WORKLINE_GITLAB_TOKEN` | in GitLab CI, the bot's token: its user, as GitLab says, is workline's own account, whose notes keep the records a role reads back, even in a job reading with another token; it may ask GitLab's GraphQL who last edited them, as a job's own token may not |
| `GITLEAKS_CONFIG`, `GITLEAKS_CONFIG_TOML` | the committer's term list, before the repository's and yours |
| `CI`, `GITHUB_ACTIONS`, `GITLAB_CI` | any set: `forge: local` refuses its writes, the clone being thrown away |

The agent's own credentials are its own, and workline holds no key:
whichever agent you chose reads its own. Claude Code, for one, reads its
login, or `CLAUDE_CODE_OAUTH_TOKEN` (`claude setup-token`) or
`ANTHROPIC_API_KEY`; an agent run through `cmd:` reads whatever it needs.

A role's `pre` and `post` receive:

- always: `WORKLINE_RUN_DIR`, `WORKLINE_EVENT`, `WORKLINE_AI`,
  `WORKLINE_ROLE`, `WORKLINE_BIN` and `WORKLINE_ROLES_DIR` (the folder of
  roles);
- `WORKLINE_FORGE` when a forge is given, `WORKLINE_TARGET`
  (`merge-request:12`) with a target too;
- `WORKLINE_AI_WAITS=ready` on a draft (`--input draft=true`): the
  agent set aside, `WORKLINE_AI` is `none`
  ([ADR-0050](adr/0050-each-act-has-its-moment-and-a-wait-budget.md#4-what-the-table-holds-to));
- `WORKLINE_OPEN_MERGE_REQUESTS`, `WORKLINE_OPEN_MERGE_REQUEST_TASKS` and
  `WORKLINE_PROPOSED_TASKS` with `--open-merge-request`, and
  `in/waiting.yaml`, the files each waiting task changes, and
  `in/held.yaml`, those each other open merge request changes;
- on a release tool's merge request: `WORKLINE_RELEASE_BRANCH`, the
  branch, to every role ([ADR-0017](adr/0017-the-release-manager.md)); the reviewer
  leaves the versions the tool bumps
  ([#384](https://github.com/JN0V/workline/issues/384));
- run by a handoff: the inputs `handoff-from` and `handoff-reason`.
- in the round a role asks last (`in/then`): `WORKLINE_THEN`, its task.
