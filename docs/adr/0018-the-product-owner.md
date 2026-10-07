# ADR-0018: A product owner keeps the backlog, between the need and its acceptance

- **Status:** proposed — the points under "Open" wait for the person;
  closing as obsolete built by ADR-0024, its default `act`; a tick, a
  kind set back to act and the pause built by ADR-0025
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

docs/research/product-owner.md found that the forges already carry a backlog
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
The panel then attacked that freedom; what it found is in the guards
below.

## Decision

### The role and its freedom

**A role, the product owner** (`product-owner`), of its own, sharing the
documentalist's suspect-and-quote core (duplicated where needed, extracted
at a third use). Named after the job, as the documentalist is, not "PO
assistant": it does a Product Owner's work, which nobody else will do
beside it; its limits are principle 1 and the guards below, not its name.
A project with a human Product Owner sets its acts to "propose". A product
manager role (`product-manager`), later, brings needs in; this one turns
them into the backlog.

**It acts alone**, through the engine, on what a Product Owner does:

- opens issues, and splits a need into technical tasks (sub-issues);
- refines: writes an issue's Scope and Verification, finds the code it
  concerns and names it as its sources, asks the reporter what is missing;
- moves an issue to `ready` once its four fields are there and its Need
  and Validation are the person's — written or accepted by them, not
  drafts (docs/spec/routing.md lets a product role do so);
- orders the backlog; creates the milestones of the next releases, fills
  them, moves what slipped;
- answers a reporter, removes the labels it set itself, merges two issues
  into one;
- closes a duplicate, linking the original, and an issue the code made
  obsolete (below); reopens its own closing when someone answers it.

### What stays the person's

**It leaves to the person** what principle 1 gives them: stating a need,
and accepting a result. So it never closes as *not planned* — refusing a
need is a product decision — never writes an issue's Need or Validation
as settled (it may draft them, marked as drafts), and never deletes.

A fix merged elsewhere than the branch a release is cut from is work in
progress: the issue stays open, the report says where the fix is.

An issue opened by someone outside the project is theirs: the first time
it is split, renamed or moved to `ready`, that is proposed to them in a
comment, not done.

**Closing as obsolete is the costliest act**: a quote proves the text is
there, not that the issue is solved (the panel, unanimous). So it is
announced first — a comment on the issue, the code quoted, the commit that
changed it named — and done at the next run if nobody answered and a
second, independent judge agreed (another model, as ADR-0005); a closing
for a test now passing, or for the code named gone, needs no second
judge. A duplicate on the same sources and claim is closed alone; one by
likeness only is proposed.

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

**Caps** per run and per kind of act — closings low at first (three),
raised as trust is earned — and on the share of the backlog moved in one
run (a fifth, `moved-percent-max`: milestones and priorities together, an
issue moved twice counted once); the order before a run — each moved
issue's priority and milestone — is written in the report, to put it
back. **Autonomy is a setting per kind of act** (act, propose, off), as
Linear's per property; the default is the list above, but closing as
obsolete, which starts at "propose" until the evaluation has measured how
often it is wrong.

**Trust is earned and lost**, as `checked` is. A closing is wrong when a
person with write access, or the reporter, reopens it, or when the weekly
sample (ADR-0015) finds it wrong; one wrong closing puts that kind of act
back to "propose", and only the person sets it to "act" again, the measure
shown in the report. A run is ignored when its report proposed something
and nobody ticked, answered or reopened anything; after three, it pauses
and says so.

### Ordering

