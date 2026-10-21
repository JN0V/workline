# ADR-0048: The agent reads the pages the reader checks flag, last in a gardening run

- **Status:** accepted (2026-10-15)
- **Date:** 2026-10-15
- **Decided by:** the maintainer, on 2026-10-15
  ([#256](https://github.com/JN0V/workline/issues/256#issuecomment-6186912301):
  "1A 2A", and the defaults proposed there)
- **Builds on:** [ADR-0003](0003-documentalist-checks-in-the-engine.md)
  (the reader checks in the engine),
  [ADR-0006](0006-gardening-opens-one-merge-request-per-task.md) (one
  merge request a task), [ADR-0013](0013-what-was-judged-is-not-asked-again.md)
  (not asked again), [ADR-0014](0014-checked-is-earned-by-what-was-read.md)
  (quotes checked by the engine)

## Context

- The reader checks ([#193](https://github.com/JN0V/workline/issues/193))
  say where a user page is hard to read: a name with no link, a wall of
  prose, a long table cell, a flow with no diagram. They cannot say what
  a reader would miss or skip there, nor rewrite it.
- On workline's `main` at v0.30.0 they flagged 8 pages, 78k characters,
  about 65k tokens estimated (711 + 0.82 a character): more than one task.
- Two choices were left open: when the agent reads them, and how what it
  proposes lands.

## Decision

- **When**: last in each gardening run, after the suspect and stale docs,
  duplicates, condensing, splitting and merging cards: docs are made true
  before they are made easy to read. A round that asked another task
  writes `in/then`, and the engine goes one round more for the reader
  task alone, once the run's other rounds are done, within what the run
  may still spend; when `ai-max-tokens` is spent, it is skipped, and said.
  The project's `schedule` sets how often.
- **What it holds**: only pages the reader checks flagged — a page only
  out of reach is fixed on another page, and is not read for that —, at
  most `reader.agent-pages` (4) a run within one task's characters
  (40,000). The rest wait for a later run, said on their findings; a page
  too large alone for a task is said (`too-large-to-read`).
- **What the agent answers**: per page, a `claim` for each passage a
  reader would miss or skip, quoting it, and one `patch`.
- **How it lands**: a merge request of its own, `reader-<first page>`,
  listing what the agent found a reader would miss.
- **What the engine checks, never blocking**:
  - a claim whose quote is not at its lines is dropped, said;
  - a page's patch changing its header, lines no kept claim quotes, or
    losing a link, a code span, a number or a heading, is withheld whole,
    said, the page left as it was.
- **Not read again until the page changes**:
  - the page's header records a hash of its body as the run leaves it
    (`reader:`), on the same merge request;
  - a page a waiting merge request changes waits too;
  - a merge request closed without merging is not proposed again each
    night: its branch stays, and where it left the default branch holds
    the page as read. While the page is still that, it is not read again.
- **Caps**: `ai-max-tokens` as every task; with no agent, the checks'
  findings stand alone, and no page is recorded as read.

## Consequences

- A night on workline's flagged pages: 3 pages, a prompt of 43,187
  characters, about 36k tokens estimated, measured with a recording agent
  (no tokens) on this engine; by their sizes, the 8 pages take three runs.
- A night with other tasks spends their tokens first; the pages wait only
  when the run's `ai-max-tokens` is spent.
- A closed merge request whose branch is deleted leaves no trace: its
  pages are read again at the next run.

## Amendment (2026-10-21): a close of workline's own is no decline

- **Context**: [#547](https://github.com/JN0V/workline/pull/547), a reader
  merge request, conflicts with `main` on another of its pages.
  `workline follow` closes such a merge request at the next gardening, its
  comment saying the docs are judged again on `main`. Its branch stays,
  so its pages unchanged since read as a person's decline, and are never
  read again ([#592](https://github.com/JN0V/workline/issues/592)).
- **Decision**: a merge request whose closing comment carries workline's
  marker, `follow`'s, is no decline: its pages are read again at the next
  gardening. One a person closed still holds its pages until they change.
- **How it is told**: the forge reads the last merge request from the
  branch and its comments — GitHub, GitLab, the local and the simulated
  forge. Through a forge plugged by a command, every close reads as a
  person's. A close the forge could not read is said
  (`reader-declined-unread`), its pages held.
