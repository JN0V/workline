# ADR-0021: The product owner talks with the reporter, a few rounds, then a person

- **Status:** proposed; amended 2026-10-05 (a reply that agrees)
- **Date:** 2026-10-05
- **Builds on:** ADR-0018 (the product owner; "an issue opened by someone
  outside the project is theirs"), principles 1, 4, 6, 7, 14

## Context

Refining asked an issue's reporter once, and never again: an answer was
read, but a follow-up was refused (`already-asked`), and the agent was not
shown what it had asked. An outsider's issue had its sections written in
its body, someone else's text, and only its move to ready waited — in the
report, for the maintainer, never for the reporter. ADR-0018 is over its
length budget; this is a decision of its own: how the role converses.

docs/research/product-owner.md ("Asking, and the answer") found that bots
count an answer as a comment after the question; that who wrote it matters
when it is a decision (no-response: the author only; Dependabot, Copilot:
a writer only); that rounds are few; and that none asks a narrower question
after an answer — the follow-up is a person's.

## Decision

**A conversation of rounds.** Each comment the role writes to a reporter —
a question, or a refined text proposed — is a round, its marker numbered
(`product-owner/ask`, `ask=2`, `proposal=1`…). The issue's comments are the
record: the questions are there, verbatim, where the reporter sees them; the
state comment does not copy them.

- **Again only after an answer**: a person's comment after the last round.
  Without one, nothing is written (`already-asked`,
  `already-proposed`): no reminders, no ping storm.
- **The answer is read**: the issue is read again (a person commented), its
  conversation given to the agent in order, with the rounds spent. It
  refines, proposes ready, or asks what is missing.
- **Never the same question twice**, as a check: a question an earlier
  round holds, spaces and case aside, drops the ask (`asked-before`).
  Telling a paraphrase is the agent's, told so.
- **A few rounds, then a person**: three (`acts.ask.rounds`); past them,
  what the role would write is proposed in the report, "settle it with its
  reporter" — never closed on silence.
- **A person's answer is respected**: what a reply decides — the scope, a
  wording, that it is not wanted — is taken as given and not asked again;
  refusing it is still the person's (never *not planned*).

**An outsider's issue is proposed to its reporter.** For an issue whose
reporter has no write access, not opened by a role, not accepted, a
`refine` is not written in the body: the engine comments to the reporter
what the role understood, the sections as it would write them, what it
still needs, and how to agree — the sections also kept in a YAML block.
It shares the rounds and the cap of `refine`.

**Agreement is read where only the right people can write it**:

- the label `workline:accepted`, which only who may triage sets: the next
  run writes the proposed sections the body lacks, with no agent, then
  moves the issue to ready (the drafts accepted with it);
- the reporter's own edit of their issue's body — on GitHub only its author
  or a writer can edit it: their sections are theirs, read again at the
  next run; ready stays the project's to accept (`reporter-outside`).

A reply alone is not agreement: the forge interface gives no comment's
author, and on a public project anyone can comment. It is read as an
answer: the agent may propose a revised text, a round. (Amended below:
`agreed`, replied by the reporter or a person of the project.)

## Consequences

- A vague issue is refined in a few exchanges, without a person relaying
  questions; three unanswered-to-the-end rounds land in the report.
- An outsider sees what the project would make of their issue before
  anything of theirs is changed. Splitting, renaming and moving it to
  ready stay proposed to the project, in the report (ADR-0022): how the
  backlog cuts and names a need is the project's, its text the reporter's;
  `workline:accepted` lifts all.
- GitLab says nothing of write access in an issue: every reporter there
  counted as outside, so every refine there was proposed in a comment —
  until its members were read (ADR-0023).
- Left, then built (amendment below): comments read with their author, so
  a reporter's `agreed` agrees and a writer's answer is told from a
  stranger's.

## Amendment (2026-10-05): a reply that agrees, read with its author

Every forge now gives a comment's author, and whether they are of the
project (GitHub's author association; GitLab's members, Planner and
above, ADR-0023; a plugged forge's answer). "A reply alone is not
agreement" held because none was known. Now:

- **A reply agrees** when, after the last round — that round a proposal —
  the last comment of those who may agree has `agreed` as its first line
  (case, spaces and a final "." or "!" aside). Those who may agree: the
  reporter and the persons of the project — never a bot, nor an author
  the forge does not name. Anyone else's comment after it, a stranger's
  "+1", neither gives nor takes it back; a later word of the reporter's
  or the project's does, whatever it says (no reading of a withdrawal: a
  later `agreed` agrees again). Words a
  check reads, as triagebot's `@rustbot ready` or Dependabot's commands
  from writers: telling a "yes" in free text is not the engine's, and
  the agent's reading of one would let a reply write a body (principle 6).
  The proposal says so: "reply `agreed` alone".
- **What it agrees to is the text, not the project's acceptance**: the
  next run writes the sections of the last proposal the body still lacks,
  with no agent, Need and Validation as drafts — as the reporter's own
  edit of their body would. Moving it to ready, splitting and renaming
  stay the label's, `workline:accepted`, an act the forge records with
  who set it, by someone who may triage. Conservative on purpose: an
  insider's reply is a word in a comment where the label is a decision
  in the project's own tool; taking a reply for the label is left to a
  later decision, with what live use shows.
- **Checked again when applied**: the refine carries who agreed; the plan
  reads the comments again and writes nothing unless the agreement is
  there, by that person, and the sections are those last proposed
  (`not-agreed`) — never taken from the agent's answer.
- **The agent is told who wrote each comment**: its reporter, of the
  project, outside it, a bot. A stranger's word is still an answer to read.
