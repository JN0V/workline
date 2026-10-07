# ADR-0029: A parent is accepted by a person, from what its parts delivered

- **Status:** accepted; amended 2026-10-07 (a test named as proof, read from the code)
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
