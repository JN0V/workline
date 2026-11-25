---
sources:
  - roles/documentalist/code/documentalist.go#preRun.asksNothing
  - roles/documentalist/code/documentalist.go#preRun.askStale
  - roles/documentalist/code/freshness.go#staleForAgent
  - roles/documentalist/code/freshness.go#staleTask
  - roles/documentalist/code/freshness.go#unreadNote
  - roles/documentalist/code/documentalist.go#preRun.askRewrite
  - roles/documentalist/code/dedupe.go#pickDedupe
  - roles/documentalist/code/dedupe.go#judgeDedupe
  - roles/documentalist/code/mergecard.go#pickMergeCard
  - roles/documentalist/code/mergecard.go#writeMergeCardTask
  - roles/documentalist/code/mergecard.go#judgeMergeCard
  - roles/documentalist/code/documentalist.go#preRun.askReader
checked: e81df7b34
verified: agent:claude-code
---
# Documentalist — the gardening tasks

Gardening is the scheduled run, nightly or weekly (`schedule`): what the
agent is asked then, beside [the suspect docs](tasks.md), and how its fixes
reach a person. Each task says when it comes. Part of
[the documentalist](../README.md).

- [Condensing and splitting a card](gardening-condense.md): a doc over
  its budget, a card too long.
- [Its merge requests](gardening-merge-requests.md): the records kept on
  merged merge requests, one merge request per task, and when the base
  moves.

## Stale docs

With no suspect doc to judge, the stale docs become the task, each with its
sources as they are now, in full.

- The agent confirms it, fixes it, or says with a `note` that the sources
  do not tell.
- A doc whose sources are too large to be given in full is
  [judged in parts](parts.md), unless the project turns
  it off ([`judge-in-parts`](../README.md#settings)); then it stays for a
  person.
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
- the passage is no longer repeated;
- no reader problem is added (`patch-introduces`).

Then come [condensing, and splitting a card](gardening-condense.md).

## Merging cards

Last, with nothing else to do, a card too short to stand alone goes into
the card it belongs with, chosen by the agent among the others, those of
its folder first: its text added as it is, under a heading, the card
deleted, and every link to it pointed there. The judge checks that:

- the text is found in one card, which loses nothing;
- the docs linking to it change those links only;
- no link is left dangling;
- no reader problem is added (`patch-introduces`).

## Pages read for their reader

Last in each run, after the other tasks: in the same round when it asks
nothing else, else in a round of its own, after condensing's. The pages the reader checks
flagged, read for their reader ([more](reader.md)).
