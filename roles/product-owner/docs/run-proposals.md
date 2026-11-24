---
sources: [roles/product-owner/role.yaml, roles/product-owner/prompts/instruction.md, roles/product-owner/prompts/policy.md, roles/product-owner/code, internal/backlog]
checked: e81df7b34
verified: agent:claude-code
---
# Product owner — a run: what it proposes

Part of [a run](run.md), step 3: what the agent proposes, and what the
engine checks before it does anything.

## 3. What it proposes, what the engine checks

### What it proposes

- **Completing**: the sections an issue lacks (`refine`), a question to
  its reporter (`ask`), the move to `ready` — to `workline:needs-tests`
  first where the project routes the tester and its Validation has
  scenarios —, the code it is about
  (`sources`), an evident duplicate (`close`), the kind of an issue no
  label tells (`kind`, [the kind](kind.md)).
- **A blocked issue**: `unblock` to its state or to refine, or
  `blocked` afresh; blocked on its own read in refine, its text
  rewritten with neither, back to refine. The developer's ask for files
  outside the Scope:
  the Scope widened — a new file its pull request adds counts —, or
  why not; a widening left out keeps the issue blocked. A run too big
  for its role, or one predicted past its normal at ready: the Scope
  narrowed, split, granted `workline:size/large` or left to a person ([how](../../../docs/spec/backlog-acts/refine.md#its-size-at-ready)).
- **A question unanswered** past `acts.ask.wait`: its assumption
  written (`assume`)
  ([how long a question waits](settings.md#how-long-a-question-waits)).
- **Written for a reader**: a Need in one line, who needs what and why;
  one real example; a Validation as scenarios a person runs; the
  builder's part below a line; plain words in English, a link for each
  decision or doc cited ([writing an issue](writing.md)).
- **Never invented**: an issue too thin, holding several topics, or
  resting on a closed issue gets one plain question; one the code
  already does is said on it for you to close.
- **Off by default**: milestones, priorities, splits, renames, links,
  obsolete issues ([settings](settings.md#off-by-default)).

### What the engine checks

- The engine checks each act when it applies it: never a section a person
  wrote, edited or deleted; `ready` only with Need, Validation and Scope, and no draft;
  a Scope's files read from its text, each path found or on a line
  `New:`, a line naming a path the repository lacks left out, the Scope
  whole when no line is left ([writing](writing.md)); never `ready` on
  an issue developing
  ([#1504](https://github.com/JN0V/workline/issues/1504)); an
  outsider's issue proposed; never closed as not planned.
- An issue read and left with nothing that refines it — a kind or an
  order alone counts for nothing — is said in the summary, and on the
  issue: what it lacks, each act dropped and why, the agent's note, and
  when it is read again.
- In refine, the line wakes the product owner on it again, twice at
  most on the same text, told to refine it now; then it blocks the
  issue for you
  ([ADR-0059, amended](../../../docs/adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#amendment-2026-11-03-an-issue-left-unrefined-is-woken-again)).
  Given the same code as its last read, it is not woken again: blocked
  at once, the code it got said
  ([amended again](../../../docs/adr/0059-an-issue-moves-by-its-state-each-step-wakes-the-next.md#amendment-2026-11-03-a-read-again-is-given-new-code-or-none-is-made)).
- Its sources named from the repository's folders, or only code the
  run did not give, need no line quoted: the issue is read again with
  that code.
- A Scope written with no code given for the issue is left out, its
  paths kept as the code to read: it is written from that code at the
  next run, never guessed
  ([#970](https://github.com/JN0V/workline/issues/970)). Beside sources
  whose code it was given, it is written at once.
- Then it does it, proposes it on its issue, or drops it, by the kind's
  mode and cap, and by the issue: one set aside gets nothing; one where a
  person undid that kind gets a proposal.
