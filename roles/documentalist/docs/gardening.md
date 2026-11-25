---
sources:
  - roles/documentalist/code/documentalist.go#preRun.asksNothing
  - roles/documentalist/code/documentalist.go#preRun.askStale
  - roles/documentalist/code/freshness.go#staleForAgent
  - roles/documentalist/code/freshness.go#staleTask
  - roles/documentalist/code/freshness.go#unreadNote
  - roles/documentalist/code/documentalist.go#preRun.askReader
checked: e81df7b34
verified: agent:claude-code
---
# Documentalist — the gardening tasks

Gardening is the scheduled run, nightly or weekly (`schedule`): what the
agent is asked then, beside [the suspect docs](tasks.md), and how its fixes
reach a person. Each task says when it comes. Part of
[the documentalist](../README.md).

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

## Budgets and duplicates

A doc over its budget, a card too long or too short, a passage written
in two docs: reported by [the checks](checks-docs.md), for a person. No
agent rewrites them (removed 2026-11-25: the tasks never ran).

## Pages read for their reader

Last in each run, after the other tasks: in the same round when it asks
nothing else, else in a round of its own. The pages the reader checks
flagged, read for their reader ([more](reader.md)).
