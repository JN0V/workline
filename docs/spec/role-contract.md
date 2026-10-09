---
sources: [internal/engine, internal/intent, internal/role, internal/agent, roles_test.go]
checked: fb2293a
verified: agent:claude-code
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
  pre               deterministic: gathers input, decides whether a decision is needed
  post              deterministic: validates, decides the verdict
  lenses/*.md       the reviewer's: one facet per lens
  skills/           optional, Agent Skills format (SKILL.md); not read yet
  docs/             about the role, never loaded: reference pages, status.md, tried.md
```

- The top level holds README.md and what the engine loads, nothing else.
- A page about the role goes to `docs/`; `roles_test.go` fails otherwise.

`pre` and `post` are executables in any language. The engine runs them; it does
not care what they are written in.

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

context:
  knowledge: []                 # which knowledge/ files to load; none by default
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
                                # `$settings.<name>` uses a project setting

intentions: [commit-message, note]   # subset of the catalogue below
part-intentions: [claim]        # what a part of a question answers with (In parts);
                                # claims when unset; a reviewer's lens, findings
on-block: []                    # what a run still applies when post blocks: what
                                # only says why (the reviewer's comment); none by default

without-ai: block               # block | report | pass — see "No AI" (not read yet)

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
| `persona.md` | who the AI is in this role; goes in the system prompt | no |
| `knowledge/*.md` | facts it needs; loaded only when listed in `context.knowledge` | no |
| `instruction.md` | the task, in a few lines | yes |
| output contract | the schema of `out/intentions.yaml`, restricted to the role's `intentions` | generated |
| `policy.md` | the rules it must follow | no |

The policy comes last on purpose: models follow what they read last more
closely. The output contract is generated by the engine from `intentions`, so a
role cannot describe an intention it is not allowed to emit.

Each facet is looked up in the project (`.workline/roles/<name>/`), then in the
user's config (`~/.config/workline/roles/<name>/`), then in the role as shipped.
The first one found wins, so a project can replace one facet without copying the
role.

## One run

```
run-role <name> --event <event> [--ai <agent>|none]
```

Three trust zones, in the order the Linux Foundation uses for its issue triage:
**prepare** (trusted, no AI), **propose** (the agent, untrusted, no write
token), **apply** (trusted, no AI key).

1. **Check.** The engine reads `role.yaml`, verifies `requires`, creates a run
   directory with `in/` and `out/` inside the repository's git directory
   (`.git/workline/runs/<id>/`, so a run never shows up as a change), and
   writes the merged settings to `in/settings.json` — a level the project
   picks laid over the defaults first, and `by-level` beside them
   (role-adapting.md, "Levels").
2. **Prepare — `pre`.** Gathers what the role needs into `in/`. If a decision
   needs judgement, it writes the question to `in/task.md`. No `task.md`, no AI
   call: the AI is paid for decisions, not for routine. It may also write
   `in/fallback.yaml`: the proposals to use when no agent answers — the
   documentalist's derived blocks, for example; one marked `if-answered:
   true` is written only when the agent's answer was read (the product
   owner's record that it read an issue). And `in/task-kind`: one word
   naming the kind of question, when `model.tasks` asks something else of it.
   The engine then records a digest of `in/`.

   **In parts** (ADR-0009). A question too large for one call is written as
   several instead: `in/parts/<name>/task.md`, each whole in itself, and no
   `in/task.md`; or several lenses on one change, each a question of its
   own (ADR-0020), or all of them in one part (`lenses-together`, #147).
   The engine asks each part in a context of its own, one after the other,
   on the tier `model.tasks.part` names, each call counted and kept like
   any other, named by its part (`for: <name>` in `out/calls.jsonl`); a part answers only what the role's
   `part-intentions` name — `claim`s when it names none, a reviewer's lens
   `finding`s — which are never applied: they inform. The answers are put
   back as `in/parts/<name>/answer.yaml`, and `pre` runs again, with
   `WORKLINE_PARTS=answered`, to put them together — no AI — and write the
   one question that follows, `in/task.md`, or none. Only then is the digest
   recorded. A part that fails, or answers what it may not, is said
   (`part-unanswered`), never read as an answer, and why is written beside
   its task, `in/parts/<name>/unanswered`, its first word the kind of
   failure (`unavailable`, `spent`, `invalid`, `refused`); a lens proposing
   anything but findings is refused besides (`intention-refused`). An answer
   that reads not at all is asked for again once, as the main question's
   (step 3), when the role asks again at all (`promote-after`): the call
   counted like any other, said (`part-asked-again`), the answer not read
   kept as `out/unread-answer.txt`; still unreadable, the part fails. One
   broken in places is read claim by claim (`part-partly-read`); without an
   agent, no part is asked, and `pre` runs again with none answered.

   **Questions for a judge** (ADR-0020). `pre` may then write questions
   only a judge answers, one each: `in/judge/<key>/question.yaml`,
   `{question, material}`, and `author`, the model whose work is judged
   when an earlier run's agent did it (an issue announced obsolete,
   ADR-0024). The engine asks each apart, at the best independence
   available from that author, else from the agents that answered the run
   (as in step 4), writes its answer beside it — `answer.yaml`, `{yes, why, model,
   judge, author, level}`, or `{error}`; the call kept with the run's,
   `out/calls.jsonl`, named by its question (`for: judge/<key>`) — and runs `pre` again with
   `WORKLINE_JUDGED=answered`, to read them. A no refuses nothing there:
   `pre` reads the answers one by one (the reviewer drops a finding judged
   no; the product owner keeps an announced issue open). Without an agent, or once `ai-max-tokens` is spent, none is asked;
   the agent unreachable, the rest are not either.
3. **Propose — agent.** Only if `in/task.md` exists and `--ai` is not `none`.
   The agent runs on the model the grid resolves for the role's `model` needs
   (`model-grid.md`), receives the facets and `in/task.md`. It runs read-only
   and writes one file: `out/intentions.yaml`. The agent's proposals replace
   the fallback ones of the same kind — for patches, only those of the files
   the agent's patches touch; fallback proposals of other kinds stay. A
   fallback patch given as a diff (the documentalist's derived blocks, with no
   line of context) is never replaced: it is applied after the agent's.
   With no agent, or none that answered, the fallback proposals are used alone.
   An answer that cannot be read as proposals (`agent-invalid-output`, with
   what the YAML reader said) is asked for again once, on the same tier, with
   that error, when the role asks again at all (`promote-after`) and `post`
   did not block: nothing of it was judged, and a quote left unescaped is a
   slip, not a refusal. The documentalist's `post` passed with no proposal,
   and such a doc was judged again, whole, the next night (DomoticsCore,
   ADR-0014 step 4). The answer not read is kept as `out/unread-answer.txt`.
   The output contract asks for text holding code, quotes, a backtick, `: `
   or ` #` as a YAML block scalar (`|`), which takes code as it is; the
   request again repeats it (#138). YAML, not JSON: code wrapped in JSON
   comes out worse (docs/research/portability.md).
   Before any of that, the engine mends three slips agents keep making, in
   the one place every role's answers are read (`agent.Mend`), and says
   what it mended (`answer-mended`, the answer as it came kept in
   `out/agent-answer.txt`): code quoted with its tabs under a block
   scalar — its first line starting with a tab, or lines pasted at the
   start of the line — which the YAML reader refuses as indentation, is
   indented with spaces, and given an indentation indicator (`|2`) when
   its first line starts with a tab, the text of the block kept as
   written (PR #157); this only when the answer does not read for that
   reason, and never tabs indenting the YAML itself. A value opening on a
   quoted phrase and going on after it (`title: "60 minutes" is unclear`),
   which the reader refuses, is read whole, its quotes kept, when the
   answer does not read for that reason (#128), never a line of a block
   scalar; one closing on the quote it opened with, its inner quotes
   unescaped, is asked again. An answer holding both slips has each
   mended in turn. A plain value
   followed on its line by ` #…`, which the reader would drop as a
   comment, is read whole, as written, when it is free text — several
   words, or a `#` stuck to what follows (`#20`) — while a single word
   annotated `# a note` (a severity, a path) keeps only its value (#156).
   Read whole rather than asked again: the text on the line is exactly
   what the agent wrote, a quote included, and asking costs a call that
   may repeat the slip.
   *Claude Code runs without tools; a `cmd:` agent is not sandboxed, so
   read-only is its command's promise, not the engine's.*
4. **Judge — `post`.** Reads `in/` and, if present, `out/intentions.yaml`.
   Writes `out/verdict.yaml`. It guards what the AI must not decide (for
   example, the committer's `post` refuses a rewritten message that fails the
   same checks, or carries a secret). A question no check can answer, `post` does not answer: it
   writes it to `out/judge.yaml` (`{question, material}`), and once `post`
   passes, the engine puts it to a judge model at the best independence
   available from the agent (ADR-0005): `WORKLINE_JUDGE` when set, else
   another Claude model (Sonnet for work of Opus, Opus otherwise), else the
   same agent in a context of its own. A no refuses the proposal with its
   reason (`judged-no`), and the agent is asked again; a yes is reported
   (`judged`), with the level and both models. Asked again as often as the
   role allows and still refused, the files the refusals name are taken out
   of the patches, and what is left — some of the agent's own — is judged
   once more, without asking: passed, it is applied, and each file left out
   is a `left-out` finding with the refusal's reason. A role listing `claim`
   among its intentions may give claims beside a patch, saying why it takes
   words out (the documentalist, ADR-0014): `post` reads them; alone, with no
   patch, they are refused, and they are never applied. A `post` that passes
   may narrow the proposals in `out/intentions.yaml` — a place it refuses
   taken out, the rest kept — rather than block for the whole task to be
   asked again (the documentalist, ADR-0014 step 4); what it leaves there is
   what is applied, held to the same catalogue and bounds.
5. **Apply.** The engine checks that `in/` still matches its digest, then
   validates the intentions against the catalogue, the role's `intentions` list
   and its `duties.writes`. An invalid intention set is refused whole. Valid
   intentions are applied in the catalogue's order — files first, then what
   depends on them, then what only informs — whatever order the agent used,
   and each one is recorded in `out/run.yaml` (`applied:`) as it succeeds.
   A role that keeps a backlog comes here with no intention too, when its
   `pre` did not end the run: each issue's state is read — a person's
   answer, an act undone — and written back (docs/spec/backlog-acts.md).
   A run that blocks applies nothing, but the kinds its role names in
   `on-block`, of its intentions: what only says why it blocks, the
   reviewer's summary comment and the record in it (#226). The run still
   blocks; judged with `--no-apply`, they wait in `pending` like any
   other, for the applying job, which runs whatever the judge's outcome.
6. **Again.** If `pre` took less than there was to do, it writes `in/more`:
   the findings it defers, one `<rule> <where>` a line. When the run passed
   and applied something other than a note, the engine then runs the role
   again, from step 2, in a new run folder that sees what was applied — five
   rounds at most. The result adds up the rounds' calls and applied
   intentions; a later round's finding replaces an earlier one of the same
   rule and place, and a deferred one is dropped once the next round runs. A
   run judged with `--no-apply` applies nothing a next round could see: it
   goes round again only when, with `--open-merge-request`, the round
   proposed a merge request for a task no earlier round of the run proposed
   (the key of `out/merge-request.yaml`). The next round is told that task
   waits, as one open on the forge, and takes up what was deferred — on a
   gardening night, the docs judged in parts after those judged whole
   (ADR-0013, amended). Each round is a run folder to apply, listed in the
   result's `pending`, a merge request each.

   **A cap on tokens.** A role setting `ai-max-tokens` caps what one run may
   spend, every round, part and retry together: the tokens in, cache
   included, and out, as the agents report them. It is checked before each
   call against what was spent, never estimated, so the call that crosses it
   is paid; the agent is asked nothing more, and the run says so
   (`ai-max-tokens`, a warning): what is left waits for the next run. 0 or
   unset, no cap.

### When apply stops half-way

Writes to git and to a forge are not transactional: a comment can be posted and
the label call fail. So every intention is applied idempotently — a label that
is already there, a comment carrying the run's marker, a merge request already
open from the branch (updated, not opened again) all count as done — and `workline apply <run-dir>` resumes a
run from `out/run.yaml`, with the evidence of the original run, without calling
the agent again.

Scripts receive `WORKLINE_RUN_DIR`, `WORKLINE_EVENT`, `WORKLINE_AI` (the agent
name or `none`), `WORKLINE_ROLE`, `WORKLINE_BIN` (the engine running them,
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
`WORKLINE_PROPOSED_TASKS`. On a merge request a release tool opened (its
branch one of `release.branches`), a role that runs on `release` is run on
it: `WORKLINE_EVENT` is `release`, and `WORKLINE_RELEASE_BRANCH` names the
branch (ADR-0017). A fix judged there is not pushed onto that branch, which
the release tool rewrites: it goes to a merge request of its own into the
release's base, and the release waits until it is merged (`fixed-elsewhere`);
never over a person's commit on that branch, where it goes in a comment
instead. `workline follow` rebuilds that merge request on its base as the
base moves (ADR-0034).

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
