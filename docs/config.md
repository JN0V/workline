---
sources:
  - internal/role/config.go#config
  - internal/role/config.go#CheckConfig
  - internal/role/config.go#Retired
  - internal/role/config.go#asWritten
  - internal/role/role.go#ProjectConfig
  - internal/role/role.go#Board
  - internal/role/role.go#LoadProjectConfig
  - internal/role/role.go#Role.MergedSettings
  - internal/role/role.go#Role.Enforcement
  - internal/kit/verdict/verdict.go#Levels
  - internal/role/personal.go#Role.WithPersonal
  - internal/engine/engine.go#run
  - internal/engine/engine.go#callAgent
  - internal/engine/engine.go#personalSettings
  - internal/engine/engine.go#keepRuns
  - internal/engine/engine.go#newRunDir
  - internal/engine/tally.go#TallyFile
  - cmd/workline/main.go#personal
  - cmd/workline/main.go#hookCmd
  - internal/hooks/approve.go#Push.channels
  - internal/agent/prompt.go#facetDirs
  - internal/agent/seen.go#Seen
  - internal/forge/forge.go#Open
  - internal/forge/local.go#Local
  - internal/forge/local.go#Local.StateOf
  - internal/forge/local.go#InCI
  - internal/forge/local.go#Local.writable
  - internal/work/work.go#Path
  - internal/release/release.go#Branches
  - internal/release/release.go#Tags
  - internal/routing/routing.go#Load
  - internal/routing/routing.go#Config.StopsAtFirst
  - defaults/routing.yaml#fail-fast
  - defaults/routing.yaml#max-handoffs
  - defaults/routing.yaml#wip
  - defaults/routing.yaml#stale
  - defaults/routing.yaml#revisit
  - defaults/routing.yaml#budgets
  - internal/stall/stall.go#DefaultHours
  - internal/routing/timed.go#checkTimed
  - roles/committer/role.yaml#settings.subject-max
  - roles/committer/role.yaml#settings.rewrite
  - roles/committer/role.yaml#personal
  - roles/documentalist/role.yaml#settings.derive
  - roles/documentalist/role.yaml#settings.history
  - roles/documentalist/role.yaml#settings.versions
  - roles/documentalist/role.yaml#settings.language
  - roles/documentalist/role.yaml#settings.whole-chars
  - roles/reviewer/role.yaml#settings.ai-findings
  - roles/reviewer/role.yaml#settings.judge-lines-max
  - roles/reviewer/role.yaml#settings.judge-read-rounds
  - roles/reviewer/role.yaml#settings.diff-alone
  - roles/reviewer/role.yaml#settings.spec-rounds
  - roles/reviewer/role.yaml#settings.specs-per-run
checked: e81df7b34
verified: agent:claude-code
reader: ce73d8d5f995
---
# Configuration

Everything workline reads besides its options: the project's
`.workline/config.yaml`, your own config, the files it keeps, and the
variables. The commands and options: [usage.md](usage.md).

## Which agent, which forge

**Which agent answers**, first found wins: `--ai` on the command line;
`WORKLINE_AI` (the git hooks); the project's `ai`; yours; else `none`.
**Which forge**: `--forge`, else the project's `forge`, else `none`.
**A role's settings**: the role's defaults (its `role.yaml`, listed on its
[page](roles.md)), the listed [packs](packs.md), its
[autonomy](autonomy.md) level, then the project's `settings`.

## Keys: agent, roles, gates, merge

The keys of `.workline/config.yaml`, in two tables.

