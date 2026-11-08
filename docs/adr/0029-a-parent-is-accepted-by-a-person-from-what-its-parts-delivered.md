# ADR-0029: A parent is accepted by a person, from what its parts delivered

- **Status:** accepted; amended 2026-10-07 (a test named as proof, read from the code); amended 2026-11-08 (the line tells a parent; a person accepts it with one label)
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

- **Its parts**, one rule for the comment, the report and the order: the
  forge's own relation (GitHub's sub-issues, GitLab's tasks through
  GraphQL) and the task list under `## Sub-issues` in its body, where a
  split links or lists its children. A part a person unlinked is no
  longer one; an issue the role split is still never closed by it.
- **Each part**: open (ready or not), closed as completed — and what
  closed it, from the forge (GitHub's `ClosedEvent.closer`, GitLab's
  state events: the commit or merge request of its last closing; a forge
  that refuses to say it, said "not read", never "by hand") —, or closed
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
  use shows parents whose items are proved in other words. A test named
  as proof, read from the code: decided below.

## Amendment (2026-10-07): a test named as proof, read from the code

An item of a Verification often names its proof: "`TestWriteRows`
passes", "the case `parent-x`". A part quoting it proved the item, yet
the test was taken on trust: a part could quote a test nobody wrote.

- **What names a test**: a code span in the item holding a test file's
  path (a tests or spec folder, a `_test`, `.test`, `.spec` or `_spec`
  file, a `test_` file), `path::name`, a test's own name (`TestX`,
  `test_x`, `testX`), or any name right after the words "test", "test
  case" or "conformance case" — never "case" alone, prose's "in that
  case".
  The rest of the item is prose.
- **Read from the code** at the run's commit, with git alone: the file
  there; the name a word in a test file. The comment says where.
- **Not there**: the item is not proved, whatever quotes it; the comment
  says which test is missing, and a finding (`proof-test-missing`). Git
  failing is not a test missing: "could not be looked for"
  (`proof-test-unread`), and the item is not proved either.
  Mechanical, no agent (principle 4).
- Whether the test passes is not read: that is CI's, on the pull request
  that closed the part, and its link is on the comment already.

Left: a test named in prose, outside a code span; a test in another
repository.

## Amendment (2026-11-08): the line tells a parent; a person accepts it with one label

[#1208](https://github.com/JN0V/workline/issues/1208): in a trial, a
split's parent stayed open for 10 hours, both parts merged, with no
comment and no label. "Each run" never came: in CI a product owner run
reads only the issue it was woken on, and nothing woke one on the
parent. A person lifting its block, as asked, took it off the line.
[ADR-0063](0063-every-open-issue-is-due-waiting-or-done.md#1-the-invariants)
wants every open issue due, waiting or done, said.

- **The line tells it**: where its state wakes nobody (no state label,
  or Blocked), a part closed or `workline:accepted` put on since the
  line last read it is new input. The product owner is woken on it with
  no agent: the comment above is written. Nothing new, it waits, said:
  on its open parts, or on a person.
- **The closing is proposed**: all its parts closed, the engine proposes
  closing it as completed, `workline:proposed` on it, at every level.
  A person's `workline:accepted` closes it, their yes named on it: one
  click, as for any proposal
  ([ADR-0038](0038-the-product-owner-proposes-on-the-issue-a-person-answers-there.md)).
  Closing it by hand still works.
- **Still a person's**: the role never closes a parent alone. The
  closing carries no quote: the comment on its parts is the evidence a
  person reads.
- **The mark is checked**, at the proposal and again at the yes: the
  issue was split (its parts in the forge, its task list or the role's
  split), and every part is closed now. An agent claiming the mark on
  another issue is dropped (`parent-not-split`); a part open again
  drops the yes (`parent-part-open`).
- **In refine too**: a parent left in refine after a split for its
  size is told the same way, when its own step is not due.

### A bug's parent

Decided by a panel (3 of 3): the middle path. The line may close a
bug's parent alone only when the parent's own reproduction test, the
reporter's symptom, passes on the default branch. A part's tests prove
the part, not the symptom the reporter saw. The engine keeps no such
test for a parent today, so this amendment always asks a person;
[#1212](https://github.com/JN0V/workline/issues/1212) builds the middle
path.

### Left

In CI, a part closing starts no run on its parent: the parent is told
at the next hourly reconcile, up to an hour late.
