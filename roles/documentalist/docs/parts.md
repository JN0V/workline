---
sources: [internal/builtin/documentalist/parts.go, internal/builtin/documentalist/documentalist.go]
checked: 7eadd606
verified: agent:claude-code
---
# Documentalist — judged in parts

Part of [the agent's tasks](tasks.md), in
[the documentalist](../README.md).
[ADR-0009](../../../docs/adr/0009-docs-far-behind-are-judged-in-parts.md),
on by default since
[#82](https://github.com/JN0V/workline/issues/82#issuecomment-6163487736):
a suspect or stale doc whose sources, as they are now, do not fit a task
is judged in parts instead of going to a person. `judge-in-parts: false`
sends it to a person; `parts-max-per-run` (16) caps the parts a run asks.

- **Only when gardening**
  ([#458](https://github.com/JN0V/workline/issues/458)): on a merge
  request, at the push or the release, such a doc is left for the next
  gardening, which judges it in parts, and its finding says so.
- **Why**: judged on each merge request touching its sources, one doc
  cost 6 or 7 calls, 100k to 130k tokens in, a merge request; the
  gardening judges it once.
- **A doc judged on its change alone**
  ([#462](https://github.com/JN0V/workline/issues/462)): its sources do
  not fit a task, what changed in them does. A merge request judges it on
  that change, which leaves its other claims unread; gardening reads it
  in parts against all its sources, as they are now.
  - A `judged` that change earned does not hold it; a `judged-in-parts`
    since its sources last changed does.
  - One parts cannot read ([below](#not-judged-in-parts)) is judged on
    its change, as before.
  - A record kept for it on a merged merge request is not written: the
    parts record when it was read
    ([gardening](gardening.md#records-kept-on-merged-merge-requests)).
  - What a night then costs:
    [ADR-0009, amended](../../../docs/adr/0009-docs-far-behind-are-judged-in-parts.md#amendment-2026-10-18-a-doc-judged-on-its-change-alone).
- `parts-always`, set only to measure, judges in parts on any event.

## The parts

- Each part holds the doc whole, then a share of its source files packed
  to fill the task.
- A file larger than a part is cut into consecutive pieces, each read, so
  no line goes unread.
- Sources that are docs are left out: following a doc is propagation, not
  evidence.
- Each part answers with `claim`s only — a range of the doc's lines
  `contradicted`, `partial` or `supported`, quoting the doc and the source
  — and the engine asks each in a context of its own
  ([role contract](../../../docs/spec/role-contract.md#one-run), "In
  parts").

## Reading the claims, no AI

- A quote running across the line break into the lines it cites, from
  the line before or into the line after, is placed where it stands: a
  sentence often starts at the end of the line before. One line either
  way, no more
  ([ADR-0009, amended](../../../docs/adr/0009-docs-far-behind-are-judged-in-parts.md#amendment-2026-10-18-checked-after-parts-measured-again-on)).
- A claim whose quotes are not at the lines it cites is dropped
  (`claims-dropped`, naming the part, the quote and where it was looked
  for), for a person, who reads the doc whole: the claim dropped may have
  been the one saying what is wrong. The doc is recorded as judged all
  the same — asked again each night, its parts cost as much to the same
  end
  ([ADR-0014, step 4](../../../docs/adr/0014-checked-is-earned-by-what-was-read.md#4-acceptance-written-before-it-is-measured)).
- Contradicted and supported by no other part: wrong.
- Contradicted, and another part supports some of the same words: a
  conflict. A support of other words on the same lines disputes nothing:
  a sentence spans lines
  ([#440](https://github.com/JN0V/workline/issues/440)).
- Partial: to be read together — unless another part supports the same
  lines, which settles them.
- A support quoted from a code comment only is no support
  ([ADR-0014, amended](../../../docs/adr/0014-checked-is-earned-by-what-was-read.md#amendment-2026-10-15-what-step-3-left)):
  - it neither settles a partial nor makes a contradiction a conflict;
  - the fix is told, with the code around the comment;
  - a comment the code contradicts is reported (`comment-disagrees`).
- A line naming a name from the code that no part speaks of: `uncovered`,
  for a person.

## The fix

- One last call (`fix`) gets the doc and only the source lines the parts
  cited, and patches what is wrong.
- The judge refuses a fix that moves `checked` (`checked-moved-in-parts`).
- A quote a part judged partial is no proof
  ([#467](https://github.com/JN0V/workline/issues/467)): words taken out
  on it alone are withheld for a person (`citation-unchecked`), the rest
  applied. The task says so.
  - Why: on DomoticsCore, three parts called a line partial; the fix
    cited one of their quotes and made the line false.
  - Measured on every fix after parts kept, 2026-10-01 to 10-17: the two
    such removals both took out true words.
- The header records the commit it was judged in parts at,
  `judged-in-parts`, found wrong or not, and a doc fixed stays suspect,
  saying so: the fix saw only the parts' excerpts.
- It is not put before an agent again until one of its sources changes
  after that commit; a person clears it by reading it whole and moving
  `checked`.
- When gardening, the fix, or only the record, goes on a merge request of
  its own, `workline/documentalist/fix-<first doc>`; the doc waits while
  it is open ([gardening](gardening.md#one-merge-request-per-task)).

## When `checked` moves after parts

On by default since five evaluation runs held the bars it was to be
adopted on
([ADR-0009, amended](../../../docs/adr/0009-docs-far-behind-are-judged-in-parts.md#amendment-2026-10-18-checked-after-parts-measured-again-on);
[#482](https://github.com/JN0V/workline/issues/482));
`parts-move-checked: false` leaves every such doc to a person. The engine
moves `checked`, never the fix, when the parts found nothing wrong, in
conflict or to be read together, and:

- no claim was dropped (a part's note is one), no line is `uncovered`,
  no source is a doc;
- each word of every line stating something is quoted, where it stands,
  in a part's support from the code: each part is asked to quote a
  passage it supports whole. Headings, fences, derived blocks, HTML
  comments and link definitions state nothing; a code example does.

The header then sets `checked` and `judged-in-parts` to that commit, and
`verified: agent:documentalist`: the doc leaves the list for a person,
and the weekly sample, which cannot read it whole, lists it for a person
([weekly sample](../../../docs/weekly-sample.md)). Otherwise the finding
says what kept it ("no part supports line 14"), or that the engine's
patch is not applied: a set of proposals refused whole takes it out.

## Not judged in parts

Left to a person, saying why:

- a doc leaving less than a quarter of a task for its sources: condense or
  split it first;
- one with sources in another repository;
- one needing more than `parts-max` parts (8), reported `sources-too-wide`
  with how many it would take: narrow its sources or split the doc;
- one whose parts did not all answer (`part-unanswered`): nothing they
  found is kept, it is judged again later.

At most `parts-max-per-run` parts (16) are asked in a run; a doc past it
waits for the next round. A run asking parts asks nothing else. The docs
read longest ago go first, by the later of their `checked` and
`judged-in-parts`: a doc read last night waits behind the others, not
first by name each night its sources change
([#462](https://github.com/JN0V/workline/issues/462)).
