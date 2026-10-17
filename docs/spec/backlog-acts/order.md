---
type: reference
sources: [internal/backlog/backlog.go, internal/backlog/order.go, internal/scale/scale.go, internal/builtin/productowner/productowner.go, internal/engine/backlog.go]
status: draft
checked: 9fcf6cf6
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

Sets an issue's priority: one label, `workline:priority/1` (the most
pressing) to `workline:priority/4`, created when the project has none,
the other three taken off. A level outside 1 to 4 is dropped
(`priority-level`); the priority it has already, too (`priority-same`).
The issue's state records the priority set. A priority label other than
the one recorded — set by a person, or by a person over the role's, or
taken off by one — is a person's: the act is dropped and the label kept
(`priority-kept`); the task shows such a label as a person's. No quote; the state
must read, as for any act.

The levels are read by name through one table (`internal/scale`,
[ADR-0040](../../adr/0040-priority-and-effort-are-the-forges-the-product-owner-estimates-them.md)):
the two scales, Urgent to Low and an effort Low to High, and the names
each is read under — a field's options whatever their order, a
project's label family, GitLab's `weight`. The engine reads and writes
only the labels above, and no effort, until it detects where a project
keeps them.
