---
sources: [cmd/workline, internal/role/config.go, internal/engine/engine.go, internal/hooks]
checked: 245015c
verified: agent:claude-code
---
# Using workline

The commands, their options and exit codes, as the engine reads them today.
What a role does is on its page ([roles.md](roles.md)); the files, the
settings and the variables, in [config.md](config.md).

## Help

- `workline --help`: every command, one line each (`hook`, `builtin`:
  internal); `workline <command> --help`: its usage and options; exit 0.
- Misuse (an unknown command or option, a bad value, a missing `<role>`
  or `<event>`, a forge item id not a number) names the fault, then the
  usage, on stderr: exit 64.
- Tried: every command's help; CI's and the hooks' calls with `--ai none`,
  codes unchanged. Not tried: a GitLab job written from the help alone.

## Commands

Each in detail: [the commands](commands.md), and
[the commands that set up](commands-setup.md).

| Command | Does |
|---|---|
| [`workline run-role <role> --event <event>`](commands.md#workline-run-role) | runs one role |
| [`workline route <event>`](commands.md#workline-route) | runs the roles routing names for the event, in order |
| [`workline apply <run-dir>...`, `--line <file>`](commands.md#workline-apply) | applies runs judged with `--no-apply` |
| [`workline review`](commands.md#workline-review) | the reviewer reads this branch, or a spec, before you push |
| [`workline gate <name>`](commands.md#workline-gate) | runs a gate declared in `.workline/config.yaml` |
| [`workline item ready <id>`](commands.md#workline-item-ready) | moves a work item to `ready`, once its sections are written |
| [`workline issues`, `issues show <n>`](commands.md#workline-issues) | reads the issues kept in the clone (`forge: local`) |
| [`workline issues import <file>`](commands.md#workline-issues-import) | turns a roadmap or backlog file into issues |
| [`workline docs`](commands.md#workline-docs) | your agent fixes the docs made suspect; you keep or drop each fix |
| [`workline sample`](commands.md#workline-sample) | the weekly sample: a second look at the docs confirmed that week |
| [`workline follow`](commands.md#workline-follow) | rebuilds a release fix on the default branch's new tip |
| [`workline hooks install --global`](commands-setup.md#workline-hooks) | installs the git hooks for every repository (`--repo`: this one) |
| [`workline setup`](commands-setup.md#workline-setup) | sets up this machine: hooks, agent, tools |
| [`workline init`](commands-setup.md#workline-init) | adopts this repository |
| [`workline doctor`](commands-setup.md#workline-doctor) | says what is set up and what is missing |
| [`workline version`](commands-setup.md#workline-version) | the engine's version |

## Options of run-role and route

| Option | |
|---|---|
| `--repo <dir>` | the repository (default: here) |
| `--ai <agent>` | the agent: `none`, `claude`, `cmd:<command>`… ([below](#the-agent)) |
| `--input name=value` | an input for the role (`route`: for every step), e.g. `range=<base>..<head>` for the committer on a merge request |
| `--input-file name=path` | `run-role` only: an input read from a file, written back by the intention that targets it (the hook's message file) |
| `--forge <forge>` | where issues and merge requests live: `github`, `gitlab`, `local`, `cmd:`, `none` ([below](#the-forge)) |
| `--target issue:<n>` / `merge-request:<n>` | where comments and labels go |
| `--branch <name>` | the branch the `--target` merge request comes from ([below](#writing-to-a-merge-request)) |
| `--scope <glob>` | a path the task is about, repeatable; a patch outside it is refused |
| `--no-apply` | judge, then stop; apply later, in a job that holds the write token |
| `--push-to-merge-request` | commits what the patches write to the merge request's branch ([below](#writing-to-a-merge-request)) |
| `--open-merge-request` | puts what the patches write in a merge request of its own ([below](#writing-to-a-merge-request)) |
| `--roles <dir>` | a folder of roles used instead of the shipped ones |
| `--sarif <file>` / `--code-quality <file>` | also write the findings for GitHub's code scanning (SARIF) or GitLab's Code Quality; `route`: every step's ([role outcome](spec/role-outcome.md)) |
| `--summary <file>` | also add what a CI job's page shows ([below](#the-jobs-summary)) |
| `--json` | print the result as JSON ([below](#the-jobs-summary)) |

### The agent

- `none`, `claude`, or `claude:<model>@<effort>` to force a model (an
  alias or an exact id), an effort, or both, whatever the role's tier
  asks: `claude:opus`, `claude:@high`.
- `cmd:<command>`: any other agent ([another agent](#another-agent)).
- Default: `WORKLINE_AI` for the hook, else the project's `ai`, else
  yours, else none.

### The forge

- `github` needs `gh`.
- `gitlab` uses its API, with `GITLAB_TOKEN`, else glab's token if glab
  is set up, else CI's job token; the instance and project from CI, else
  `GITLAB_HOST` and the remote.
- `local` keeps issues and merge requests in the clone, never pushed
  (`workline issues` reads them).
- `cmd:<command>`: another forge, plugged by a command
  ([forge command](spec/forge-command.md)).
- `none`; default: the project's `forge`.

### Writing to a merge request

- **`--push-to-merge-request`**, with a forge and `--target
  merge-request:<n>`: what the patches write is committed to that merge
  request's branch and pushed, without force.
  - From a fork, or when the branch moved on, the diff goes in one comment
    instead. With `local`, committed on the local branch, nothing pushed.
  - On a release tool's merge request, which the tool rewrites, it goes to
    a merge request of its own, `workline/<role>/release`, into the
    release's base; the release waits until it is merged
    ([ADR-0017](adr/0017-the-release-manager.md)).
- **`--open-merge-request`**, with a forge: what the patches write goes on
  a branch `workline/<role>/<task>`, pushed, with a merge request opened or
  updated for it; the role is told how many of its merge requests are
  open ([ADR-0006](adr/0006-gardening-opens-one-merge-request-per-task.md)).
  With `local`, the branch stays in the clone.
- **`--branch`**: for a job that does not ask the forge (CI's judging
  job, which holds no token); default: the forge says. A release tool's
  branch (`release.branches`) has the merge request held as the release
  ([ADR-0017](adr/0017-the-release-manager.md)).

### The job's summary

- **`--summary <file>`**: the verdict, each step's verdict and findings,
  the agent's calls and notes, what was applied and what waits to be
  ([ADR-0035](adr/0035-the-engine-writes-the-jobs-summary.md)).
  - Markdown (GitHub's `$GITHUB_STEP_SUMMARY`), or HTML for a file named
    `.html`.
  - Added to, never overwritten, so a judging then an applying job write
    one page. Repeatable; `apply`, `sample` and `issues import` take it
    too.
  - The product owner's findings grouped, one line an issue linked to
    its page ([what it writes](../roles/product-owner/docs/outputs.md#in-the-ci-jobs-summary)).
- **`--json`**: status, summary, findings, the agent's notes, each agent
  call (agent, tier, effort, the exact model that answered, tokens in,
  cached and out, cost, seconds), and for a line its steps and pending
  runs.

## Another agent

`cmd:<command>` makes any command the agent: another provider's CLI, or a
wrapper around your own. It runs with `sh -c`, outside the repository:

- its input is the prompt, the role's persona first;
- `WORKLINE_TIER` and `WORKLINE_EFFORT` say what the role asks (`light`,
  `standard`, `frontier`; `none` to `max`), for it to pick its model;
- its output is the answer: the YAML list of proposals, a code fence
  tolerated;
- it may write what answered, in YAML, to the file `WORKLINE_CALL` names:
  `model`, `tokens-in`, `tokens-cached`, `tokens-out`, `cost-usd`;
- a non-zero exit, or no answer within the role's `model.timeout`, ends
  the run as `blocked-external`, as a used-up quota does.

A script of your own, or Claude Code run this way (the command the
judge's trial used):

```sh
--ai 'cmd:~/bin/my-agent'
--ai 'cmd:echo "model: haiku" > "$WORKLINE_CALL"; claude -p --model haiku --tools ""'
```

## Exit codes

| Code | Status |
|---|---|
| 0 | `pass` (findings may still be shown) |
| 1 | `block`, or an error |
| 2 | `human`: a person must decide |
| 3 | `blocked-external`: something outside failed (agent quota, forge, a repository) — never a verdict on the work |
| 64 | the command was misused |

`workline gate` returns 0 or 1. An unknown option exits 64, as any misuse,
never 2, which says `human`; asking for help exits 0.

## Before a push

A push asks nothing by default; the `pre-push` hook runs the project's
`pre-push` line with no agent, when the project routes it. Asking before
each push, and what the hook does: [before a push](commands-setup.md#before-a-push).
