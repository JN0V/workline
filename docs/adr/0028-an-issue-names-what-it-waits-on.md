# ADR-0028: An issue names what it waits on; the order never offers it first

- **Status:** accepted
- **Date:** 2026-10-05
- **Builds on:** ADR-0018 (the backlog's order, derived, never stored),
  ADR-0022 (a split's children), ADR-0026 (a kind of act has a mode and a
  cap at each level; undoing an act demotes its kind); principles 1, 4, 6,
  12, 13
- **Settles:** #161

## Context

The backlog's order (`backlog.Less`: milestone, priority, number) knows
nothing of what an issue waits on: an issue that cannot start before
another is done ranks by its labels alone, and may come first to whoever
builds next. The product owner sees such links when it refines or splits a
need, and writes them nowhere a machine reads. docs/research/product-owner.md
("What an issue waits on") found the relation native on GitHub (issue
dependencies, every plan) and on GitLab Premium (`is_blocked_by`), refused
on GitLab Free (403, "not available for current license"), absent from the
local forge and from a plugged one; and no tracker that keeps a blocked
issue out of first place by itself.

## Decision

### Where it is kept

The forge's own relation where it has one: GitHub's
dependencies, GitLab's `is_blocked_by` link. Elsewhere — GitLab Free, the
local forge, a plugged forge that answers `{native: false}` — a line in the
issue's body, `Blocked by #12, #13.`, ending with the hidden marker
`<!-- workline:blocked-by -->`, one line the engine rewrites as it adds a
blocker. Every forge reads back both: its relation with the open issues,
and any line of a body that starts with "Blocked by" followed by issue
references — a person's own line counts, as a person's link does.

### Who sets it

A new kind of act, `depend` (`{issue, blocked-by: [n],
why}`), proposed by the product owner when it reads an issue, and a
split's child naming the siblings it waits on (`after: [1]`, by their
place in the split). It is a kind of its own, not part of `refine`: it
changes the order, not the text, and a project may want one without the
other. Checked as any act — the issue and each blocker open, not the
issue itself, 1 to 5 blockers, one not there already (`depend-same`), no
cycle with the relations there and those decided in the run
(`depend-cycle`); its state must read; no quote. Its mode: `propose` at
`cautious` (it sets direction), `act` 5 a run at `normal`, 10 at
`enterprising`. It does not count in the moved share: it changes no
priority nor milestone, and the issue it holds back is still where its
labels put it once its blockers close. The role only adds: a relation
another set, a person's above all, is never removed; one the role set and
a person took off, its blocker still open, is undone, and demotes the kind
(ADR-0026).

### The order

`backlog.Order` is Kahn's: the next issue is the first, in
`Less`'s order, whose open blockers are all placed. A blocker closed, or
not in the list, holds nothing back: once it closes, the issue is ordered
by its labels again. When none can be placed, the issues left hold a
cycle: it is reported (`dependency-cycle`, the task, the report) and the
first of the cycle by `Less` is placed — never an issue that only waits on it —, so the order ends; a cycle is never
followed. `ready` is allowed while blocked — the four sections say the
issue is understood, not that it can start — but the first ready issue
offered (`next-ready`, the report's **Next**, #117's developer role) is
never one with an open blocker. The product owner's task marks each issue
with what it waits on; the report lists the issues waiting, and the
cycles.

## Consequences

- An issue's dependency is visible where people look: the forge's own
  "Blocked" mark where it has one, a line in the body elsewhere.
- One more listing call a run on GitHub only for the issues it says are
  blocked; one GraphQL query a run on GitLab, which an instance without it
  answers with an error, read as no native relation.
- A project that moves from GitLab Free to Premium keeps its body lines,
  read as before; new relations go native.
- Not decided now: removing a relation the role set once its reason is
  gone; a dependency across projects (both forges allow it; the order
  reads one project).
