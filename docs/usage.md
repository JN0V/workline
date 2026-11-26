---
sources:
  - cmd/workline/cli.go#init
  - cmd/workline/cli.go#dispatch
  - cmd/workline/cli.go#topHelp
  - cmd/workline/cli.go#parseFlags
  - cmd/workline/cli.go#misuse
  - cmd/workline/main.go#runRole
  - cmd/workline/main.go#exitFor
  - internal/agent/agent.go#ErrUnavailable
  - internal/agent/claude.go#claude
  - internal/agent/claude.go#claudeTier
  - internal/agent/claude.go#claudeEffort
  - internal/agent/claude.go#claudeSession
  - internal/agent/claude.go#claudeEnv
  - internal/agent/claude.go#withoutTokens
  - internal/forge/mask.go#Tokens
  - internal/agent/command.go#command.Propose
  - internal/forge/forge.go#Open
  - internal/sample/sample.go#Options
checked: e81df7b34
verified: agent:claude-code
---
# Using workline

The commands, their options and exit codes.
What a role does is on its page ([roles.md](roles.md)); the files, the
settings and the variables, in [config.md](config.md).

## Help

- `workline --help`: every command, one line each (`hook`, `builtin`:
  internal); `workline <command> --help`, or `workline help <command>`:
  its usage and options; exit 0.
- An unknown command is named, with a pointer to `workline --help`:
  exit 64.
- A bad option or value is named, then the command's usage, on
  stderr: exit 64.

## Commands

Each in detail: [the commands](commands.md),
[the commands that do the work](commands-work.md), and
[the commands that set up](commands-setup.md).

### Run the line and its roles

