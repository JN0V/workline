---
sources: [internal/builtin/documentalist/readertask.go, internal/builtin/documentalist/reader.go, internal/builtin/documentalist/documentalist.go, roles/documentalist/role.yaml, internal/engine/routine.go]
checked: 30139b53
verified: agent:claude-code
---
# Documentalist — pages read for their reader

[The reader checks](checks-docs.md#docs-for-their-reader) say where a user page
is hard to read. With an agent, gardening also has it read those pages as
their reader would, and propose a fix
([ADR-0048](../../../docs/adr/0048-the-agent-reads-the-pages-the-reader-checks-flag.md)).
Part of [the documentalist](../README.md).

## When

- Last in each gardening run, after the other tasks — suspect or stale
  docs, duplicates, condensing, splitting, merging cards: docs are made
  true before they are made easy to read.
  - A night with another task asks it in a round of its own, once the
    others are done (`in/then`,
    [the role contract](../../../docs/spec/role-contract.md#one-run)).
- With no agent, nothing is read: the checks' findings stand alone.
- `ai-max-tokens` caps it as any task: once the run spent it, the pages
  wait, said (`ai-max-tokens`).

## What the task holds

- Only the pages a reader check flagged: a name with no link, a paragraph
  or a table cell too long, a flow with no diagram.
  - A page only out of reach is fixed on another page: it is not read for
    that.
- Each page whole, its lines numbered, under what the checks found.
- At most `reader.agent-pages` pages (4), within one task (40,000
  characters). The rest wait for a later run, said on their findings.
- A page too large for a task alone: `too-large-to-read`, for a person.
- A page is not read again until it changes, nor while a merge request
  changing it waits for review, nor after one was closed without merging
  (below).

## What the agent answers

- A `claim` for each passage a reader would miss or skip: the page, its
  lines, the passage quoted, and why.
- One `patch` making those passages easy to read:
  - changing only the lines its claims quote;
  - keeping every link, code span, number and heading;
  - rewording, never adding a fact; the header left as it is.
- No claim for a page that serves its reader.

## What the engine checks

Never blocking: what does not hold is said, for a person, and the run
passes.

- A claim whose quote is not at its lines is dropped
  (`reader-claim-dropped`).
- A page's patch is withheld whole, the page left as it was
  (`reader-patch-withheld`), when it:
  - changes the page's header;
  - changes lines no kept claim quotes;
  - loses a link, a code span, a number or a heading.
- A patch to a page the task did not give, or deleting one, is withheld.

## Where it lands

- A merge request of its own, `workline/documentalist/reader-<first page>`:
  the pages, what became of each, what the agent found a reader would
  miss, and what was withheld.
- **It waits for a person's yes**, even where the project's merge policy
  merges workline's own docs alone (`normal`): a comment on it says so.
  A project lets these merge alone with
  `merge: {autonomy: normal, alone: [reader]}`
  ([the merge policy](../../../docs/merge-policy-normal.md#the-readers-rewordings-wait-for-a-yes)).
- Each page read gets `reader: <hash>` in its header: its body as the
  merge request leaves it. While the page's body still matches, it is not
  read again. A page with no header gets one, in a comment no rendering
  shows.
- A merge request closed without merging keeps its branch: where it left
  the default branch holds the page as read. While the page is still that,
  it is not read again, said on its findings — no fix proposed each night.
  - A branch deleted with it leaves no trace: the page is read again.
  - Branches that cannot be listed (the remote unreachable) are said
    (`reader-declined-unread`, a warning).

## Cost

- On workline at v0.30.0, measured with a recording agent (no tokens): 3
  pages a run, a prompt of 43,187 characters, about 36k tokens estimated
  (711 + 0.82 a character); each page about 9k to 10k of it.
- `reader.agent-pages: 0` reads no page.
