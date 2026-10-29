---
sources: [cmd/workline, internal/role/config.go, internal/engine/engine.go, internal/hooks, internal/agent/claude.go, internal/agent/command.go]
checked: 21ce8c2c
judged: b5899f5
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
| [`workline apply <run-dir>...`, `--line <file>`](commands.md#workline-apply) | applies runs judged with `--no-apply`; `--main <file>`: main's findings in one issue; `--release-dry-run <file>`: what would hold the release |
| [`workline review`](commands.md#workline-review) | the reviewer reads this branch, or a spec, before you push |
| [`workline gate <name>`](commands.md#workline-gate) | runs a gate declared in `.workline/config.yaml` |
| [`workline test check --base <ref>`](commands.md#workline-test) | the tester's red-first check: each test the change adds fails on the base |
| [`workline test write --issue <n>`](commands.md#workline-test) | the tester writes a ready issue's tests, proves them red, opens a Draft |
| [`workline test answer --merge-request <n>`](commands.md#workline-test) | the tester answers an objection to one of its tests: kept, or rewritten |
| [`workline test measure --history <file>`](commands.md#workline-test) | the tester's curator measures the test corpus: times, red and flaky tests |
| [`workline test curate --history <file>`](commands.md#workline-test) | the curator removes a few redundant tests, proven by coverage and mutants |
| [`workline develop`](commands.md#workline-develop) | the developer takes the first ready issue, writes its code, opens its pull request |
| [`workline item ready <id>`](commands.md#workline-item-ready) | moves a work item to `ready`, once its sections are written |
| [`workline issues`, `issues show <n>`](commands.md#workline-issues) | reads the issues kept in the clone (`forge: local`) |
| [`workline issues import <file>`](commands.md#workline-issues-import) | turns a roadmap or backlog file into issues |
| [`workline docs`](commands.md#workline-docs) | your agent fixes the docs made suspect; you keep or drop each fix |
| [`workline sample`](commands.md#workline-sample) | the weekly sample: one in ten of every role's acts of the week, re-checked |
| [`workline follow`](commands.md#workline-follow) | `--gardening`: rebuilds gardening's merge requests on the default branch's new tip; runs the [merge policy](merge-policy.md), when set |
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
| `--open-merge-request` | puts what the patches write in a merge request of its own ([below](#writing-to-a-merge-request)) |
| `--roles <dir>` | a folder of roles used instead of the shipped ones |
| `--sarif <file>` / `--code-quality <file>` | also write the findings for GitHub's code scanning (SARIF) or GitLab's Code Quality; `route`: every step's; on a merge request, the SARIF without the documentalist's, the reviewer's nor the gates' ([role outcome](spec/role-outcome.md)) |
| `--summary <file>` | also add what a CI job's page shows ([below](#the-jobs-summary)) |
| `--gates-only` / `--gates-from <file>` | `route` only: run the event's gates alone, for a CI job of their own; or read their verdicts from that run's result, with `--warn-only` their findings for the inspector ([gates in CI](gates-in-ci.md)) |
| `--gates-kept <folder>` | `route`, and `gate` with `--base`: the gates' findings kept by commit; on `main`, the head's written there; on a change, its merge base's read there ([main after a merge](main-after-merge.md)) |
| `--warn-apart` / `--warn-only` | `route` only: leave what only warns (the reviewer's lenses that warn, the inspector's triage) to a run apart; or be that run, no verdict ([apart](../roles/reviewer/docs/apart.md)) |
| `--release-wait <time>` | `route` only, on the release: wait that long (`30m`) for the checks not done on the branch it is cut from, then block only if one is red or still running; the CI job stays pending meanwhile ([what holds a release](release-holds.md#read-again)) |
| `--verdict <file>` | `route` only: also write the line's verdict on the head it judged, for the [merge policy](merge-policy.md); `--verdict-head <commit>`, the head when the commit judged is a merge result |
| `--json` | print the result as JSON ([below](#the-jobs-summary)) |

### The agent

- `none`, `claude`, or `claude:<model>@<effort>` to force a model (an
  alias or an exact id), an effort, or both, whatever the role's tier
  asks: `claude:opus`, `claude:@high`.
- `cmd:<command>`: any other agent ([another agent](#another-agent)).
- Default: `WORKLINE_AI` for the hook, else the project's `ai`, else
  yours, else none.
- `claude` runs headless, outside the repository, with no tools. It gets
  your environment without the variables a Claude Code session exports
  (`CLAUDECODE`, `CLAUDE_CODE_ENTRYPOINT`…): started from a session,
  a commit's hook included, the call would run as its child, at twice
  the tokens ([measured](research/claude-session-env.md)). They are
  dropped even when you set one (`CLAUDE_EFFORT`: the effort is the
  role's, or `claude:@<effort>`'s). Your login (`CLAUDE_CODE_OAUTH_TOKEN`,
  `ANTHROPIC_API_KEY`…) and other settings pass.
- **No agent gets a forge token**: `claude` and `cmd:` are called
  without `GH_TOKEN`, `GITHUB_TOKEN`, `GITLAB_TOKEN`, the App's key or
  the signing key; the engine writes to the forge, the agent proposes.

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

- **`--target merge-request:<n>`**, with a forge: what the patches write
  goes in one comment there, whole, to copy and apply with `git apply`:
  the docs it creates too, and a code block of a doc it changes does not
  end it. Nothing is pushed to its branch, not even a move of `checked`
  ([ADR-0051](adr/0051-a-check-sits-where-its-catch-is-worth-its-cost.md#cut-3-no-docs-obligation-per-pr)).
- **`--open-merge-request`**, with a forge: what the patches write goes on
  a branch `workline/<role>/<task>`, pushed, with a merge request opened or
  updated for it; the role is told how many of its merge requests are
  open ([ADR-0006](adr/0006-gardening-opens-one-merge-request-per-task.md)),
  and what each changes, so it asks nothing they hold again
  ([ADR-0013, amended](adr/0013-what-was-judged-is-not-asked-again.md#amendment-2026-10-14-a-doc-waits-while-a-merge-request-holds-it)).
  With `local`, the branch stays in the clone. A branch a person committed
  to is never pushed over (`not-proposed`); `workline follow --gardening`
  keeps the others on the base, or closes them
  ([ADR-0034, amended](adr/0034-the-release-fix-follows-its-base.md#amendment-2026-10-12-gardenings-merge-requests-follow-too-or-are-closed)).
- **`--branch`**: for a job that does not ask the forge (CI's judging
  job, which holds no token); default: the forge says. A release tool's
  branch (`release.branches`) has the versions the tool bumps left out
  of the review
  ([more](../roles/reviewer/docs/run.md#12-the-range-the-rules)).

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
  - A line's seconds, after the steps
    ([ADR-0050](adr/0050-each-act-has-its-moment-and-a-wait-budget.md#1-a-wait-budget-per-moment)):
    `66 s in all — committer 0.5 s, documentalist 0.8 s, reviewer 64 s
    (lenses 40 s, judges 20 s)`; a gate read from its job, `(its job)`.
  - Over its moment's budget, one line more, naming the step that took
    the most; never a block ([the budgets](spec/routing.md#each-steps-seconds-a-moments-budget)).
- **`--json`**: status, summary, findings, the agent's notes, each agent
  call (agent, tier, effort, the exact model that answered, tokens in,
  cached and out, cost, seconds), and for a line its steps and pending
  runs, each step's `seconds`, the line's, and its `moment`: `name`,
  `budget`, `seconds`, `over`, `step`.

## Another agent

`cmd:<command>` makes any command the agent: another provider's CLI, or a
wrapper around your own. It runs with `sh -c`, outside the repository:

- it gets your environment, a Claude Code session's variables included,
  the forge's tokens left out;
- its input is the prompt, the role's persona first;
- `WORKLINE_TIER` and `WORKLINE_EFFORT` say what the role asks (`light`,
  `standard`, `frontier`; `none` to `max`), for it to pick its model;
- its output is the answer: the YAML list of proposals, a code fence
  tolerated;
- it may write what answered, in YAML, to the file `WORKLINE_CALL` names:
  `model`, `tokens-in`, `tokens-cached`, `tokens-out`, `cost-usd`;
- a non-zero exit, or no answer within the role's `model.timeout`, ends
  the run as `blocked-external`, as a used-up quota does; no answer within
  what the role's `ai-max-seconds` leaves (the committer's: 30 s) ends it
  as if the agent had not answered, said (`ai-max-seconds`).

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
