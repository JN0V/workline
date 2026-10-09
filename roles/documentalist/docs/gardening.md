---
sources: [internal/builtin/documentalist/documentalist.go, internal/builtin/documentalist/freshness.go, internal/builtin/documentalist/dedupe.go, internal/builtin/documentalist/condense.go, internal/builtin/documentalist/mergecard.go]
checked: bb8a9a3
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
- A doc whose sources are too large to be given in full stays for a
  person: confirming it unread would only fake its freshness.

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

## Merging cards

Last, with nothing else to do, a card too short to stand alone goes into
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
  a doc judged in parts, whose task is `fix`, is then asked before the docs
  judged whole;
- a task proposed by an earlier round of the same run counts as waiting: a
  CI night judged with `--no-apply` that proposes the docs judged whole
  goes round again, and the docs put off to parts are judged the same
  night, on the `fix` merge request
  ([ADR-0013, amended](../../../docs/adr/0013-what-was-judged-is-not-asked-again.md#amendment-2026-10-02)).
  Workline's first nightly had put thirteen off "to a later round" a CI
  run never had;
- the merge request lists each doc it judged with what became of it: fixed
  and in how many places, still true, or nothing found wrong; `checked`
  moved or `judged` recorded.