| Key of `.workline/config.yaml` | Holds |
|---|---|
| `autonomy` | each role's level unless it sets its own; not the merge's ([autonomy](autonomy.md)) |
| `ai` | the project's agent: `none`, `claude`, `claude:<model>@<effort>`, `cmd:<command>` |
| `forge` | `github`, `gitlab`, `local`, `cmd:<command>`, `none` (default) |
| `roles.<role>.settings` | the role's settings, merged into its defaults |
| `roles.<role>.enforce` | `<rule>: block \| warn \| off`, per rule |
| `routing.events` | `<event>: [role, gate:<name>, …]`, replacing the shipped line event by event |
| `routing.fail-fast` | `<event>: false` runs every step of the event, the worst verdict standing, a refused proposal said, not failed; true unless set, but on `schedule` and `main` ([ADR-0037](adr/0037-the-schedule-runs-every-step.md)) |
| `routing.handoffs`, `routing.max-handoffs` | which role may hand over to which, and how many times ([routing](spec/routing.md)) |
| `routing.budgets` | `<moment>: <seconds>`, each moment's wait budget; over it, the summary says so, never a block ([routing](spec/routing.md#each-steps-seconds-a-moments-budget)) |
| `gates.<name>` | `criteria`, `checks` (`id`, `run`, `output`, `max`, `optional`, `timeout`, `fix`; on a merge request `new`, `base-timeout`, `config`), `enforce` ([gates](spec/gates.md)) |
| `repos.<name>` | `url`, `branch`: other repositories docs may name ([multi-repo](spec/multi-repo.md)) |
| `release` | `branches` (the release tool's), `tags` (`v*`) |
| `merge` | `autonomy: cautious`: the forge merges what a person accepted; `normal`: workline's own routine work too, a reproduced bug's fix among it, with no yes; `enterprising`: the developer's features and tasks too, a person's yes on the milestone's [release candidate](release-candidate.md); `alone: [reader]`: the reader's rewordings too; `alone: [task]`, at `normal`: a task that changes no behaviour too; `shadow: [task]`, at any level: its judge asked and said, nothing merged on it ([ADR-0073](adr/0073-a-task-that-changes-no-behaviour-may-merge-alone.md)) ([merge policy](merge-policy.md)); off unless set |

## Keys: routing, boards, packs

| Key of `.workline/config.yaml` | Holds |
|---|---|
| `routing.states`, `routing.max-hops`, `routing.reconcile-max` | the line of states, its hop cap, the reconciler's steps a run ([routing](spec/routing.md#the-line-of-states)) |
| `routing.wip` | `<state>: <issues>`, a state's WIP limit, laid over the shipped `ready: 5`; `0` lifts it ([room in the next state](spec/routing.md#room-in-the-next-state)) |
| `routing.revisit` | the days a blocked issue waits, nothing new on it, before the product owner reads it again; `7` shipped, `0` never ([blocked, read again](spec/routing.md#blocked-read-again)) |
| `routing.stalled-hours` | the hours a developer's pull request keeps its head, its verdict not a pass, before the wake and the reconcile say it stalled (`12`; `0`, never; [the reconciler](spec/routing.md#the-reconciler)) |
| `routing.stale` | `<state>: <days>`, a state's age limit, laid over the shipped `ready: 30`, `tests: 14`; `0` lifts it ([staleness](spec/routing.md#staleness)) |
| `routing.timed` | `<job>: {at: <cron>}`, `{gap: <minutes>}` or `off`: when [`workline tick`](tick.md#the-times) lists a timed job; at or gap, never both |
| `packs` | the [language packs](packs.md) every role reads, as `[go, kotlin-gradle]`; written by `workline init` |
| `board` | `ready`, `doing`, `refine`, `tests`, `blocked`: a column's name, over the names workline matches; `project`: a GitHub project's number, when several are linked ([boards](boards.md)) |
| `gitlab.bots` | usernames GitLab does not name bots that are: the service account or line account holding workline's token; its comments, labels and reactions are a bot's, never a person's ([why](ci-gitlab.md#what-the-template-does)) |

## `.workline/config.yaml`

### An example

A commented example of every key, and what its comments point to: [a config example](config-example.md).

### What blocks

- **A key the engine does not know**, with its line: an ignored setting is
  one someone believes in and nothing applies.
- **A level the top-level `autonomy` does not know**, the levels named.
- **A level `merge:` does not know**, the levels named; in `alone`, a
  kind other than `reader` or `task`, or `alone` at `cautious`; `task` in `alone` at `enterprising`; in `shadow`, a kind other than `task`.
- **A role retired**, named in `roles:` or `routing:`: the release manager
  is gone, and the message names the release tool to use instead
  ([ADR-0017](adr/0017-the-release-manager.md)).
- **A value YAML reads otherwise than written** — a leading zero
  (`0123456`, octal), an `e` between digits (`1234e56`, a float) — saying
  to quote it. A commit written unquoted is a number to YAML when all
  digits (`after: 7515148`): a setting naming a commit reads it as written.
- **`whole-chars` above 25000** is refused: a doc's task would no longer
  fit the documentalist's context
  ([#235](https://github.com/JN0V/workline/issues/235)).

### How settings merge

A project's `settings` for a role are merged into the role's at every
depth; a list replaces the default whole, a `null` removes it
([role-adapting.md](spec/role-adapting.md#settings)).

## Your own config

`~/.config/workline/config.yaml` (macOS: `~/Library/Application
Support/workline/config.yaml`), what you want wherever you work:

| Key | |
|---|---|
| `ai` | your agent when a project does not say (`workline setup` writes it) |
| `approve-push` | `true`: you approve each push (off by default, [ADR-0011](adr/0011-the-review-is-on-the-merge-request-not-the-push.md)) |
| `approve-push-via` | where you are asked, in order: `terminal`, `editor`, `dialog` |
| `roles.<role>.settings` | what a role lets you set for yourself (`personal` in its role.yaml; the committer's `rewrite`, `ai-max-seconds`, `internal-codes-allow`): a value wins over the project's, a list adds to it; a project's `ai: none` still keeps every agent away; any other setting, or one of another kind than the role's, is said and left out |

Beside it, in the same folder: `allowed-identities` (one address pattern a
line, for the committer), `gitleaks.toml` (your forbidden terms, as gitleaks
rules) and `roles/<role>/<facet>` (your own facets; on macOS,
`~/.config/workline/roles/` is read too, after it).

## Files

Where workline keeps its files, and what a local issue holds: [the files](config-files.md). Runs are kept as [below](#runs).

### Runs

Runs are kept in `.git/workline/runs/` (the last
<!-- workline:derive runs-kept -->50<!-- workline:end -->), never in the working tree. Each holds:

- each agent call with what it cost, in `out/calls.jsonl` (a judge's too,
  when `pre` asked it; `for` names the part or the judge's question it
  answered);
- the agent's last answer as it came, in `out/agent-answer.txt`;
- each refused answer in `out/refused-<n>.yaml` (as it came:
  `out/refused-<n>-answer.txt`);
- the claims an accepted answer gave, judged and never applied, in
  `out/claims.yaml`.

An older run's folder is dropped, what its agent calls spent kept first:
one line a run in `tally.jsonl`, beside them, never pruned. The weekly
sample sums a week's tokens from both ([its tokens](weekly-sample.md#what-it-writes)).
In CI, a job's runs folder (`WORKLINE_RUNS_DIR`) goes with the job, its
artifacts short-lived; `workline sample --tally <file>` writes its runs'
lines for an artifact kept 14 days.


## Environment

The variables workline reads, the agent's credentials, and what a role's
`pre` and `post` receive: [the environment](environment.md).
