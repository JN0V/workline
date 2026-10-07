# ADR-0025: A person's tick in the report is done; runs nobody answers pause the role

- **Status:** accepted
- **Date:** 2026-10-05
- **Builds on:** ADR-0018 ("a box ticked there is read only from someone
  with write access"; "only the person sets it to act again"; "after
  three, it pauses and says so"), ADR-0021 and ADR-0023 (who is of the
  project, on each forge); principles 1, 4, 6, 12, 14
- **Amended by:** ADR-0026 — the three is a setting, `ignored-runs-max`;
  undoing any act of the role, not only a closing, demotes its kind

## Context

ADR-0018 gave the person three hands on the product owner and left them
unbuilt: a box ticked in the report, a kind of act set back to `act`
after a wrong closing, and a pause when nobody answers. Until now a
proposal was a line to do by hand, a kind back to `propose` stayed there
for good, and the role kept asking its agent whether anyone read the
report or not. ADR-0018 is over its length budget; this says how, as a
record of its own.

docs/research/product-owner.md ("A tick and who ticked it") found that
Renovate's dashboard applies a box ticked at its next run and checks
nobody's identity — the forge's edit rights are its only guard; that
GitHub keeps no event for a tick but keeps every version of a body with
its editor, and that GitLab writes a system note per box with its author;
and that Dependabot pauses a repository where nobody touched its pull
requests for 90 days, resuming on the first act of a person.

## Decision

### A tick is a yes, from a person of the project

Each proposal in the report is a box ending with a hidden key
(`<!-- workline:proposal=<issue>/<kind> -->`), and the report's record
keeps, with the line, **the act as the engine decided it** — so a tick
needs no agent to be done (principles 4, 5).

At the next run, the engine reads the boxes ticked in the report's body
and asks the forge who ticked each — GitHub: the body's edit history
(`userContentEdits`), the tick given to the editor of the version that
made it, of the project when GitHub gives them write, maintain or admin;
GitLab: the system note "marked the checklist item … as completed", its
author of the project from the Planner role; the local forge: whoever
works in the clone, as its comments; a plugged forge: its `ticks`
operation.

- **A person of the project's tick** is done as the record holds it,
  whatever the kind's mode, a kind back to propose or a cap: a person
  decided it, not the role. Checked again as any act (the quote found
  again, the state readable); what no longer holds is dropped and said. A
  closing as obsolete ticked is closed at once — the person's yes stands
  for the announcement and the second judge (ADR-0024). The act never
  comes from the intention: one claiming a tick the forge does not show
  is dropped (`not-ticked`), as a reply's agreement is checked again.
- **Anyone else's tick is not a yes**, and said, the box unticked when
  the report is rewritten: an outsider (`tick-ignored`), a bot — even one
  with write access, since a token's run may tick what it wrote — and a
  tick **the forge does not say the author of**: conservative, as a
  reply with no known author agrees to nothing (ADR-0021).
- **Done once**: done or dropped, the proposal leaves the record; a box
  still ticked whose proposal the record no longer holds is nothing.
- What the engine does not do — an issue to open (the import does), a
  conversation whose rounds are spent, a slip with no milestone to go to —
  is said when ticked: to do by hand.

### Back to act: a box, never the settings

A kind of act back to `propose` after a wrong closing gets a box in the
report, the measure beside it (its closings still closed, those
reopened). A person of the project's tick sets it back to `act`
(`back-to-act`). The settings are never changed: they say what the person
set, the record says what the role earned and lost.

### Ignored runs pause the role

A run is ignored when it read issues with an agent while its report held
a proposal, and no person did anything since the last run: no box ticked
by a person of the project, no comment of theirs on the report, no
closing undone, no proposal settled by its issue closed. After **three**
in a row (ADR-0018's number), the role **pauses**: no agent is asked —
nothing read, no second judge — until a person does one of those; the
report says so, with a box to resume. The engine's own work goes on: a
state comment, a slip, a reply's agreement or a label's acceptance, a box
ticked. A run with no agent never counts: it costs nothing. (The three
is a setting since ADR-0026: `ignored-runs-max`.)

## Consequences

- The report is where a person decides: a tick, at the next run, by the
  engine, with no agent; who ticked it named in the report and on the
  issue closed.
- The record grows by each proposal's content.
- A backlog nobody reads stops costing tokens after three runs.
- Not on every forge: one that does not say who ticked takes no tick.

## Amendment (2026-10-07): a tick is never handed back

On workline's own report, three refine proposals made on 2026-10-04 —
before this decision was built, by an engine whose record kept a
proposal's line alone — could not be done when ticked: the report told
the person to do the work themselves. A person's tick is their yes; the
work stays the role's.

- **A proposal recorded without its act** has its issue read again first,
  as one proposed for a cap: the agent decides it anew.
- **Ticked**, the agent is asked to write that act; what it drafts is done
  as the person's yes, whatever the kind's mode or cap. Until a run reads
  the issue with an agent, the record keeps who ticked it (`agreed`) and
  the report says so in a line, no box to tick again.
- The agent drafting none is said (`tick-not-drafted`); the line leaves
  the report.

