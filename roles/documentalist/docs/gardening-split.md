---
sources:
  - roles/documentalist/code/documentalist.go#preRun.askRewrite
  - roles/documentalist/code/split.go#splittable
  - roles/documentalist/code/split.go#splitCandidates
  - roles/documentalist/code/records.go#isHistory
  - roles/documentalist/code/records.go#historyStems
  - roles/documentalist/code/split.go#tooLargeToRewrite
  - roles/documentalist/code/split.go#inbound
  - roles/documentalist/code/split.go#writeSplitTask
  - roles/documentalist/code/split.go#judgeSplit
  - roles/documentalist/code/split.go#normal
  - roles/documentalist/code/split.go#editedInPlace
  - roles/documentalist/code/split.go#bornHeader
  - roles/documentalist/code/split.go#relinked
  - roles/documentalist/code/split.go#applyDoc
checked: d2f2a7183
verified: agent:claude-code
---
# Documentalist — splitting a doc over its budget

Part of [the gardening tasks](gardening.md).

## Splitting

A doc over its budget (a doc too long first, then an agent's entry
point, a section, a card): split it into linked pages, as a person would.

- Whole parts — a section, a list, a table — move, as written, into new
  pages beside it, named after it (`run.md`, `run-people.md`); a sentence
  and a link stay in their place.
- A link elsewhere in the repository to a part that moved follows it: the
  task lists those lines, and the patch changes where they point.
- A new page carries the doc's header: its `checked` as it is, the lines
  moved being the lines it vouched for, and those of its `sources` the
  page speaks of.
- A card splits into cards, one concept each ([below](#splitting-a-card)).
- One doc a run: in the same round when it asks nothing else; else in a
  round of its own once the others are done, before the
  [reader's](reader.md) (`in/then`, `budget`). Suspect docs do not
  starve it, but the round is dropped once `ai-max-tokens` is spent, or
  while `max-open-merge-requests` of its merge requests wait.
- A doc whose task would not fit the role's context budget is not given
  to the agent (`too-large-to-split`): a person moves its parts out, and
  the next doc over its budget is split.
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

The judge refuses a split patch when:

- it touches an existing doc other than the doc and those linking to its
  parts (`patch-off-task`);
- a doc linking to a part changes more than where its links point
  (`rewritten`);
- a line that leaves the doc is found in no new page, unchanged but for a
  heading's level or where its links point — moved, not rewritten
  (`rewritten`); a reference updated in place, keeping most of its words,
  is fine;
- a new page's `checked` is not the doc's (`checked-unread`), or it names
  a source the doc does not, or none when the doc names some
  (`sources-not-carried`);
- a MUST or SHOULD is lost;
- a new page is not linked from the doc (`not-linked`);
- the budget problem it was for remains (`still-over-budget`);
- it brings a budget problem the docs did not have, or makes one bigger
  (`patch-introduces`) — a part over its budget already, moved whole into
  a new page, its text word for word, is the same problem moved, not a
  new one; so does a reader problem a new page among the user pages
  brings (`patch-introduces`, [reader checks](checks-docs.md#docs-for-their-reader)),
  or a link left pointing at a part that moved (`dead-link`); a folder
  grown by the new pages' headers and links, or links to other sites
  moved with their lines, are not new problems.

A split, merge-card or dedupe hunk quoting a doc wrong is placed as for a
suspect doc ([hunks](judge-hunks.md#hunks-that-misquote-the-doc)); one
whose place is in doubt is refused.

A split, merge-card or dedupe patch moving an existing doc's `checked` is
refused (`checked-unread`): moving text gives none of its sources.

## Splitting a card

A card too long holds more than one concept: it keeps its first, and each
other one moves, as written, into a card of its own that it links to.

- The judge checks what a split checks, and that each new doc is a card.
- Then a judge model is asked whether each card holds one concept, the
  one its title names. A no goes back to the agent with its reason
  ([asked again](judge-hunks.md#asked-again)).
