# ADR-0029: A parent is accepted by a person, from what its parts delivered

- **Status:** accepted
- **Date:** 2026-10-05
- **Builds on:** ADR-0022 (a split keeps the need in its parent), ADR-0026
  (a kind of act has a mode per level), ADR-0028 (`next-ready`);
  principles 1, 4, 5, 14
- **Settles:** #162

## Context

A split (ADR-0022) keeps the need in its parent and opens 2 to 6 parts,
each refined, built and closed on its own. Nothing then looks at the
parent: it stays open with every part closed, or is closed by hand on a
count of ticks, a part of the need lost between them. ADR-0022 left "a
parent's own readiness while its children are open" unbuilt. This adds
two forge reads and a report, so it is a record of its own rather than an
amendment. docs/research/product-owner.md ("A parent and its parts")
found that GitHub and GitLab show a parent's progress and never close it,
that Jira's common automation closes it when its last part is done, and
that both forges link what closed an issue: a pull or merge request, a
commit.

## Decision

### Readiness

`ready` on a parent means what it means on any issue: its four sections
there, its need understood. The role refines it as any issue. But a
parent is **never offered to build** (`next-ready`): its parts are what
is built. No new label, no new check.

### What its parts delivered

Each run, with or without an agent, paused or not, the engine writes one
comment on every open parent (`<!-- workline:sticky=<role>/parts -->`),
edited in place, never written again when nothing changed:

- **Its parts**: the forge's own relation (GitHub's sub-issues, GitLab's
  tasks through GraphQL), the task list under `## Sub-issues` in its body,
  and the parts its state records (`split`).
- **Each part**: open (ready or not), closed as completed — and what
  closed it, from the forge (GitHub's `ClosedEvent.closer`, GitLab's
  `closed_by` merge requests and "closed via commit" notes) —, or closed
  without delivering: not planned, a duplicate, gone from the forge.
- **Each item of its Verification** (each list item, or the section
  whole): proved when a part delivered quotes it, in its own Verification
  or in the text of what closed it, case, spaces, Markdown marks and the
  final punctuation aside; "not proved" otherwise. A quote is mechanical
  and checkable; a paraphrase is not proof.

### Acceptance is a person's

When every part is closed, the comment asks a person to accept the need
by **closing the parent**, naming first what is missing: parts not
delivered, items not proved. A finding (`parent-to-accept`) and the
report's "To accept" say it too. The role never closes a parent nor sets
a label on it: an agent's closing of one is dropped
(`parent-accepted-by-a-person`), announced as obsolete included.

No agent act is added. Judging that a paraphrase proves an item would
weaken what the person reads; the mechanical list already names every
item not proved, and a person judges those.

## Consequences

- A person accepts or reopens a split need from one comment, without
  opening each part.
- Per run: one listing of every issue when an open parent exists, one
  call per part delivered (GitHub), two (GitLab). GitHub lists sub-issues
  only for parents its listing counts; GitLab one GraphQL query a page.
- A part in another repository (GitHub allows it) is left out.
- Not decided now: an agent drafting which items look proved, if live
  use shows parents whose items are proved in other words; a test named
  as proof, read from the code.
