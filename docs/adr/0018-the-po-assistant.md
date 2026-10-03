# ADR-0018: A PO assistant keeps the backlog, between the need and its acceptance

- **Status:** proposed — the points under "Open" wait for the person
- **Date:** 2026-10-03
- **Builds on:** ADR-0014 (`checked` earned by what was read), ADR-0015
  (the weekly sample), ADR-0016 (writes go to the project's forge),
  docs/spec/routing.md ("Work starts from a clear need")

## Context

Work state belongs in the forge's issues, not in files (docs/BACKLOG.md).
DomoticsCore shows the alternative: a roadmap file of 7,570 lines, 190
entries, its state split between a table and the headings, written by every
session. The person asked for a role that does a Product Owner's work on
the backlog, and for borrowing what exists before building.

docs/research/po-assistant.md found that the forges already carry a backlog
(forms, milestones, sub-issues, close reasons, labels), that bots already
keep one report issue in place (Renovate) and cap an agent's writes
(GitHub's *safe outputs*), and that no tool re-checks an issue **because
the code it names changed**: every "obsolete" in use is inactivity, or a
reporter's silence taken for consent.

A first draft, reviewed by a panel of five (product, analysis,
architecture, test, UX: the BMAD agents) over three rounds, left the role
proposing everything and doing nothing: no closing, no milestones, no
refining. The person rejected it: a role whose every act a person must
repeat adds nothing. The line follows principle 1 instead — people state
the need and accept the result, the machine does the work in between —
and the Scrum Guide's split: a product manager owns why and what (needs,
market, what others do), a Product Owner owns how and when (the backlog:
its items, their order, refining them, milestones and releases).

## Decision

### The role and its freedom

**A role, the PO assistant**, of its own, sharing the documentalist's
suspect-and-quote core (duplicated where needed, extracted at a third use).
A product manager assistant, later, brings needs in; this one turns them
into the backlog.

**It acts alone**, through the engine, on what a Product Owner does:

- opens issues, and splits a need into technical tasks (sub-issues);
- refines: writes an issue's Scope and Verification, finds the code it
  concerns and names it as its sources, asks the reporter what is missing;
- moves an issue to `ready` once its four fields are there
  (docs/spec/routing.md lets a product role do so); `ready` is a state,
  never a gate on other work;
- orders the backlog; creates the milestones of the next releases, fills
  them, moves what slipped;
- closes a duplicate, linking the original; closes an issue the code made
  obsolete, the code quoted, with a comment inviting an answer if wrong —
  an answer has it judged again.

**It leaves to the person** what principle 1 gives them: stating a need,
and accepting a result. So it never closes as *not planned* — refusing a
need is a product decision — never writes an issue's Need or Validation
as settled (it may draft them, marked as drafts), and never deletes.

A fix merged elsewhere than the branch a release is cut from is work in
progress: the issue stays open, the report says where the fix is.

### What makes that freedom safe

**Evidence**: no act on an issue's truth — obsolete, duplicate, its
sources — without a quote the engine finds again in the code or the
issues (as ADR-0014's claims). Without one, it does nothing: "when in
doubt, skip".

**Re-checked when its sources changed** since it was last confirmed, never
because it is old; sources name symbols or line ranges where they can, so
a busy file does not make every issue on it suspect; at most once in a
window (a week by default), in gardening.

**What it knows of an issue lives in one comment the engine keeps**
(`Sticky`): its sources, the commit it was last confirmed at. The body gets
one readable line. A comment missing or broken: the issue is never judged,
nothing is written on it.

**One report issue**, kept in place (`KeepIssue`), lists what it did and
what it proposes, each with its quote and how to undo it; one comment,
"N new", so it notifies. A box ticked there is read only from someone with
write access.

**Caps** per run and per kind of act. **Autonomy is a setting per kind of
act** (act, propose, off), as Linear's per property; the default is the
list above.

**Trust is earned and lost**, as `checked` is: the weekly sample reads its
acts (ADR-0015); one closing found wrong puts that kind of act back to
"propose" until the person sets it again. Proposed, done and undone are
counted; if its report is ignored for three runs, it pauses and says so.

### Opening issues, for every role

Every role that needs one opens an issue through one engine mechanism, set
now: no duplicate (a stable key per role and subject), labelled
`needs-triage`, its sources and the commit it was seen at in the engine's
comment, a cap per run; no role but the PO assistant closes. The
documentalist's issues and the reviewer's go through it; the PO assistant
takes them from there.

### Borrowed, built, proved

**Borrowed**: issue forms for the four fields, milestones, sub-issues,
close reasons, labels, `KeepIssue`, `Sticky`. **To build**: listing a
project's open issues on every forge; milestones and closing with a reason
in the forge interface; "its sources changed since it was confirmed" for an
issue; the judgement with quotes; reading a tick with its author.

**Proof** before calling it done: conformance cases (each act's cap; no
quote, no act; a broken comment writes nothing; never *not planned*; a
wrong closing found drops that act to propose); an evaluation on planted
issues — truly obsolete, duplicates, and true ones that look obsolete —
five runs or more; the weekly sample over its acts. The first measure is
wrong closings, then wrong milestones and orders.

**First real work: DomoticsCore's roadmap**, its critical, high and medium
entries closed, the low ones left: sorted, opened as issues keeping their
old ids, grouped into milestones, the file then kept read-only — first on
a copy, applying nothing.

## Open — for the person

1. The pause and the drop: after how many ignored runs, and whether one
   wrong closing is the right trigger.
2. When the product manager assistant is written up.

## Consequences

- The backlog is kept by the machine between a need and its acceptance;
  the person states needs, accepts results, and can undo any act.
- Every role's issues share one way in; their lifecycle is the PO
  assistant's.
- A roadmap file goes, once its entries are issues.