*Amended 2026-10-04* (docs/research/product-owner.md, "Priority and
ranking").

**Stored in the forge's own fields**: an issue's release in its milestone,
its priority in one label of four, `workline:priority/1` (the most
pressing) to `/4` — as Kubernetes', Rust's and Linear's levels; one at a
time, as GitLab's scoped labels. The order is derived, not stored:
**the nearest milestone first** (titles in version order, an issue in
none last), **then the priority** (none after 4), **then the lowest
number**. Any role reads it the same way (`backlog.Less`), the developer
taking the first ready issue in it. A forge's native rank (GitLab's
reorder, a GitHub project's position) is deferred: not on every forge, nor
writable with a role's token.

**A person's priority is kept**: the issue's state records the priority
the role last set; a label other than that one is a person's, and the
role's act on it is dropped (`priority-kept`).

**What slipped moves without AI**: an open issue in a milestone named
after a tag that exists is moved by the engine to the nearest open
milestone not released, or proposed in the report when there is none —
a check, not a judgement (principle 4), as GitLab rolls issues over to the
next iteration.

### Opening issues, for every role

*Amended 2026-10-04* (docs/research/product-owner.md, "A subject found
again"): the way in, made precise when the reviewer became its second user.
An amendment, not a new record: the decision — one way in, the product
owner's from there — stands; this says what it guarantees.

Every role that needs one opens an issue through one engine mechanism
(`backlog.Openings`), mechanical, no AI in it:

- **A key per subject, made by the engine**, never by an agent, hidden in
  the issue's body (`<!-- workline:issue=<key> -->`), as Sentry's
  fingerprint or code scanning's: a finding on code quotes its line (`at`),
  which the engine finds again — the key is the file and that line as it
  reads, spaces aside; without one, the title. **Two roles finding the
  same line get the same key**: one issue.
- **Looked for in the issues open and closed**, read once a run. Open: left
  as it is, nothing written. Closed as *not planned* or as a duplicate: a
  person's no, as Renovate leaves a pull request a person closed and
  SonarQube an issue *accepted* or *false positive* — nothing written.
  Closed otherwise (done, or a forge that keeps no reason): the subject is
  back, perhaps a regression, which Sentry and SonarQube reopen; here it is
  **said once on the closed issue**, with the commit, and it stays closed —
  reopening is the person's, and no role closes nor reopens another's
  issue. **Something new** is a different key: the line changed, a new
  subject, a new issue.
- **The role named**, in a line and a hidden `opened-by` marker; labelled
  `needs-triage`; the product owner's state comment, its sources and the
  commit it was seen at.
- **A cap per run** (`issues-max`, three by default, a role's setting): a
  new subject past it is counted (`issues-capped`) and opened at a later
  run that finds it again.

The product owner takes them from there, as any issue it never read:
first in its next run, named to its agent as that role's draft to refine.
Its own import of a backlog file goes through the same way, keyed by the
text it quotes; a text whose issue was closed is not opened again. The
report issues — the documentalist's, the product owner's, the weekly
sample's — are not subjects: each is one issue kept in place
(`KeepIssue`), as Renovate's dashboard.

### Borrowed, built, proved

**Borrowed**: issue forms for the four fields, milestones, sub-issues,
close reasons, labels, `KeepIssue`, `Sticky`. **To build**: listing a
project's open issues on every forge; milestones and closing with a reason
in the forge interface; the priority labels and the order rule; "its sources changed since it was confirmed" for an
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

1. The numbers: three ignored runs, three closings a run at first, a fifth
   of the backlog, one wrong closing to drop an act — the panel's, to be
   measured.
2. When the product manager role is written up.

## Consequences

- The backlog is kept by the machine between a need and its acceptance;
  the person states needs, accepts results, and can undo any act.
- Every role's issues share one way in; their lifecycle is the PO
  assistant's.
- A roadmap file goes, once its entries are issues.

## Amendment (2026-10-07): a person's new issue is read before the catch-up

A gardening run on workline's own backlog read eight issues: three
proposals a person had ticked, then the five oldest never read. A raw
issue the maintainer had just opened was not read: it got its state that
night, and then waited behind the catch-up of older issues, several
nights at eight a run.

**Borrowed**: triage practice puts a fresh issue first — Kubernetes
labels every new issue `needs-triage` until a person sorts it — while a
bot with a cap per run (gitlab-triage's `limits`) works through the rest
over several runs ([research](../research/product-owner.md)).

**The order of a run's reading**:

1. What a person asked for: a ticked proposal's issue, an issue read
   again for a changed need, then an act proposed only for the cap.
2. What came since the last run, newest first: an issue opened since —
   by a person, or by a role for a finding — and one a person wrote on,
   edited or reopened since it was read.
3. The catch-up, oldest first: the issues never read from before — the
   backlog there at the role's first run, and what an import opened.
4. The issues whose code changed, or with spec findings to answer.

**How "since the last run" is known, with no date**: an issue the role
finds without a state comment, once it ran on that backlog (a state or
its report exists), was opened since; its first state says `new`, and it
is read in the same run, its acts checked against that first state,
written before them. `new` stays until it is read, so the cap or a run
with no agent never sends it to the catch-up. A role's finding is
opened with `new`; an import's issues are not. On the role's first run,
nothing is new: the whole backlog is the catch-up.
