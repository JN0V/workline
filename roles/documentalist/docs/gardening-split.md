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
  - roles/documentalist/code/hygiene.go#Budgets
  - roles/documentalist/code/split.go#relinked
  - roles/documentalist/code/split.go#applyDoc
  - roles/documentalist/code/split.go#retargeted
  - roles/documentalist/code/split.go#headWords
  - roles/documentalist/code/reader.go#readerIntroduced
checked: e81df7b34
verified: agent:claude-code
---
# Documentalist — splitting a doc over its budget

Part of [the gardening tasks](gardening.md).

## Splitting

A doc over its budget (a doc too long first, then an agent's entry
point, a section): split it into linked pages, as a person would.

- Whole parts — a section, a list, a table — move, as written, into new
  pages beside it, named after it (`run.md`, `run-people.md`); a sentence
  and a link stay in their place.
- A link elsewhere in the repository to a part that moved follows it: the
  task lists those lines, and the patch changes where they point.
- A new page carries the doc's header: its `checked` as it is, the lines
  moved being the lines it vouched for, and those of its `sources` the
  page speaks of; nothing else.
- Nothing is written anew but pointers (a paragraph of at most 30 words
  linking to a new page, or back to the doc) and headings made of the
  doc's headings' words.
- A new page, and the doc left behind, hold at least `page-words` words
  (80): a smaller part is a fragment, and stays with its neighbour.
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
- a doc linking to a part changes more than where its links point, or
  points them elsewhere than the part's new page (`rewritten`);
- a line of the doc is found in no new page, or twice, unchanged but for
  a heading's level or where its links point — moved, not rewritten
  (`rewritten`); a reference updated in place, keeping most of its words,
  is fine;
- a link moved points elsewhere than where it did, the doc or a new page;
  a link to another site moves unchanged (`link-retargeted`);
- a line is written anew, whatever the folder's size: only pointers and
  headings of the doc's headings' words are (`written-anew`);
- a new page's `checked` is not the doc's (`checked-unread`), or it names
  a source the doc does not, or none when the doc names some
  (`sources-not-carried`), or its header holds another key, such as
  `verified` (`header-not-carried`);
- a new page, or the doc left behind, holds fewer than `page-words` words
  (`page-too-small`): a section moved whole would otherwise move again
  the next night, a stub left each time;
- a MUST or SHOULD is lost;
- a new page is not linked from the doc (`not-linked`);
- the budget problem it was for remains (`still-over-budget`);
- it brings a budget problem the docs did not have, or makes one bigger
  (`patch-introduces`) — a part over its budget already, moved whole into
  a new page, its text word for word, is the same problem moved, not a
  new one; so does a reader problem the doc and its new pages bring
  (`patch-introduces`, [reader checks](checks-docs.md#docs-for-their-reader)),
  counted together: a line moved takes its problem with it; or a link
  left pointing at a part that moved (`dead-link`).

A split or dedupe hunk quoting a doc wrong is placed as for a
suspect doc ([hunks](judge-hunks.md#hunks-that-misquote-the-doc)); one
whose place is in doubt is refused.

A split or dedupe patch moving an existing doc's `checked` is
refused (`checked-unread`): moving text gives none of its sources.
