---
sources:
  - cmd/workline/main.go#hookCmd
  - internal/agent/seen.go#Seen
  - internal/sample/sample.go#Options
  - internal/engine/engine.go#askJudge
  - internal/builtin/reviewer/reviewer.go#askJudges
  - tests/evaluation/eval_test.go#judge
  - internal/forge/retry.go#RunsDirEnv
  - roles/committer/code/secrets.go#commonList
  - internal/forge/mask.go
  - roles/tester/code/tester.go#shell
  - internal/gate/gate.go#maskLog
  - internal/agent/claude.go#lastLine
  - internal/builtin/documentalist/examples.go#outputSaid
checked: dd77bf922
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
| `WORKLINE_JUDGE` | the agent reading the weekly sample (`workline sample`), before the `sample.judge` setting; the judge of a role's run (a reviewer's findings, [a documentalist's split](../roles/documentalist/docs/gardening-condense.md#splitting-a-card)); and the evaluation's judge |
| `WORKLINE_RUNS_DIR` | where runs are kept, e.g. a folder a CI artifact carries to the job that applies; never seen by the project's commands ([the line's own](environment-roles.md#the-lines-own)) |
| `GITLEAKS_CONFIG`, `GITLEAKS_CONFIG_TOML` | the committer's term list, before the repository's and yours |

- **The forge's and CI's variables**: tokens, the GitHub App, GitLab's
  trigger ([the forge's variables](environment-forges.md)).
- **A role's `pre` and `post`**: what they receive, and what the
  project's commands never see ([a role's variables](environment-roles.md)).

The agent's own credentials are its own, and workline holds no key:
whichever agent you chose reads its own. Claude Code, for one, reads its
login, or `CLAUDE_CODE_OAUTH_TOKEN` (`claude setup-token`) or
`ANTHROPIC_API_KEY`; an agent run through `cmd:` reads whatever it needs.

## Secrets in quoted output

A command's output a comment quotes shows `***` in place of a secret
([#1215](https://github.com/JN0V/workline/issues/1215)): the tests' lines
in a red proof, a check's log, an agent's failure, a docs example's output.

- **Which**: the values of the forge's tokens, and of each
  variable whose name ends in `_TOKEN`, `_KEY`, `_SECRET`, `_PASSWORD`,
  `_PAT` or `_CREDENTIALS`, case aside.
- **Kept**: a value under 8 characters, and every variable's name.
- **Only the exact value**: a form the command changed — URL-encoded,
  JSON-escaped, in another case — is not masked.
- **When**: before a line is cut to fit a comment, so no cut leaves a
  part of one.
