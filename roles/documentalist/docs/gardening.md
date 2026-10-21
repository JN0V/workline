---
sources: [internal/builtin/documentalist/freshness.go, internal/builtin/documentalist/dedupe.go, internal/builtin/documentalist/mergecard.go, internal/builtin/documentalist/waiting.go, internal/builtin/documentalist/condense.go, internal/builtin/documentalist/documentalist.go, internal/builtin/documentalist/kept.go]
checked: a840c0a3
verified: agent:claude-code
---
# Documentalist — the gardening tasks

Gardening is the scheduled run, nightly or weekly (`schedule`): what the
agent is asked then, beside [the suspect docs](tasks.md), and how its fixes
reach a person. Each task says when it comes. Part of
[the documentalist](../README.md).

## Stale docs

With no suspect doc to judge, the stale docs become the task, each with its
sources as they are now, in full.

- The agent confirms it, fixes it, or says with a `note` that the sources
  do not tell.
- A doc whose sources are too large to be given in full is
  [judged in parts](parts.md), unless the project turns
  it off; then it stays for a person.
- One of whose sources cannot be read (a section whose heading is gone)
  stays for a person, the finding saying why: confirming it unread would
  only fake its freshness.

## Duplicates

With no suspect or stale doc, the first passage written in two docs: keep
it in the doc it belongs to, and in the other a sentence and a link to it.
It comes before condensing: two copies drift apart. The judge checks that:

- what leaves a doc is found in the other;
- it links to the one keeping it;
- no MUST or SHOULD is lost;
- the passage is no longer repeated.

## Condensing

