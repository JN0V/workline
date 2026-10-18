---
type: reference
sources: [internal/backlog/backlog.go, internal/backlog/order.go, internal/backlog/undo.go, internal/scale/scale.go, internal/scale/detect.go, internal/builtin/productowner/productowner.go, internal/engine/backlog.go, internal/forge/github_fields.go]
status: draft
checked: 698042ac
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
  forge's own single-select issue field named as the axis, `Priority` or
  `Effort` (GitHub's, an organisation's, listed by GraphQL's
  `repository.issueFields`: none on a user's repository); the project's
  own family of labels, two values or more and one on an open issue,
  tried in this order: `P0`…, `P-*`, `priority/*`, `Priority/*`,
  `priority::*` (effort: `effort1`…, `effort/*`, `effort::*`); then
  workline's own labels, one on an open issue; else the forge's words.
- **`{in: labels, prefix: P}`**, **`{in: field, name: Severity}`**: that
  family, or that field; one the project lacks stops the run, said
  (`values-where`, block).
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
created at the first act writing one. A family made only of those words
(`priority/high` and `priority/low`) is read as them, its missing words
created too; any other family of the project's is never added to. The
run says once where each axis was found
(`values-where`, info): "Priority: labels P0 to P3, the project's.
Effort: labels effort::*, created when first set."

**One place an axis**: an issue holding a value of that axis in another
family — alone, or beside one of the family read — gets nothing written
(`priority-elsewhere`): never two values, nor a copy. A level no label of
a project's family carries is dropped (`priority-no-label`). Both are
checked when the plan is decided, and again when applied — the applying
job finds where the axis is kept anew and reads the issue again —; one
found then is said, "(found when applying)". With another
family than workline's own read, the role's texts say the label beside
the level: "priority 1 (P0)".

### In GitHub's fields

- **Read and written by the option's name**, never its place: an issue's
  values come with it (`issue_field_values`, with no API version
  header), and are written by REST
  (`POST …/issues/{n}/issue-field-values`, `X-GitHub-Api-Version:
  2026-03-10`). No label is created; a label of the axis on an issue, or
  a value in the field beside the labels read, is another place
  (`priority-elsewhere`).
- **Visible to organization members only, on a public repository** (or a
  visibility not read): the values are never shown — not in the role's
  comment, nor in its state (`priority: -1`, set by the role, value
  hidden), nor in the job's summary (Next keeps the order and says
  "Values hidden", `values-hidden`; no Urgent is listed apart). Such an
  act is never proposed (`priority-hidden`, `effort-hidden`): a proposal
  shows its value; nor watched for a person's undoing: its record would.
- **Never read as empty**: an outsider's token reads those values as
  none. A token not shown yet to read them — the job's own
  (`GITHUB_TOKEN`), an App's, a person who is no member of the
  organisation (GraphQL's `viewerIsAMember`) — writes nothing in them
  (`priority-blind`, `effort-blind`), said where each axis is found.

### The effort, set once

```yaml
- order: {issue: 12, priority: 2, effort: 1, why: "one loop to fix, every report loses rows"}
```

With either setting set, the agent is told (the task's "Priority and
effort"): each value is set once, only where the issue has none; an
effort is 1 Low (hours), 2 Medium (days) or 3 High (weeks), written in
the issue's order, the priority left out when it has one.

- **Two acts**: the engine splits an order carrying an effort into an
  `order` and an `effort` (`splitEffort`), each with its own mode, cap,
  undo and record in the issue's state (`effort: 1`). The kind `effort`
  is the engine's, never in the agent's output contract; it is `off` by
  default, `max` 10 a run, 20 at `enterprising`, and counts toward
  `moved-percent-max`.
- **Dropped, said**: a level outside 1 to 3 (`effort-level`); the setting
  `effort` unset (`effort-nowhere`); an issue whose code the run did not
  give the agent, all the files it names (`effort-unread`, `pre` lists
  them in `in/code-read`); a value elsewhere or a level no label carries
  (`effort-elsewhere`, `effort-no-label`).
- **Never written over**: an effort the role did not set is a person's
  (`effort-kept`); one it set is never estimated again, changed or
  cleared since included (`effort-once`). With a setting set, the same
  holds for a priority the role set (`priority-once`); a person's change
  of either counts as an act undone (ADR-0038).
- **Read again when applied**: a value the issue gained since the plan
  read it is a person's (`priority-changed`, `effort-changed`).
- **Said apart**: each Urgent the role set (`urgent-set`), for a person
  to check at once; past 10 values the role set on an axis, one over 60%
  of them (`estimates-stuck`): "Effort: Medium on 10 of the 11 issues the
  role estimated."
