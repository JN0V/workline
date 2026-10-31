# ADR-0059: An issue moves by its state, each step wakes the next

- **Status:** proposed, until the maintainer merges it
- **Date:** 2026-10-31
- **Decided by:** the maintainer, on 2026-10-31: a line of states, after
  a six-persona panel critiqued their own proposal; the panel's
  recommendations followed ([research](../research/line-of-states.md))
- **Settles:** [#752](https://github.com/JN0V/workline/issues/752)
- **Builds on:** [ADR-0018], [ADR-0042], [ADR-0050], [ADR-0054],
  [ADR-0055], [ADR-0057]; principles 1, 4, 5, 10, 12, 13, 14
- **Amends, once built:**
  - [ADR-0054]: the developer is woken by Ready alone, never by the
    schedule; it accepts a proof kind in place of a Draft; past its
    caps an issue goes Blocked, not to a person;
  - [ADR-0055]: the tester is woken by `workline:needs-tests`, not by
    `workline:ready` on the schedule; an objection past three exchanges
    goes to the product owner, then Blocked; `objection-person` and its
    default `workline:human` go;
  - [ADR-0057]: three slots more, refine, tests and blocked;
  - [ADR-0018]: the product owner sets `workline:needs-tests` where the
    tester is routed, `workline:ready` where it is not;
  - [routing.md](../spec/routing.md): its states, and how a step starts
    the next ([below](#explicit-chaining-and-the-engines-own-writes)).

## Context

- **Since the build path went live (about 2026-10-29), nothing went
  end to end**: one developer pull request, red and idle 25 h
  ([#699](https://github.com/JN0V/workline/pull/699)); two tester
  Drafts ([#677](https://github.com/JN0V/workline/pull/677),
  [#713](https://github.com/JN0V/workline/pull/713)); all open.
- **25 issues carry `workline:ready`**, a median wait of about 18 days;
  some stale (one names a function deleted weeks ago), some not one
  run's work.
- **A refusal stays in a CI log**: an issue claimed by the developer
  since 2026-10-30, no pull request, no word on it.
- **The night drives the line**: the gardening cron starts hours late,
  skipped 2026-10-22 to 10-26; the tester takes one issue a night.
- **Ready wakes the developer too early**: it refuses, the tester's
  Draft not written yet ([#752](https://github.com/JN0V/workline/issues/752)).
- **Some issues need no new test**: a flaky test made reliable
  ([#619](https://github.com/JN0V/workline/issues/619)) never reaches
  the developer.

## Decision

```mermaid
flowchart LR
  refine["<b>Refine</b><br/>product owner"] --> tests["<b>Tests</b><br/>tester"]
  tests --> ready["<b>Ready</b><br/>waits for room"]
  ready --> doing["<b>Doing</b><br/>developer"]
  doing --> done["<b>Merged</b><br/>the forge closes it"]
  tests -.-> blocked["<b>Blocked</b><br/>its reason, on the board"]
  doing -.-> blocked
```

- Each arrow is one hop: the engine sets the next label and starts the
  next role at once.
- A reconciler, hourly, does what an event missed.
- The night only tidies: gardening, staleness, the curator.

### 1. Every refusal, claim, wait or block is on the issue

Built first, whatever else.

- **One sticky comment a role on the issue**, edited, never a new one a
  run: what it did, what it waits for and why, the next step
  (principle 14). A refusal in a job's log alone is a defect.
- **A claim with no pull request after `claim-hours` (24)** is released
  by the engine: its label off, the reason in the comment.

### 2. States are labels, one at a time

| State | Label | Board slot | Set by | Wakes |
|---|---|---|---|---|
| Refine | `workline:to-refine` | refine | a new issue; a stale one; the product owner | the product owner |
| Tests | `workline:needs-tests` | tests | the product owner, the need validated | the tester |
| Ready | `workline:ready` | ready | the engine, on the tester's result | the developer, when Doing has room |
| Doing | `workline:developing` | doing | the developer's claim ([ADR-0054]) | — |
| Blocked | `workline:blocked` | blocked | any role, with its reason | nobody; the reconciler re-reads it |
| Done | none | done | the forge's close at the merge (`Closes #n`) | — |

- **Exclusive**: the engine puts the new label on and takes the old one
  off, in one call, every forge. On a GitLab board with scoped column
  labels, the forge drops the old column itself ([ADR-0057]).
- **Existing labels kept**: `workline:to-refine`, `workline:ready` and
  `workline:developing` keep their names; two are new.
- **No tester routed**: the product owner goes straight to Ready, as
  today.

### 3. Tests wakes the tester; its result makes Ready

- **`workline:needs-tests`** starts the tester at once, not on the
  night's schedule.
- **Its result moves the issue to Ready**, checked by the engine with
  no agent (principle 4):
  - each Validation scenario has its row in the Draft's `## Tests`
    table;
  - the Draft passes the project's lint.
- **The product owner reads no tests by default.** It is called only
  when the tester says "no test possible", or an objection is about
  the need itself ([ADR-0055]).
- **A product owner's review of the tests** comes later, only if the
  developer's objections show it is needed (build step 5).

### 4. A proof kind, not always a new test

| Kind | When | Proof |
|---|---|---|
| `new-tests` | a behaviour to add or a bug to show | the tester's Draft, red first ([ADR-0055]) |
| `flaky-n-of-n` | a flaky test to make reliable ([#619](https://github.com/JN0V/workline/issues/619)) | the named test passes N runs of N |
| `existing-suite` | a refactor | the project's suite, unchanged and green |
| `docs-checks` | a docs change | the documentalist's checks |

- **The tester records the kind and its reason** in its sticky
  comment; the developer accepts it in place of a Draft.
- **"No test possible"** goes to the product owner, which picks a kind
  or sends the issue back to Refine.

### 5. To validate is the pull request's state, not an agent pass

- **A required check**: the frozen tests green, no agent ([ADR-0055]'s
  green proof).
- **A real try only for scenarios marked not tested**, by the tester,
  told under `## Tried` ([ADR-0043]).
- **Done is the merge**, by the project's policy ([ADR-0042]).

### 6. Disagreements end at the product owner, then Blocked

- **The product owner settles** a disagreement between roles, inside
  the existing cap of three exchanges ([ADR-0055]).
- **Past the cap**: Blocked, its reason on the issue and the board. No
  person is asked; a person on the loop reads the column.
- **Goes**: `workline:human` as a request to a person, and
  `objection-person`, whose default it is.

### 7. Staleness, per state

- **One limit per state** in `.workline/config.yaml`, counted from when
  the state's label was set: GitHub's issue timeline, GitLab's
  `resource_label_events`. Never `updated_at`, which any comment moves.
- **First a check with no agent**: the age; the Scope's paths and names
  still in the code. The product owner is woken only for what it flags.
- **Stale goes back to Refine**, with the reason.

### 8. Event-driven, the night for upkeep

- **The state machine lives in the engine**: one table, state → role →
  next state, read by a pure function, as routing is.
- **Thin forge adapters**: on GitHub one workflow on `issues: labeled`
  and `workflow_dispatch`; on GitLab one job set, started by the
  pipelines API.
- **A reconciler**, `workline route reconcile`: walks every open issue
  and runs what is overdue. Hourly in CI; a command on a person's
  machine. Events only make the line faster: its correctness never
  depends on them.
- **Idempotent**: a step re-reads the issue first, and records the
  state it acted on and the body's hash; the same record twice does
  nothing.
- **One concurrency group per issue** across every role: GitHub's
  `concurrency`, GitLab's `resource_group`.

### Explicit chaining and the engine's own writes

- **The rule stays**: what the engine writes never wakes the line by
  itself ([routing.md](../spec/routing.md#the-line)). A label set by
  the App is ignored by the state workflow; only a person's label is
  read as an event.
- **The next step is started explicitly**, by the apply job that moved
  the state: `workflow_dispatch` on GitHub, a pipeline by the API on
  GitLab, the next loop of `reconcile` on a machine. It does not depend
  on which token set the label.
- **Not a handoff**: `max-handoffs` caps handoffs inside one line run.
  A state hop is a new run. Hops are counted per issue in its record
  and capped (`max-hops`); past the cap, Blocked with the count.

### 9. Pull, and room in the next state

- **A role starts only when the next state has room**: a WIP limit per
  state in the config. A full Ready keeps the tester waiting, said on
  the issue.
- **A Definition of Ready per state**, checked by the engine:
  - to Tests: the four sections ([routing.md](../spec/routing.md#work-starts-from-a-clear-need)),
    named scenarios, Scope paths that exist, and fits one developer
    run;
  - to Ready: a proof kind, and for `new-tests` the checks of §3.

### 10. Tracked apart, not fixed by states

- **The developer's proof environment**: ten conformance cases red on
  [#699](https://github.com/JN0V/workline/pull/699), none touched by it.
- **The 25 ready issues** re-checked once by the product owner.
- **The tester's malformed answer** asked again once
  ([#753](https://github.com/JN0V/workline/pull/753)'s try).

## Build plan

One pull request a step, its cases first in `tests/conformance/cases/`,
`pending.txt` listing what does not pass.

1. **Written on the issue**: the sticky comment a role, a stale claim
   released (§1).
2. **The line**: the state table, the reconciler, explicit chaining
   and the hop cap; Tests wakes the tester, Ready the developer; proof
   kinds (§2–§6, §8).
3. **A first result**: the product owner re-checks the ready issues
   once; then two small fresh issues through the whole line, timed.
4. **The board and the flow**: the tests and blocked slots on every
   forge ([ADR-0057]); WIP limits; staleness per state (§7, §9).
5. **A product owner's review of the tests**, only if the developer's
   objections after step 3 show it is needed.

Docs to update with each step: [routing.md](../spec/routing.md) (its
states, the chaining), [boards.md](../boards.md),
[the developer's triggers](../../roles/developer/docs/triggers.md),
[the tester's hand-over](../../roles/tester/docs/hand-over.md) and
[README](../../roles/tester/README.md) (`objection-person`),
[refine](../spec/backlog-acts/refine.md); the "Amended by" lines of the
ADRs above.

## Consequences

- **Need to pull request in hours, not nights**: each hop starts the
  next; the hourly reconciler bounds the wait an event missed.
- **Tokens**: the same agent calls an issue as today, less the
  developer runs that only refused. The reconciler asks no agent.
- **A person reads the board**: Blocked is a column with a reason; no
  label asks them to act.
- **A GitLab project needs its webhook tool** for events; without it,
  the reconciler alone moves the line, hourly.
- **More labels**: two new ones, set and cleared by the engine.

## Not chosen

- **Today's line**: Ready wakes the developer, the tester writes one
  issue a night. The evidence above: nothing end to end.
- **The maintainer's six states** (to-refine, needs-tests, tests-ready,
  ready-to-dev, to-validate, done):
  - a product owner's re-read of every Draft: tokens and a hop on each
    issue, for a mismatch the engine's scenario check catches; kept as
    step 5, if objections show it;
  - a tester's agent pass at to-validate: the required green proof
    already says it, with no agent;
  - done as a label: the forge's close does it.
- **The tester woken on `workline:ready`** ([#752]'s alternative): no
  new state, but Ready means "tests to come", and the developer still
  waits.
- **A `workline:human` label for a blocked issue**: it asks a person to
  act; the person is on the loop, not in it.
- **The night as the line's engine**: a late or skipped cron stops all
  work.

[#752]: https://github.com/JN0V/workline/issues/752
[ADR-0018]: 0018-the-product-owner.md
[ADR-0042]: 0042-the-engine-merges-by-the-projects-policy.md
[ADR-0043]: 0043-a-try-is-told-in-its-pull-request.md
[ADR-0050]: 0050-each-act-has-its-moment-and-a-wait-budget.md
[ADR-0054]: 0054-the-developer-takes-a-ready-issue-to-a-pull-request.md
[ADR-0055]: 0055-the-tester-writes-the-acceptance-tests-before-the-code.md
[ADR-0057]: 0057-workline-finds-the-forges-board-and-moves-its-cards.md