With no suspect or stale doc and no repeated passage, the doc most over its
budget (a doc too long first, then an agent's entry point, a section): bring
it within budget by moving whole parts into a new doc, and linking to it.

- One doc a run.
- A doc whose task would not fit the role's context budget is not given
  to the agent (`too-large-to-condense`): a person moves its parts out,
  and the next doc over its budget is condensed. A card too large to split
  is reported the same way (`too-large-to-split`).
- A history doc — a changelog, a decision record, a tried or research
  record — is never the task: moving its parts away rewrites the record;
  its budget stays reported, for a person.
- A merge request or a push never turns into a rewrite of the docs.
- A patch still refused once asked again is said as warnings; at night
  the job stays green, and the next run asks again
  ([ADR-0037, amended](../../../docs/adr/0037-the-schedule-runs-every-step.md#amendment-2026-10-09-a-refused-proposal-is-said-not-failed)).

The judge refuses a condense patch when:

- it touches another existing doc;
- a line that leaves the doc is found in no new doc, unchanged but for a
  heading's level — moved, not rewritten (`rewritten`); a reference
  updated in place, keeping most of its words, is fine;
- a MUST or SHOULD is lost;
- a new doc is not linked from the doc (`not-linked`);
- the budget problem it was for remains (`still-over-budget`);
- it brings a budget problem the docs did not have, or makes one bigger
  (`patch-introduces`) — a part over its budget already, moved whole into
  a new doc, its text word for word, is the same problem moved, not a
  new one.

A condense, split, merge-card or dedupe hunk quoting a doc wrong is placed
as for a suspect doc ([hunks](judge.md#hunks-that-misquote-the-doc)); one
whose place is in doubt is refused.

A condense, split, merge-card or dedupe patch moving an existing doc's
`checked`, or creating a doc that carries one, is refused
(`checked-unread`): moving text gives none of its sources, so a new doc
starts without `checked`, listed unchecked for a person.

## Merging cards

With nothing else to do, a card too short to stand alone goes into
the card it belongs with, chosen by the agent among the others, those of
its folder first: its text added as it is, under a heading, the card
deleted, and every link to it pointed there. The judge checks that:

- the text is found in one card, which loses nothing;
- the docs linking to it change those links only;
- no link is left dangling.

## Splitting a card

Next, a card too long holds more than one concept: it keeps its first, and
each other one moves, as written, into a card of its own that it links to.

- The judge checks what condensing checks, and that each new doc is a card.
- Then a judge model — not the one that split it — is asked whether each
  card holds one concept, the one its title names. A no goes back to the
  agent with its reason.

## Pages read for their reader

Last in each run, after the other tasks, in a round of its own: the pages
the reader checks flagged, read for their reader ([more](reader.md)).

## Records kept on merged merge requests

First, in a round of its own: the docs judged on a person's merge request
with nothing to fix, whose record was kept in the documentalist's comment
there, not committed
([on a merge request](push.md#a-doc-found-with-nothing-to-fix-no-commit)).

- **Which merge request**: for each suspect doc, the last commit changing
  it or a source, and the merge request the forge says became it, or the
  closest after — GitHub's merge commit, GitLab's merge or squash commit.
- **The record holds** when the doc and each source are, in that commit and
  now, as judged: the doc is not put before an agent, and `judged: <that
  commit>` (or `judged-in-parts`, as the record says) goes in
  `workline/documentalist/records`, which only records
  when docs were read: it merges alone at `normal`
  ([the merge policy](../../../docs/merge-policy-normal.md)).
- **Otherwise the doc is judged again**, its finding naming the file no
  longer as judged.
- **Not for a doc read in parts that night**: its sources too large to go
  whole, its record says only what changed was read; it is
  [read in parts](parts.md) instead
  ([#462](https://github.com/JN0V/workline/issues/462)).
- **The cost**: one listing of the merge requests merged since the oldest
  such commit, and one read of the comments of each one a doc points to
  ([ADR-0010, amended](../../../docs/adr/0010-docs-judged-on-the-merge-request-and-by-gardening.md#amendment-2026-10-17-the-commit-a-merge-request-became-stands-for-its-record)).
- **Never a pass by guess**: a listing or a comment unread is said
  (`record-unread`), and the docs are judged again; so is a comment
  carrying its marker that workline's own account did not write
  ([why](push.md#a-doc-found-with-nothing-to-fix-no-commit)).

## One merge request per task

Run with `--open-merge-request` (the CI templates' scheduled jobs):

- a gardening task's patch goes on a branch of its own,
  `workline/documentalist/<task>`, with a merge request titled after the
  task; running the task again updates it
  ([ADR-0006](../../../docs/adr/0006-gardening-opens-one-merge-request-per-task.md));
- while `max-open-merge-requests` of them wait for review, gardening
  proposes nothing (`gardening-paused`), and what it finds is still
  reported;
- while a task's own merge request waits, the docs it would judge wait too
  ([ADR-0013](../../../docs/adr/0013-what-was-judged-is-not-asked-again.md)):
  the docs judged whole, while `suspect` or `stale` waits;
- a doc judged in parts waits while any of these merge requests changes
  it — its fix, or only the record of when it was judged —; the next doc
  goes, each run's on a merge request of its own, `fix-<its first doc>`
  (a short hash added when another doc's has that name)
  ([ADR-0013, amended 2026-10-14](../../../docs/adr/0013-what-was-judged-is-not-asked-again.md#amendment-2026-10-14-a-doc-waits-while-a-merge-request-holds-it));
- a doc to condense or split, a card to merge, a repeated passage wait
  while their own merge request does; the next one goes;
- `records` is proposed again only for a doc it does not hold yet: then
  with every record that still holds;
- a task proposed by an earlier round of the same run counts as waiting: a
  CI night judged with `--no-apply` that proposes the docs judged whole
  goes round again, and the docs put off to parts are judged the same
  night
  ([ADR-0013, amended](../../../docs/adr/0013-what-was-judged-is-not-asked-again.md#amendment-2026-10-02)).
  Workline's first nightly had put thirteen off "to a later round" a CI
  run never had;
- the merge request lists each doc it judged with what became of it: fixed
  and in how many places, still true, or nothing found wrong; `checked`
  moved or `judged` recorded; the code examples a fix wrote, unverified
  or checked ([code examples](examples.md)); for a doc whose sources did
  not all go whole, what was read and what was not
  ([the judge](judge.md#what-the-finding-says)).

## When the base moves

Each night, after applying what it proposed, gardening tends the merge
requests it opened before (`workline follow --gardening`,
[ADR-0034, amended](../../../docs/adr/0034-the-release-fix-follows-its-base.md#amendment-2026-10-12-gardenings-merge-requests-follow-too-or-are-closed)).
One the base moved under is:

- **rebuilt** on the base's tip, its change redone, the merge request kept:
  a `checked` the base moved beside the `judged` it wrote is no conflict;
- **closed**, one comment saying why, when nothing is left to merge:
  - the base holds its change already;
  - the base changed the same lines: the next gardening judges the docs
    again and proposes what is still needed;
  - it only records when docs were read (`judged`), and the base changed
    those docs since; a record whose docs the base left as they were is
    rebuilt, so they are not judged again for the same record;
- **left to a person** who committed to it: neither rebuilt nor closed,
  and named in the job's summary when it no longer merges, or when git
  cannot tell (`conflict-unread`). Running the task again never pushes
  over that commit either (`not-proposed`).

A push to the base leaves them alone: only, when set, people's pull
requests in conflict on workline's lines follow each push
([more](follow.md)).
