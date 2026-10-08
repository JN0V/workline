# ADR-0026: The product owner's autonomy is a level; ignored runs are a setting

- **Status:** accepted; partly superseded by [ADR-0038](0038-the-product-owner-proposes-on-the-issue-a-person-answers-there.md)
  (one autonomy axis)
- **Date:** 2026-10-05
- **Decided by:** the project's author, after a round table (analyst,
  product manager, architect, developer, UX designer, test architect)
- **Builds on:** ADR-0018 (autonomy per kind of act; trust earned and
  lost), ADR-0022, ADR-0024, ADR-0025 (back to act; three ignored runs
  pause); principles 1, 4, 12, 14

## Context

Each kind of act has a mode and a cap, eleven kinds in all, so a project
has to set many numbers to say how far the role may go. A project with a
human Product Owner has to switch every kind to `propose`, including the
ones that only check facts. ADR-0025 fixed the pause after three ignored
runs, and that number cannot be changed. Settings merged one level deep
(#100): a project that set one kind lost the others, which fell to
`propose` without saying so; since #153 they merge field by field, at
every depth, which a preset needs.

## Decision

One setting, `autonomy`: `cautious`, `normal` (the default) or
`enterprising`. It is a preset of the acts' modes and caps
(docs/spec/backlog-acts.md, "Autonomy and caps"), not a new mechanism:
the role ships it in its `role.yaml`, as `levels`, laid over its defaults
before the project's settings (docs/spec/role-adapting.md). `normal` is
the defaults as they were. `cautious` keeps acts that check facts
(sources, asking the reporter, announcing what is obsolete, Scope and
Verification from the code) and proposes the acts that set direction:
Need and Validation drafts (`refine: {drafts: propose}`), splits,
renames, milestones, priorities, duplicates. `enterprising` raises the
caps and the moved share. At every level, `ready` stays the engine's
check: Need and Validation are a person's.

The level sets what is done with what is read, not how much is read.
`issues-per-run` and `code-lines-max` stay their own settings, the token
budget.

Order of precedence: the level, then an explicit setting of a kind,
merged field by field; then a kind demoted to `propose`, which only a
person's tick lifts (ADR-0025), whatever the level; then a person's tick
on one act. The task and the report say each kind's mode and where it
comes from: `level`, `setting` or `demoted`.

Undoing an act of any kind demotes that kind, found at the next run from
what the role's record says it set: a title, a priority or a milestone a
person put back as it was, `ready` taken off, a split's child closed as
not planned — as a closing reopened does.

`ignored-runs-max` (3; 1 to 20; 0 never pauses, said in every report)
replaces ADR-0025's fixed three. It does not depend on the level.

Not decided now: `enterprising` does not move drafts to `ready` on
silence — to reconsider when 90% of 20 drafts or more are accepted
unchanged, and never for an outsider's issue. `cautious` still announces
what is obsolete itself, with 14 days' notice.

## Consequences

- A project says how far the role goes in one word, and still sets any
  kind on its own.
- Changing the level never restores trust the role lost.
- `workline init` asks whether a person is the Product Owner, and sets
  `cautious` if so.
- The record keeps the level with each act; the report suggests another
  level from what it measures — at `cautious`, more than 80% of the
  proposals settled ticked as proposed suggests `normal` — and never
  changes the setting. The weekly sample over these acts, and its
  suggestion from the acts undone, are ADR-0033.
