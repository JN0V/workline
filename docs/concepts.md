---
sources:
  - defaults/routing.yaml#events
  - defaults/routing.yaml#fail-fast
  - defaults/routing.yaml#states
  - cmd/workline/cli.go#init
  - cmd/workline/main.go#exitFor
  - internal/routing/timed.go#TimedJobs
  - internal/gate/gate.go#Check
  - internal/gate/gate.go#RunNew
  - internal/kit/verdict/verdict.go#Finding
  - internal/kit/verdict/verdict.go#Finding.Blocks
  - internal/kit/verdict/verdict.go#Levels
  - internal/kit/verdict/verdict.go#Enforce
  - internal/engine/engine.go#RunsDir
  - internal/engine/engine.go#tiers
  - internal/engine/routine.go#underived
  - internal/forge/forge.go#Open
  - internal/agent/agent.go#Parse
checked: 542fc9cb6
verified: agent:claude-code
---
# Concepts

The words the other pages use, each in a few lines. The rules behind them:
[principles](PRINCIPLES.md).

## Roles, events and the line

- **Role** — one job, like a post on an assembly line: the committer, the
  documentalist, the reviewer, the product owner ([roles](roles.md)). A role
  is a folder around its `role.yaml`
  ([role contract](spec/role-contract.md#layout)).
- **Event** — what starts work: `commit-msg` and `pre-push` (git hooks on
  your machine), `merge-request`, `main` (after a merge,
  [how](main-after-merge.md)), `schedule` (gardening), `release`,
  `issue` (an issue's state), `reconcile`, and a few more
  ([triggers](triggers.md)). workline never watches for events: a hook, a
  CI job or you run the command for one.
- **Routing, the line** — which roles run, in order, for an event
  (`defaults/routing.yaml`, changed in `.workline/config.yaml`). `workline
  route <event>` runs the line; the first step that does not pass stops it,
  but on `schedule` and `main`, where every step runs and the worst verdict
  stands ([ADR-0037](adr/0037-the-schedule-runs-every-step.md)). A
  release has a dry run and reads what holds it on the forge
  ([release](release.md#with-each-gardening-a-dry-run),
  [what holds a release](release-holds.md)).
  `workline run-role <role> --event <event>` runs one role alone.
- **State** — where an issue stands on the line, one label at a time:
  to refine, needs tests, ready, developing, blocked. Each wakes its role
  on `issue`, whose result moves the issue on; a reconciler (`route
  reconcile`), hourly or at each [tick](tick.md), runs what an event missed
  ([the line of states](spec/routing.md#the-line-of-states)).
- **Gate** — a step that runs your own commands (tests, scanners) and
  reads their output against thresholds, no agent asked (`gate:<name>` in
  a line); on a merge request, only the findings the change adds count
  ([gates](spec/gates.md)).

## One run

1. **Prepare** (no AI) gathers what the role needs and runs the
   mechanical checks.
2. **Propose** (the agent) answers with proposals from the role's closed
   list: a patch, a comment, an issue, a commit message…
3. **Check** (no AI) refuses a proposal that breaks the role's rules.
4. **Apply** (the engine, no AI) writes what was accepted: the working
   tree, a commit, a comment, an issue.

In CI, `--no-apply` stops after the check; `workline apply` runs later, in
a job that holds the write token and no AI key. Each run is kept in
`.git/workline/runs/` ([one run](spec/role-contract.md#one-run)).

## Verdicts

| Status | Exit | Means |
|---|---|---|
| `pass` | 0 | nothing blocks; findings may still be shown |
| `block` | 1 | a rule refused the work, or an error |
| `human` | 2 | a person must decide |
| `blocked-external` | 3 | something outside failed (agent quota, forge): never a verdict on the work |

A **finding** has a rule, a place, a message and a level: `block` (when
none is said), `warn`, or another level that only tells; only `block`
blocks. `enforce` in the config lowers a rule to `warn` or `off`, never
raises one. How a rule moves from warn to block:
[warn before block](spec/role-outcome.md#warn-before-block).

## Without AI

`--ai none`: no agent is called, and each role's rule for running
without AI applies. Every check still runs and still blocks; what needed
judgement goes to a person. Each role's page says what it does then.

## Docs and code

A doc names the code it describes in its header, `sources`, and the
commit it was last checked against, `checked`. A change to those sources
makes the doc **suspect** until someone reads it again
([documentalist](../roles/documentalist/README.md)). **Gardening** is
the scheduled run (`schedule`) that keeps the repository in order. A
**derived block** (`<!-- workline:derive name -->`) is a fact
regenerated from the code
([ADR-0027](adr/0027-derived-blocks-are-regenerated-on-the-default-branch.md)).

## Forges

Where workline writes comments, issues and merge requests: `github`,
`gitlab`, `local` (files kept in the clone), `cmd:<command>`
(any other, plugged by a command), or `none`. Writes go where the
project lives ([ADR-0016](adr/0016-writes-go-where-the-project-lives.md)).

## Agents and independence

The agent is Claude Code (`--ai claude`, `claude:<model>@<effort>`) or any
command (`cmd:`); a role asks for a **tier** (`light`, `standard`,
`frontier`) and an effort, not a model. A **judge** checks what an agent
produced from another context, another model or another provider — the
best available
([ADR-0005](adr/0005-independence-takes-the-best-level-available.md)).

## People

- People state the need and accept the result; workline never approves.
  With the project's [merge policy](merge-policy.md), it turns the
  forge's auto-merge on for what a person accepted; the policy says what
  else merges alone. Else the person merges.
- A question the product owner asks times out, by its level, into an
  assumption written on the issue
  ([ADR-0060](adr/0060-the-person-is-on-the-loop.md)).
- What a role cannot settle is written where a person looks — an issue, a
  comment, a label — with what to do.
- A **parent** — a need the product owner split into parts, each an
  issue — is accepted by a person from what its parts delivered
  ([ADR-0029](adr/0029-a-parent-is-accepted-by-a-person-from-what-its-parts-delivered.md)).