| Command | Does |
|---|---|
| [`workline run-role <role> --event <event>`](commands.md#workline-run-role) | runs one role on an event: prepare, propose, judge, apply |
| [`workline route <event>`](commands.md#workline-route) | runs the steps routing names for the event, in order |
| [`workline apply <run-dir>...`, `--line <file>`](commands.md#workline-apply) | applies runs judged with `--no-apply`, or resumes a run stopped while applying |
| [`workline review`](commands-work.md#workline-review) | the reviewer reads this branch before you push, or a spec before it is built |
| [`workline gate <name>`](commands.md#workline-gate) | runs a gate declared in `.workline/config.yaml` |
| [`workline test check --base <ref>`](commands-work.md#workline-test) | the tester's check: each test the change adds fails on the base, read from the test runner's JUnit report |
| [`workline test write --issue <n>`](commands-work.md#workline-test) | the tester writes a ready issue's tests, opened as a Draft |
| [`workline test answer --merge-request <n>`](commands-work.md#workline-test) | the tester answers an objection to one of its tests: kept, or rewritten |
| [`workline test measure`, `test curate`](commands-work.md#workline-test) | experimental: the tester's curator measures the test corpus; curate weighs it against its history |
| [`workline develop`](commands-work.md#workline-develop) | the developer takes the first ready issue nobody took and claims it; `--merge-request <n>`: a review round on its pull request; `--bounds`: checks its files against its issue's bounds |

### The backlog, the docs and main

| Command | Does |
|---|---|
| [`workline item ready <id>`](commands-work.md#workline-item-ready) | moves a work item to `ready`, once its sections are written |
| [`workline issues`](commands-work.md#workline-issues) | reads and writes the issues kept in the clone (`forge: local`): `list`, `show`, `open`, `label`, `comment` |
| [`workline issues import <file>`](commands-work.md#workline-issues-import) | turns a roadmap or backlog file into the forge's issues |
| [`workline docs`](commands.md#workline-docs) | judges the docs made suspect since they were last judged; you review each change |
| [`workline sample`](commands.md#workline-sample) | the week's sample of every role's acts, one in ten per role and kind, the docs the documentalist passed among them, each read again; `--apply` writes it to the forge |
| [`workline size`](commands.md#workline-size) | the normal size of an issue here, measured from the line's records, kept in one comment |
| [`workline wake`](wake.md) | after a merge, prints where the line may be drifting, with no AI; `--issue` keeps it in one issue |
| [`workline tick`](tick.md) | lists the line's jobs due now, each with why, with no AI: what a regular wake starts |
| [`workline follow`](commands-follow.md) | turns auto-merge on for what a person accepted, with the [merge policy](merge-policy.md); `--gardening`: rebuilds the roles' merge requests on their base's new tip |

### Set up

| Command | Does |
|---|---|
| [`workline hooks install --global`](commands-setup.md#workline-hooks) | installs the git hooks for every repository (`--repo`: this one); `uninstall` removes them |
| [`workline setup`](commands-setup.md#workline-setup) | sets workline up on this machine, asking what to enable |
| [`workline init`](commands-setup.md#workline-init) | adopts this repository |
| [`workline doctor`](commands-setup.md#workline-doctor) | says what is set up and what is missing |
| [`workline version`](commands-setup.md#workline-version) | the engine's version |

## Options of run-role and route

| Option | |
|---|---|
| `--repo <dir>` | the repository (default: here) |
| `--ai <agent>` | the agent: `none`, `claude`, `cmd:<command>`… ([below](#the-agent)) |
| `--input name=value` | an input for the role (`route`: for every step), repeatable |
| `--input-file name=path` | `run-role` only: an input read from a file, written back by the intention that targets it |
| `--forge <forge>` | where issues and merge requests live ([below](#the-forge)) |
| `--target issue:<n>` / `merge-request:<n>` | where comments and labels go |
| `--branch <name>` | the branch the `--target` merge request comes from ([below](#writing-to-a-merge-request)) |
| `--scope <glob>` | a path the task is about, repeatable |
| `--no-apply` | judge, then stop; apply later with `workline apply` |
| `--open-merge-request` | puts what the patches write in a merge request of its own ([below](#writing-to-a-merge-request)) |
| `--roles <dir>` | a folder of roles used instead of the shipped ones (or `WORKLINE_ROLES`) |
| `--sarif <file>` / `--code-quality <file>` | also write the findings for GitHub's code scanning (SARIF) or GitLab's Code Quality ([role outcome](spec/role-outcome.md)) |
| `--summary <file>` | also add what a CI job's page shows ([below](#the-jobs-summary)) |
| `--json` | print the result as JSON |

`route` takes more, for CI jobs (`--gates-only`, `--verdict`,
`--dry-run`…): [workline route](commands.md#workline-route), or
`workline route --help`.

### The agent

- `none`, `claude`, or `claude:<model>@<effort>` to force a model, an
  effort (`none`, `low`, `medium`, `high`, `xhigh`, `max`), or both,
  whatever the role asks: `claude:opus@high`.
- `cmd:<command>`: any other agent ([another agent](#another-agent)).
- Default: the project's `ai`, then yours ([config](config.md)).
- `claude` runs Claude Code headless, outside the repository, with no
  tools, so the project's `CLAUDE.md` is not loaded; a role that edits
  code (the developer's) gives it a worktree and the tools to work in it
  ([its agent](../roles/developer/docs/agent.md#claude)).
- `claude` gets your environment without the variables a Claude Code
  session exports (`CLAUDECODE`, `CLAUDE_CODE_ENTRYPOINT`…): started
  from a session, a commit's hook included, the call would run as its
  child, at twice the tokens ([measured](research/claude-session-env.md)).
  They are dropped even when you set one (`CLAUDE_EFFORT`). Your login
  (`CLAUDE_CODE_OAUTH_TOKEN`, `ANTHROPIC_API_KEY`…) and other settings
  pass.
- **No agent gets a forge token**: `claude` and `cmd:` are called
  without `GH_TOKEN`, `GITHUB_TOKEN`, `GITLAB_TOKEN`, CI's job token, the
  App's key or the signing key; the engine writes to the forge, the
  agent proposes.

### The forge

- `github`, `gitlab`: their tokens and hosts in
  [config](config.md#environment).
- `local` keeps issues and merge requests in the clone, never pushed
  (`workline issues` reads them).
- `cmd:<command>`: another forge, plugged by a command
  ([forge command](spec/forge-command.md)).
- `none`: what needs a forge is refused, and says so.
- Default: the project's `forge` ([config](config.md)).

### Writing to a merge request

- **`--target merge-request:<n>`**: comments and labels go there; what
  a role writes there:
  [ADR-0051](adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md#cut-3-no-docs-obligation-per-pr).
- **`--open-merge-request`**, with a forge: what the patches write goes
  on a branch of the role, with a merge request opened for it
  ([ADR-0006](adr/0006-gardening-opens-one-merge-request-per-task.md)).
  How that branch follows its base:
  [ADR-0034, amended](adr/0034-the-release-fix-follows-its-base.md#amendment-2026-10-12-gardenings-merge-requests-follow-too-or-are-closed).
- **`--branch`**: for a job that does not ask the forge; default: the
  forge says.

### The job's summary

- **`--summary <file>`**: a summary of the run added to the file, for a
  CI job's page (GitHub's `$GITHUB_STEP_SUMMARY`)
  ([ADR-0035](adr/0035-the-engine-writes-the-jobs-summary.md)).
  - Markdown, or HTML for a file named `.html`.
  - Added to, never overwritten, so a judging then an applying job write
    one page. Repeatable.

## Another agent

`cmd:<command>` makes any command the agent: another provider's CLI, or a
wrapper around your own. It runs with `sh -c`, outside the repository:

- it gets your environment, a Claude Code session's variables included,
  the forge's tokens left out;
- its input is the prompt: first the part every call of the role
  shares, then the task; a wrapper may cache the first part;
- `WORKLINE_TIER` and `WORKLINE_EFFORT` say what the role asks (`light`,
  `standard`, `frontier`; `none` to `max`), for it to pick its model;
- its output is the answer: the role's proposals, in YAML;
- it may write what answered, in YAML, to the file `WORKLINE_CALL` names:
  `model`, `tokens-in`, `tokens-cached`, `tokens-out`, `cost-usd`, and
  `turns` for an agent that loops;
- where a role checks the code out for it (the developer's), it runs in
  that worktree, its path in `WORKLINE_WORKTREE`: it edits the files in
  place, and what it says beside its edits never fails the call; its
  caps in `WORKLINE_MAX_SECONDS` and `WORKLINE_MAX_TURNS`, its process
  group killed past its time
  ([the caps](../roles/developer/docs/proof.md#the-caps));
- a non-zero exit, or no answer within the role's `model.timeout`, counts
  as an agent out of reach, as a used-up quota does: the run ends
  `blocked-external` if the role cannot pass without it.

A script of your own, or Claude Code run this way:

```sh
--ai 'cmd:~/bin/my-agent'
--ai 'cmd:echo "model: haiku" > "$WORKLINE_CALL"; claude -p --model haiku --tools ""'
```

## Exit codes

| Code | Status |
|---|---|
| 0 | `pass` |
| 1 | `block`, or an error |
| 2 | `human`: a person must decide |
| 3 | `blocked-external`: something outside failed (an agent's quota, its login…), never a verdict on the work |
| 64 | the command was misused |

Misuse exits 64, never 2, which says `human`; asking for help exits 0.

## Before a push

What the `pre-push` hook runs, and asking before each push:
[before a push](commands-setup.md#before-a-push).
