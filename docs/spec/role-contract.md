---
sources: [internal/engine/backlog.go#report, internal/engine/demote.go#commits, internal/engine/demote.go#holds, internal/engine/engine.go#feedback, internal/engine/engine.go#script, internal/engine/engine.go#unanswered, internal/engine/follow.go#follow, internal/engine/routine.go#alone, internal/role/config.go#shape, internal/role/config.go#config, internal/role/config.go#check, internal/role/role.go#Role.Writes, internal/agent/agent.go#Agent, internal/agent/claude.go#claude, internal/agent/claude.go#without, internal/agent/prompt.go#facet, internal/agent/prompt.go#Skills, internal/agent/prompt.go#Tokens, tests/roles_test.go]
checked: a19e8cb0
verified: agent:claude-code
judged: b530b72
---
# Role contract — v1 (draft)

A role is one job on the line: it takes a precise input, produces a precise output,
and hands over. This document defines what a role is, so that any engine, on any
machine, can run it the same way. The engine is replaceable; this contract is not
(ADR-0001).

## Layout

```
roles/<name>/
  role.yaml         the contract below
  README.md         for humans: what pre and post check; never loaded for the AI
  persona.md        facets: what the AI reads, and nothing else (see "Facets")
  knowledge/*.md
  instruction.md
  policy.md
  pre               optional; deterministic: gathers input, decides whether a decision is needed
  post              optional; deterministic: validates, decides the verdict
  lenses/*.md       the reviewer's: one facet per lens
  kinds/*.md        the auditor's: one facet per kind of act
  tasks/*.md        optional: a kind of task's own instruction (model.tasks)
  skills/<n>/SKILL.md  optional: the role's own copy of a shared skill (see "Shared skills")
  docs/             about the role, never loaded: reference pages, status.md, tried.md (archived)
```

- The top level holds README.md and what the engine loads, nothing else.
- The target, reached before v1.0, is in [layout.md](layout.md):
  `prompts/`, `criteria/`, `cases/` and `code/` in each role's folder.
- A page about the role goes to `docs/`; `tests/roles_test.go` fails otherwise.

`pre` and `post` are executables in any language. The engine runs them; it does
not care what they are written in.

Each is optional, on its own (ADR-0070, §1). With no `pre`, the shared one
writes `in/task.md` from the role's `instruction.md` and `in/skills` from
its declared `context.skills`; a missing `instruction.md` stops the run,
naming the file. With no `post`, the shared one passes and applies nothing:
what the agent proposed is held to the role's `intentions` and
`duties.writes` where it is applied, an intention not declared or a write
outside the duties refused and named.

## Where roles come from

A role is a folder, so it can live anywhere git can reach. A project says which
roles it uses and where each comes from, pinned to a version:

```yaml
# .workline/config.yaml
roles:
  committer:     {from: builtin}
  documentalist: {from: "https://gitlab.example.com/tools/roles.git//documentalist", ref: v1.4.0}
```

*Not built yet: the engine refuses `from` today. What exists is `--roles <dir>`
or `WORKLINE_ROLES`, a folder of roles used instead of the shipped ones.*

The engine fetches each role at its pinned version and records what it got —
commit and content digest — in `.workline/roles.lock`, so every machine and
every CI job runs the same role. A role fetched from elsewhere is still bound by
the contract: it declares its `requires`, `duties` and `intentions`, and the
engine enforces them whatever the role's origin.

The roles shipped with the engine live in this repository, next to the contract
and its conformance tests, because while the contract changes, a change to it
and to every role must land together. A team keeps its own roles — or its own
version of a shipped one — in its own repository, and a project can still
override a single facet locally without forking the role.

## `role.yaml`

```yaml
contract: 1                     # version of this document
name: committer
mission: One sentence a newcomer understands.

events: [commit-msg, merge-request]   # what the role can be run on

requires: [git]                 # binaries that must be on PATH; anything beyond
                                # git and the engine is declared here, visibly
uses: []                        # optional binaries: used when present; when
                                # missing, the verdict says which check did not run
                                # (not read yet)

model:                          # what kind of thinking, never a model name
  capability: writing           # see model-grid.md
  tier: light
  effort: low
  timeout: 3m                   # how long one answer may take; 3m by default
  tasks:                        # a kind of task that needs other thinking, as pre
    condense: {tier: frontier}  # names it in in/task-kind; unset fields keep the role's
    # kind: {instruction: tasks/kind.md, intentions: [kind, note]}
    #   its own instruction facet, read in place of instruction.md, and the
    #   few intentions its agent answers with; any other it answers is
    #   dropped, said. What pre falls back on keeps the role's intentions.

context:
  knowledge: []                 # which knowledge/ files to load; none by default
  skills: []                    # shared skills a run may load, when its pre
                                # names them (see "Shared skills"); none by default
  budget: 16000                 # tokens, estimated 711 + 0.82 a character
                                # (agent.Tokens); the engine refuses a larger prompt:
                                # the agent is not asked, the run goes on without it,
                                # and says so (`prompt-over-budget`, a part's
                                # `part-unanswered`), never an engine error; on a
                                # gate's event the run then ends `human` (a part:
                                # as the role treats a part not answered); on
                                # schedule, reported only (ADR-0037)

duties:
  reads: ["**"]                 # paths the role may read
  writes: []                    # paths a patch may touch; empty = none;
                                # `$settings.<name>` uses a project setting;
                                # `test-paths` unset: the generic test files

intentions: [commit-message, note]   # subset of the catalogue below
part-intentions: [claim]        # what a part of a question answers with (In parts);
                                # claims when unset; a reviewer's lens, findings;
                                # the inspector, triages
on-block: []                    # the kinds a run still applies when post blocks:
                                # what only says why, every proposal of them, on
                                # any run (step 5, Apply); none by default
on-moved-on: []                 # what a merge request's run still applies once its
                                # branch moved on from the head judged: what only says
                                # what that head held (the reviewer's review, the
                                # inspector's triage); never a patch nor a label;
                                # none by default
apart: []                       # events, of its own, where what the role finds that
                                # only warns runs apart from the verdict: the verdict's
                                # line leaves it (`route --warn-apart`, input
                                # `warn=apart`), a line of its own runs it alone
                                # (`--warn-only`, `warn=only`), never the head's
                                # verdict (ADR-0050; the reviewer's lenses that warn,
                                # the inspector's triage); none by default
personal: []                    # settings a person may set for themselves, in their
                                # own config (role-adapting.md, "A person's own")

without-ai: block               # block | report | pass — see "No AI" (not read yet)
merge-request-ai: ready         # on a merge request, the agent judges once, on the
                                # run at ready (routing.md, "Once, at ready"); unset:
                                # on every push but a draft's

settings:                       # defaults, overridable per project
  subject-max: 72

levels: {}                      # named sets of settings a project picks with
                                # one setting, laid over the defaults first
                                # (role-adapting.md, "Levels")
```

## Facets

What the AI reads is split by purpose, so each part can be changed, overridden
or reused on its own. The engine assembles them in a fixed order:

| Facet | Holds | Required |
|---|---|---|
| `persona.md` | who the AI is in this role | no |
| `knowledge/*.md` | facts it needs; loaded only when listed in `context.knowledge` | no |
| `instruction.md` | the task, in a few lines; a kind of task may name its own (`model.tasks.<kind>.instruction`) | yes |
| output contract | the schema of `out/intentions.yaml`, restricted to the role's `intentions` | generated |
| the task | `in/task.md`, then the skills `pre` named, then a refused answer's feedback | written by `pre` |
| `policy.md` | the rules it must follow | no |

A role may keep its facets, and its `knowledge/`, `lenses/`, `tasks/` and
`skills/` folders, under a `prompts/` folder. A role that has one is read
under `prompts/` only, in the project's `.workline/roles/<role>/prompts/`
and the person's `roles/<role>/prompts/` as in the role's own folder; a file
missing there is missing. A role without one is read from its top level.

The first four are the system prompt: the same, byte for byte, in every
call of the same role and mode (of a kind of task, for one with its own
instruction; with a worktree or without, whose answer's shape differs).
An agent that caches a prompt's prefix, as Claude Code does, reads them
back on the role's next call instead of paying them fresh
([#1128](https://github.com/JN0V/workline/issues/1128)). The task and the
policy are the user message; a `cmd:` agent reads both, system prompt
first. The policy comes last on purpose: models follow what they read last
more closely. The output contract is generated by the engine from
`intentions`, so a role cannot describe an intention it is not allowed to
emit.

Each facet is looked up in the project (`.workline/roles/<name>/`), then in the
user's config folder, beside their config.yaml (`~/.config/workline/roles/<name>/`;
macOS: `~/Library/Application Support/workline/roles/<name>/`, then
`~/.config/workline/roles/<name>/`), then in the role as shipped.
The first one found wins, so a project can replace one facet without copying the
role.

## Shared skills

A method several roles use — the five whys first — ships once, in
`skills/<name>/SKILL.md` at the repository's root, in the
[Agent Skills](https://agentskills.io/specification) format: a
frontmatter with `name` and `description`, then the method.

- **Declared**: a role lists the skills it may load in
  `context.skills`; any other is refused.
- **Triggered by the role's `pre`**, never chosen by the agent: when the
  task calls for one, `pre` writes its name in `in/skills`, one a line.
  The role's code decides when (the tester: an objection's thread still
  open after two answers).
- **Given whole, after the task**, under `# Skill: <name>`, its
  frontmatter left out; the role's context budget counts it.
- **Looked up** as a facet (`skills/<name>/SKILL.md` in the role's
  folders, the project's first), then in the project's
  `.workline/skills/<name>/`, then as shipped.
- **Its answer** is what the skill says, among the role's intentions;
  the role's `post` checks its form. The five whys: one `note`, read by
  the engine's `whys` package
  ([role-adapting.md](role-adapting.md#defects-become-guards)).

## One run

One run of a role is in [role-contract-run.md](role-contract-run.md).

### When apply stops half-way

Writes to git and to a forge are not transactional: a comment can be posted and
the label call fail. So every intention is applied idempotently — a label that
is already there, a comment carrying the run's marker, a comment on a line
whose key an open thread holds, a merge request already
open from the branch (updated, not opened again), the commits a stopped run left on
the role's branch (pushed as they are, `already-committed`: nothing new to commit) all
count as done — and `workline apply <run-dir>` resumes a
run from `out/run.yaml`, with the evidence of the original run, without calling
the agent again.

Scripts receive `WORKLINE_RUN_DIR`, `WORKLINE_EVENT`, `WORKLINE_AI` (the agent
name or `none`), `WORKLINE_AI_WAITS` (`ready` on a draft: given the input
`draft=true`, the engine sets the agent it would give aside, `WORKLINE_AI`
`none`, and the result says so, `ai-waits`, ADR-0050; `night` so too, on
a release tool's merge request run on `merge-request`, for a role that
runs on `release`), `WORKLINE_ROLE`, `WORKLINE_BIN` (the engine running them,
which the shipped roles call for their built-in steps), `WORKLINE_ROLES_DIR`
(the folder the role was taken from, where its siblings are),
`WORKLINE_FORGE` (the
configured forge, when one is set), `WORKLINE_TARGET`
(`merge-request:12`, `issue:3`) when a forge and a target are given, and
`WORKLINE_OPEN_MERGE_REQUESTS`, how many of the role's merge requests are
open, and `WORKLINE_OPEN_MERGE_REQUEST_TASKS`, their tasks (each branch, the
role's prefix cut, separated by spaces), when the run opens one (ADR-0006,
0013). The tasks an earlier round of the same run proposed and did not open
yet (`--no-apply`) are counted among them, and also given alone in
`WORKLINE_PROPOSED_TASKS`. `in/waiting.yaml` then names, for each of those
tasks, the files its change touches: an open one's branch from where it
left the commit judged, fetched from `origin` (the clone's own branch for
a forge kept there); an earlier round's, the files its patches name. A
branch that cannot be read leaves its task out, said
(`merge-request-unread`, a warning; ADR-0013, amended 2026-10-14).
`in/held.yaml` names, by number, the files each other open merge request
of the repository changes, a fork's left out: the documentalist leaves
those docs to it (ADR-0051). On a merge request a release tool opened (its
branch one of `release.branches`), every role is told the branch,
`WORKLINE_RELEASE_BRANCH`: the reviewer leaves the versions the tool bumps
(#384). `workline follow --gardening`, after gardening applies, rebuilds the
role's merge requests on their base as it moves, or closes those with
nothing left to merge (ADR-0034, amended). A task's branch a person committed to is never force-pushed over
(`not-proposed`, ADR-0034 amended).

None of these reaches the project's commands a script runs, its tests,
its checks or the agent (`forge.LineEnv`;
[the line's own](../environment-roles.md#the-lines-own), #953).

### Exit codes

| Code | `pre` | `post` |
|---|---|---|
| 0 | continue | pass |
| 1 | error | block |
| 2 | — | a human must decide |
| 3 | an outside service failed (auth, quota, network); a verdict it wrote keeps its findings | same |
| 10 | no question for the agent: stop; a verdict `pre` wrote is final, none means pass | — |

Anything else is an error. An error is loud and blocks; it never passes.

## What a run produces, and adapting a role

The verdict, how hard its rules bite, the intentions catalogue, staying on the
task and running with no AI are in [role-outcome.md](role-outcome.md). Settings,
the levels for adapting a role to a project, and how defects become guards are
in [role-adapting.md](role-adapting.md).

## Not built yet

What this contract describes and the engine does not do yet, each marked where
it is described: roles taken from elsewhere (`from`, `roles.lock`), `uses`,
`without-ai`, baselines, the defect ledger. The rest is built and
covered by the conformance cases (conformance.md); routing and gates are in
routing.md and gates.md.
