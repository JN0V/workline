---
type: reference
sources: [internal/backlog/backlog.go, internal/builtin/productowner/productowner.go]
status: draft
checked: 21e91932
verified: agent:claude-code
---
# The conversation with the reporter

Part of [Acts on the backlog](../backlog-acts.md): the rounds written to an issue's reporter, and their agreement.

Each comment the engine writes to a reporter is a round (ADR-0021): an
`ask`, or a `refine` proposed to an outsider. Its marker holds its round —
`<!-- workline:product-owner/ask -->` for the first question,
`ask=2`, `ask=3` after; `proposal=1`, `proposal=2` for a text proposed. The
issue's comments are the record; the state does not copy them.

| Check | Otherwise |
|---|---|
| A person commented after the last round (a comment not ending with an engine marker: one quoting the engine's is a person's) | Dropped: `already-asked`, or `already-proposed` for a text; nothing written |
| No question the act holds — each from the sentence before it (". " or "! " then a capital; "e.g. Which" cuts too) or its list item's line (one starting with a capital) to its `?`, the bullet, spaces and case aside — is one an earlier round asked, its lead ("to refine this issue:", "What it still needs:") aside | That question left out, from its sentence or its list item to its `?`, the rest of the act written — a refine's drafts, its other questions (`question-left-out`, info, said "The question "…" was asked before, so it was left out; the rest was kept." once the act is done or proposed, never of one dropped later); an `ask` with no question left is dropped: `asked-before` |
| Rounds before it under `acts.ask.rounds` (three) | Proposed on the issue: "Settle #N with its reporter, written to 3 times already", the questions or the text it would write (`asks-spent`) |

A later ask thanks the reporter ("thank you; to refine this issue,
still: …"). The agent is given the conversation in order — the engine's
rounds among the people's comments — and the rounds spent.

**An outsider's issue** — its reporter without write access, not opened by
a role, not bearing `workline:accepted` — is theirs: a `refine` writes
nothing in its body. The engine comments to the reporter, naming them:
what the role understood (its `why`), the sections it would add (Need and
Validation as drafts from their words), what it still needs (`questions`),
and how to agree; the sections also in a YAML block, under "As the engine
reads it" — the block read back. Its fence is longer than any run of
backticks it holds, and only that fence at a line's start ends it, so a
code block in an example or steps is read back whole (#391); elsewhere the
agent's fences are turned to `'''`. The issue gets `workline:to-refine`; the act counts in
`refine`'s cap. Agreement is read where only the right people write:

- **`workline:accepted`**, set by a person of the project: the next run
  writes the sections of the last proposal the body still lacks, with no
  agent, then moves the issue to ready in the same run, the drafts
  accepted with it — unless its reporter or a person of the project
  commented since, a comment the role has not read: the agent reads the
  answer first, its `refine` is written in the body, and the label moves
  the issue to ready at the next run (#390). A bot's or a stranger's
  comment changes nothing, nor one the role read leaving its text as it
  was. With no agent, nothing of it is written while the answer is
  unread: the issue waits on a person;
- **the reporter's own edit** of the body (only its author or a writer
  can edit it): read again as any body changed; moving it to ready stays
  proposed (`reporter-outside`) until a person accepts;
- **a reply `agreed`** (ADR-0021, amended): after the last round, that
  round a proposal, the last comment of the reporter or of a person of
  the project (not a bot, its author named) has `agreed` as its first line
  (case, spaces, a final "." or "!" aside); a stranger's or a bot's
  comment after it changes nothing, a later word of theirs replaces it.
  With `workline:accepted` set too, the label's path applies at once: the
  text written and the issue moved to ready. The next run writes the sections of the last
  proposal the body still lacks, with no agent, Need and Validation as
  drafts, and does not read the issue again for that reply; ready, a
  split, a rename stay the label's. The plan checks the agreement again
  on the forge, and that the sections are those proposed (`not-agreed`).

The engine's block of the last proposal that does not read (#302) writes
nothing, whoever agrees: `pre` says it each run while the body lacks a
section, a warning (`proposal-unread`), for a person to copy the sections
in. The ask, draft and reply marks hold nothing but their name, a round or
a comment's id, nothing
that can fail to read: a comment ending in one is the engine's, any other
a person's.

Any other reply of the reporter's, or of a person of the project, is read
as an answer — the agent may propose a revised text, a round; a
stranger's or a bot's brings nothing back ([A person's answers](answers.md)). Each
comment is given to the agent with who wrote it: the reporter, of the
project, outside it, a bot.
