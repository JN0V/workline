---
sources: [internal/role/config.go, internal/role/role.go, cmd/workline, internal/engine/engine.go, internal/forge/gitlab.go, internal/hooks]
checked: 59f2c51
verified: agent:claude-code
---
# Configuration

Everything workline reads besides its options: the project's
`.workline/config.yaml`, your own config, the files it keeps, and the
variables. The commands and options: [usage.md](usage.md).

**Which agent answers**, first found wins: `--ai` on the command line;
`WORKLINE_AI` (the git hooks); the project's `ai`; yours; else `none`.
**Which forge**: `--forge`, else the project's `forge`, else `none`.
**A role's settings**: the role's defaults (its `role.yaml`, listed on its
[page](roles.md)), a level the project picks (`autonomy`, the product
owner), then the project's `settings`.

| Key of `.workline/config.yaml` | Holds |
|---|---|
| `ai` | the project's agent: `none`, `claude`, `claude:<model>@<effort>`, `cmd:<command>` |
| `forge` | `github`, `gitlab`, `local`, `cmd:<command>`, `none` (default) |
| `roles.<role>.settings` | the role's settings, merged into its defaults |
| `roles.<role>.enforce` | `<rule>: block \| warn \| off`, per rule |
| `routing.events` | `<event>: [role, gate:<name>, …]`, replacing the shipped line event by event |
| `routing.handoffs`, `routing.max-handoffs` | which role may hand over to which, and how many times ([routing](spec/routing.md)) |
| `gates.<name>` | `criteria`, `checks` (`id`, `run`, `output`, `max`, `optional`, `timeout`), `enforce` ([gates](spec/gates.md)) |
| `repos.<name>` | `url`, `branch`: other repositories docs may name ([multi-repo](spec/multi-repo.md)) |
| `release` | `branches` (the release tool's), `tags` (`v*`) |

## `.workline/config.yaml`

```yaml
ai: claude                  # the project's agent; none by default
forge: github               # github | gitlab | local | cmd:<command> | none (default)
roles:
  committer:
    settings: {subject-max: 60}          # a role's settings, see its role.yaml
    enforce: {internal-code: warn}       # block (default) | warn | off, per rule
  documentalist:
    settings:
      derive: {cases: "ls tests/*.yaml | wc -l"}   # fills <!-- workline:derive cases -->…<!-- workline:end --> in a doc
      history: ["docs/journal/**"]                 # records, beside changelogs and release notes (role.yaml)
      whole-chars: 25000                           # sources judged whole up to this (25000 at most); more docs vouched, more tokens
      versions: {pattern: '\d{2}\.\d+', files: [pyproject.toml]}   # calendar versions, and where the version is said
      language: fr                                 # the docs' language, for the removal rule; unset, read from each doc
      sample: {judge: "claude:opus", at-least: model, after: v1.4.0}   # who reads the weekly sample, the least independence (ADR-0005), and nothing vouched for before your tag
  reviewer:
    settings:
      ai-findings: {correctness: block}   # a verified important finding blocks for these lenses; the others warn (one value, warn or block, sets every lens)
      judge-lines-max: 300                # lines of code a judge reads: the cause's function, then those it reaches, the rest named (200; 0: 31 lines around the cause)
      diff-alone: true                    # one call more a run, given the change only, not the files nor the messages (#126; off by default, measured)
routing:                    # replaces the shipped line, event by event
  events: {merge-request: [committer, documentalist, gate:merge]}
  handoffs: [{from: my-role, to: documentalist}]   # a role of your own (--roles); none shipped hands over
  max-handoffs: 3
gates:                      # docs/spec/gates.md
  merge:
    checks: [{id: secrets, run: "gitleaks detect --report-format sarif --report-path {out}/secrets.sarif", output: sarif, max: {error: 0}}]
repos:                      # other repositories docs may depend on (docs/spec/multi-repo.md)
  api: {url: "https://github.com/acme/api.git", branch: main}
release:                    # how the project's release tool works (ADR-0017)
  branches: ["release/*"]   # its pull requests' branches; default: release-please's, releaser-pleaser's, release-plz's, changesets'
  tags: "v*"                # its tags (the default); the last release is the highest version merged, prereleases left out
```

A key the engine does not know blocks, with its line: an ignored setting is one
someone believes in and nothing applies. So does a role retired, named in
`roles:` or `routing:`: the release manager is gone, and the message names the
release tool to use instead (ADR-0017). A commit written unquoted is a number
to YAML when all digits (`after: 7515148`): a setting naming a commit reads it
as written. One YAML reads otherwise than written — a leading zero
(`0123456`, octal), an `e` between digits (`1234e56`, a float) — blocks,
saying to quote it. A project's `settings` for a role are merged into the
role's at every depth; a list replaces the default whole, a `null` removes
it ([role-adapting.md](spec/role-adapting.md#settings)).

- **`whole-chars` above 25000 is refused**: its task would pass the
  documentalist's context budget, tokens estimated at 711 + 0.82 a
  character (#235), and each doc's prompt would be refused instead.


## Your own config

`~/.config/workline/config.yaml` (macOS: `~/Library/Application
Support/workline/config.yaml`), what you want wherever you work:

| Key | |
|---|---|
| `ai` | your agent when a project does not say (`workline setup` writes it) |
| `approve-push` | `true`: you approve each push (off by default, ADR-0011) |
| `approve-push-via` | where you are asked, in order: `terminal`, `editor`, `dialog` |

Beside it, in the same folder: `allowed-identities` (one address pattern a
line, for the committer), `gitleaks.toml` (your forbidden terms, as gitleaks
rules) and `roles/<role>/<facet>` (your own facets).

## Files

| File | Holds |
|---|---|
| `.workline/config.yaml` | the project's settings, below |
| `.workline/roles/<role>/<facet>` | a facet replacing the shipped one (`policy.md`, `instruction.md`…) |
| `.workline/work/<id>.md` | a work item, without a forge |
| `.git/workline/issues/<n>.md` | `forge: local`: an issue — title, state, labels, milestone in its front matter, then its body and comments — never committed; `workline issues` reads them. With no forge (`none`), an issue a role opens is refused, as every write that needs a forge; in CI (`CI`, `GITHUB_ACTIONS` or `GITLAB_CI` set), the local forge refuses its writes |
| `.git/workline/merge-requests/<n>.md` | `forge: local`: a merge request, its local branch and base named in the front matter; merged once its base holds the branch, closed once the branch is gone |
| `.workline/off` | empty: the global hook skips this repository |
| `~/.config/workline/config.yaml` | yours: `ai:`, your default agent when a project does not say; `approve-push: true` has you approve each push (ADR-0011); `approve-push-via` lists where you are asked, in order (terminal, editor, dialog) |
| `~/.cache/workline/models-seen.yaml` | the last model that answered each alias on this machine: when another one answers, a run reports `model-changed` once, without blocking (ADR-0004) |
| `~/.config/workline/roles/<role>/<facet>` | your own facets, used when the project has none |
| `.git/workline/reviewer-record` | the commits a review on this machine answered whole, not asked again (roles/reviewer) |

Runs are kept in `.git/workline/runs/` (the last
<!-- workline:derive runs-kept -->50<!-- workline:end -->), never in the working tree: each
agent call with what it cost in `out/calls.jsonl` (a judge's too, when
`pre` asked it; `for` names the part or the judge's question it answered), the agent's last answer
as it came in `out/agent-answer.txt`, each refused answer in
`out/refused-<n>.yaml` (as it came: `out/refused-<n>-answer.txt`), and the
claims an accepted answer gave, judged and never applied, in
`out/claims.yaml`.


## Environment

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
| `GITLEAKS_CONFIG`, `GITLEAKS_CONFIG_TOML` | the committer's term list, before the repository's and yours |
| `CI`, `GITHUB_ACTIONS`, `GITLAB_CI` | any set: `forge: local` refuses its writes, the clone being thrown away |

The agent's own credentials are its own: Claude Code reads its login, or
`CLAUDE_CODE_OAUTH_TOKEN` (`claude setup-token`) or `ANTHROPIC_API_KEY`;
workline holds no key.

A role's `pre` and `post` receive `WORKLINE_RUN_DIR`, `WORKLINE_EVENT`,
`WORKLINE_AI`, `WORKLINE_ROLE`, `WORKLINE_BIN` and `WORKLINE_ROLES_DIR`
(the folder of roles), `WORKLINE_FORGE` when a
forge is given, `WORKLINE_TARGET` (`merge-request:12`) with a target too, and
`WORKLINE_OPEN_MERGE_REQUESTS`, `WORKLINE_OPEN_MERGE_REQUEST_TASKS` and
`WORKLINE_PROPOSED_TASKS` with `--open-merge-request`. On a release tool's
merge request, a role that runs on `release` gets `WORKLINE_EVENT=release`
and `WORKLINE_RELEASE_BRANCH`, the branch (ADR-0017). A role run by a handoff
receives the inputs `handoff-from` and `handoff-reason`.
