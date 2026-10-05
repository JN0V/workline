---
sources: [internal/engine/engine.go, internal/line, internal/routing, internal/verdict, routing.default.yaml]
checked: d293323
verified: agent:claude-code
---
# Concepts

The words the other pages use, each in a few lines. The rules behind them:
[principles](PRINCIPLES.md).

## Roles, events and the line

- **Role** — one job, like a post on an assembly line: the committer, the
  documentalist, the reviewer, the product owner ([roles](roles.md)). A role
  is a folder: `role.yaml` (its events, model, settings), facets the agent
  reads, and two programs, `pre` and `post`.
- **Event** — what starts work: `commit-msg` and `pre-push` (git hooks on
  your machine), `merge-request`, `schedule` (gardening), `release`,
  `review`, `import`, `init`. workline never watches for events: a hook, a
  CI job or you run the command for one ([triggers](triggers.md)).
- **Routing, the line** — which roles run, in order, for an event
  (`routing.default.yaml`, changed in `.workline/config.yaml`). `workline
  route <event>` runs the line; the first step that does not pass stops it.
  `workline run-role <role> --event <event>` runs one role alone.
- **Gate** — a step that runs your own scanners and reads their output
  against thresholds, never asking a model (`gate:<name>` in a line).

## One run

1. **Prepare** (`pre`, no AI) gathers what the role needs and runs every
   mechanical check. If nothing needs judgement, no agent is called.
2. **Propose** (the agent) answers with proposals from the role's closed
   list (`intentions`: a patch, a comment, an issue, a commit message…).
   It holds no write token.
3. **Judge** (`post`, no AI) refuses a proposal that breaks the role's
   rules: a quote not found again, a file outside what the role may write.
4. **Apply** (the engine, no AI key) writes what was accepted: the working
   tree, a commit, a comment, an issue.

In CI, `--no-apply` stops after judging; `workline apply` runs later in a
job that holds the write token and no AI key. Each run is kept in
`.git/workline/runs/`, with every agent call and what it cost.

## Verdicts

| Status | Exit | Means |
|---|---|---|
| `pass` | 0 | nothing blocks; findings may still be shown |
| `block` | 1 | a rule refused the work, or an error |
| `human` | 2 | a person must decide |
| `blocked-external` | 3 | something outside failed (agent quota, forge): never a verdict on the work |

A **finding** has a rule, a place and a level: `block` or `warn`. A new
rule starts as `warn`; `enforce` in the config moves a rule between `block`,
`warn` and `off`.

## Without AI

`--ai none`, no quota, no network: every check still runs and still
blocks; what needed judgement goes to a person (a list in a comment, the
report, the job's log). Each role's page says what it does then.

## Docs and code

A doc names the code it describes in its header, `sources`, and the commit
it was last checked against, `checked`. A commit changing one of those
sources makes the doc **suspect** until someone reads it again: the agent,
which fixes it or moves `checked`, or a person. **Gardening** is the
scheduled run (`schedule`) that keeps the whole repository in order, one
merge request per task. A **derived block**
(`<!-- workline:derive name -->`) is a fact regenerated from the code.

## Forges

Where workline writes comments, issues and merge requests: `github`,
`gitlab`, `local` (files kept in the clone, never pushed), `cmd:<command>`
(any other, through a small contract), or `none`. Writes go where the
project lives (ADR-0016).

## Agents and independence

The agent is Claude Code (`--ai claude`, `claude:<model>@<effort>`) or any
command (`cmd:`); a role asks for a **tier** (`light`, `standard`,
`frontier`) and an effort, not a model. A **judge** checks what an agent
produced from another context, another model or another provider — the
best available, said in the verdict (ADR-0005).

## People

People state the need and accept the result; workline never merges, never
approves. What a role cannot settle is written where a person looks — an
issue, a comment, the report — with what to do. A **parent** — a need the
product owner split into parts, each an issue — is never closed by a
role: as its parts close, a comment on it says what each delivered and
which items of its Verification are proved, and a person accepts the need
by closing it (ADR-0029).
