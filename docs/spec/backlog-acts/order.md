---
type: reference
sources: [internal/backlog/backlog.go, internal/backlog/order.go, internal/scale/scale.go, internal/scale/detect.go, internal/builtin/productowner/productowner.go, internal/engine/backlog.go]
status: draft
checked: 432aeef8
verified: agent:claude-code
---
# Milestones and ordering

Part of [Acts on the backlog](../backlog-acts.md): putting an issue in a milestone, and setting its priority.

## Milestones

```yaml
- milestone: {issue: 12, milestone: "v2.14.0", why: "the next release's fix"}
```

Puts an open issue in the milestone of a release, created when none with
that title is open. Ordering says nothing of an issue's truth: no quote,
but its state must read as for any act. The task gives the last release
tag and the milestones open, in the milestones' order
([The order, and what waits](next.md#the-order-and-what-waits)), each with its due date
when it has one; each issue says its own.

**What slipped** is moved by the engine, with or without an agent: an
open issue whose milestone is named after a tag that exists is put in
the nearest open milestone not released, in the milestones' order;
with none left, the move is proposed on the issue,
the issue left where it is. The engine's move comes first: an agent's
milestone for the same issue in the same run is dropped.

## Ordering

```yaml
- order: {issue: 12, priority: 1, why: "it loses rows from every report"}
```

Sets an issue's priority: one label of the family the project keeps it
in (below), by default `workline:priority/1` (the most pressing) to
`workline:priority/4`, created when the project has none, the family's
others taken off. A level outside 1 to 4 is dropped (`priority-level`);
the priority it has already, too (`priority-same`). The issue's state
records the priority set, as its level. A priority label other than the
one recorded — set by a person, or by a person over the role's, or
taken off by one — is a person's: the act is dropped and the label kept
(`priority-kept`); the task shows such a label as a person's. No quote;
the state must read, as for any act.

The levels are read by name through one table (`internal/scale`,
[ADR-0040](../../adr/0040-priority-and-effort-are-the-forges-the-product-owner-estimates-them.md)):
the two scales, Urgent to Low and an effort Low to High, and the names
each is read under — a field's options whatever their order, a
project's label family, GitLab's `weight`.

### Where the project keeps them

Each axis has a setting of the role, `priority` and `effort`
(`internal/scale/detect.go`, `backlog.Where`):

- **Unset**, the default: the engine reads and writes workline's own
  labels above, and no effort; nothing of the forge is read for it.
- **`auto`**: detected each run, with no agent, the first that holds — the
  project's own family of labels, two values or more and one on an open
  issue, tried in this order: `P0`…, `P-*`, `priority/*`, `Priority/*`,
  `priority::*` (effort: `effort1`…, `effort/*`, `effort::*`); then
  workline's own labels, one on an open issue; else the forge's words.
- **`{in: labels, prefix: P}`**: that family; one the project lacks
  stops the run, said (`values-where`, block).
- **`names`**, under either: a label of the family no table knows mapped
  to a level, `{names: {sev-1: urgent}}`. One not mapped carries none,
  said once in the run (`level-unknown`).

The project's labels are listed from the forge (a plugged forge that
cannot list them: the labels on its open issues), so a family's numbers
count from its own first label, `P0` though on no issue. **The forge's
words**: `priority/urgent` to `priority/low` and `effort/low` to
`effort/high`, the names of GitHub's Priority and Effort fields'
options; on GitLab, `priority::1` to `::4`, ranked 1 to 4 as prioritized
labels, and `effort::low` to `::high`. All of an axis's words are
created at the first act writing one; a family of the project's is
never added to. The run says once where each axis was found
(`values-where`, info): "Priority: labels P0 to P3, the project's.
Effort: labels effort::*, created when first set."

**One place an axis**: an issue holding a value of that axis in another
family — alone, or beside one of the family read — gets nothing written
(`priority-elsewhere`): never two values, nor a copy. A level no label of
a project's family carries is dropped (`priority-no-label`). With another
family than workline's own read, the role's texts say the label beside
the level: "priority 1 (P0)".
