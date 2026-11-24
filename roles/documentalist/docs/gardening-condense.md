---
sources:
  - roles/documentalist/code/documentalist.go#preRun.askRewrite
  - roles/documentalist/code/condense.go#condensable
  - roles/documentalist/code/condense.go#condenseCandidates
  - roles/documentalist/code/records.go#isHistory
  - roles/documentalist/code/records.go#historyStems
  - roles/documentalist/code/condense.go#tooLargeToRewrite
  - roles/documentalist/code/condense.go#writeCondenseTask
  - roles/documentalist/code/condense.go#judgeCondense
  - roles/documentalist/code/condense.go#normal
  - roles/documentalist/code/condense.go#editedInPlace
  - roles/documentalist/code/condense.go#bornChecked
  - roles/documentalist/code/condense.go#applyDoc
  - roles/documentalist/code/condense.go#pickSplit
  - roles/documentalist/code/condense.go#writeSplitTask
checked: e81df7b34
verified: agent:claude-code
---
# Documentalist — condensing and splitting a card

Part of [the gardening tasks](gardening.md).

## Condensing

With no suspect or stale doc and no repeated passage, a doc over its
budget (a doc too long first, then an agent's entry point, a section):
bring it within budget by moving whole parts into a new doc, and linking
to it.

- One doc a run.
- A doc whose task would not fit the role's context budget is not given
  to the agent (`too-large-to-condense`): a person moves its parts out,
  and the next doc over its budget is condensed. A card too large to split
  is reported the same way (`too-large-to-split`).
- A history doc — a changelog, a decision record, a tried or research
  record — is never the task: moving its parts away rewrites the record;
  its budget stays reported, for a person.
- An archived doc (`archived: <date>` or `true` in its header) is never
  the task, nor reported over budget or as a duplicate; its links are still
  checked ([budgets](checks-docs.md#budgets)).
- Only when gardening: a merge request or a push never turns into a
  rewrite of the docs.
- A patch still refused once asked again is said as warnings; at night
  the job stays green, and the next run asks again
  ([ADR-0037, amended](../../../docs/adr/0037-the-schedule-runs-every-step.md#amendment-2026-10-09-a-refused-proposal-is-said-not-failed)).

### What the judge refuses

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
  new one; so does a reader problem a new page among the user pages
  brings (`patch-introduces`, [reader checks](checks-docs.md#docs-for-their-reader)):
  the task says to put a new doc in a subfolder of the doc's own, unless
  it is written for readers.

A condense, split, merge-card or dedupe hunk quoting a doc wrong is placed
as for a suspect doc ([hunks](judge-hunks.md#hunks-that-misquote-the-doc)); one
whose place is in doubt is refused.

A condense, split, merge-card or dedupe patch moving an existing doc's
`checked`, or creating a doc that carries one, is refused
(`checked-unread`): moving text gives none of its sources, so a new doc
starts without `checked`, listed unchecked for a person.

## Splitting a card

After condensing, a card too long holds more than one concept: it keeps
its first, and each other one moves, as written, into a card of its own
that it links to.

- The judge checks what condensing checks, and that each new doc is a card.
- Then a judge model is asked whether each card holds one concept, the
  one its title names. A no goes back to the agent with its reason
  ([asked again](judge-hunks.md#asked-again)).
