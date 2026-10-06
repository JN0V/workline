# ADR-0033: The weekly sample draws the product owner's acts, for a person

- **Status:** accepted
- **Date:** 2026-10-06
- **Builds on:** ADR-0015 (the weekly sample, written on the forge),
  ADR-0018 (the product owner; trust earned and lost), ADR-0026 (autonomy
  levels; an act undone demotes its kind), principles 1, 4, 5, 13

## Context

The product owner acts alone on the backlog at the level the project chose
(ADR-0026). A person sees each act once, in the report of the run that did
it, which the next run rewrites. Nobody is shown, later and at a steady
pace, a share of what it did alone to judge; and the only suggestion of
another level comes from the proposals a person settled at `cautious`.
ADR-0026 left "the weekly sample over these acts" to build (#186).

The weekly sample exists for the documentalist (ADR-0015): `workline
sample` draws one in ten of a week's docs, the same on a rerun, and
`workline sample --apply`, in a job holding the forge's token and no AI
key, writes a tracking issue, a comment a week. The record on the product
owner's report kept each closing and each act a person may undo, with its
level, but not its day, nor acts like a refine or a question to a
reporter, and it forgets an act once its issue is closed.

## Decision

**Borrowed, not built again**: the same command, week, draw (one in ten,
rounded up, ranked by a digest of the week and the act) and write job.
Drawing the acts needs the forge and no AI, so `--apply` does it, after
the docs: the read job, which holds the agent's key, never reads the
backlog. A project with no product owner's report gets no acts sample.

**The record keeps the acts done alone** (`did`): each act a run decided to
do alone — its kind, its issue, the level, the day, a line saying it — for
35 days, at most 200. A person's tick and the engine's own move of a
slipped milestone are not the role's choice: not kept, as `done` already
does not watch them.

**For a person to judge**: the sample writes its own tracking issue, "workline:
the weekly sample of the product owner's acts", a comment a week edited in
place on a rerun: each act drawn with its kind, issue, day, level, and
whether a person undid it since — a closing reopened, or what the record's
`undone` says. The person's verdict is the one ADR-0026 already reads: an
act found wrong is undone on its issue, and the next run demotes that kind.
No box to tick in the sample: a second way to say "wrong" would have to be
read, checked for who ticked it, and reconciled with the undo.

**A level suggested from the acts**: over the acts done alone at the level
in force that the record keeps, up to the week's end, with at least 10 of
them: more than one in ten undone suggests the level below (`enterprising`
→ `normal` → `cautious`); none undone at `normal` suggests `enterprising`.
At `cautious` the role does alone only what checks facts: the report's
suggestion, from the proposals settled, stays the one. The sample never
changes the setting.

## Consequences

- A person sees a steady share of what the role did alone, with the level,
  and is told when the acts undone say the level is too high or too low.
- Undoing stays the one verdict on an act; the suggestion is only as good
  as people's undoing. An act right but never looked at counts as
  standing.
- The record grows by one line an act for five weeks: about 200 short
  lines at most, in one comment.
- Left: the suggestion written in the role's own report too; a person's
  tick in the sample read as a verdict, if undoing proves too coarse; a
  GitLab project's acts sampled live.
