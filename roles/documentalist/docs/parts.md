---
sources:
  - roles/documentalist/role.yaml#settings.judge-in-parts
  - roles/documentalist/role.yaml#settings.parts-max
  - roles/documentalist/role.yaml#settings.parts-max-per-run
  - roles/documentalist/code/documentalist.go#preRun.giveSourcesNow
  - roles/documentalist/code/documentalist.go#preRun.holdJudged
  - roles/documentalist/code/documentalist.go#preRun.askSuspects
  - roles/documentalist/code/documentalist.go#preRun.askStale
  - roles/documentalist/code/documentalist.go#preRun.asksNothing
  - roles/documentalist/code/documentalist.go#preRun.inPartsTask
  - roles/documentalist/code/parts.go#leftForGardening
  - roles/documentalist/code/parts.go#partsLater
  - roles/documentalist/code/parts.go#restInParts
  - roles/documentalist/code/parts.go#holdJudgedInParts
  - roles/documentalist/code/parts.go#held
  - roles/documentalist/code/parts.go#planParts
  - roles/documentalist/code/parts.go#partsRoom
  - roles/documentalist/code/parts.go#partsTaskHeader
  - roles/documentalist/code/parts.go#tooWide
  - roles/documentalist/code/parts.go#readLongestAgoFirst
checked: e81df7b34
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
  cost 6 or 7 calls a merge
  request ([tokens](../../../docs/research/token-costs-measured.md#documentalist));
  the gardening judges it once.
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
    ([gardening](gardening-merge-requests.md#records-kept-on-merged-merge-requests)).
  - What a night then costs:
    [ADR-0009, amended](../../../docs/adr/0009-docs-far-behind-are-judged-in-parts.md#amendment-2026-10-18-a-doc-judged-on-its-change-alone).
- `parts-always`, set only to measure, judges in parts on any event.

More on other pages:

- [the parts' answers](parts-claims.md): how a run asks the parts, and
  how their claims are read, with no AI;
- [the fix after parts](parts-fix.md): the one call fixing what they
  found, and when `checked` moves.

## The parts

- Each part holds the doc whole, then a share of its source files packed
  to fill the task.
- A file larger than a part is cut into consecutive pieces, each read, so
  no line goes unread.
- Sources that are docs are left out: following a doc is propagation, not
  evidence.
- A file named twice, by two parts of it (`f.go#A`, `f.go#B`), is read
  once, whole.
- Each part answers with `claim`s only — a range of the doc's lines
  `contradicted`, `partial` or `supported`, quoting the doc and the source
  — and the engine asks each in a context of its own
  ([role contract](../../../docs/spec/role-contract.md#one-run), "In
  parts").

## Not judged in parts

Left to a person, saying why:

- a doc leaving less than a quarter of a task for its sources: split it
  first;
- one with sources in another repository;
- one needing more than `parts-max` parts (8), reported `sources-too-wide`
  with how many it would take: narrow its sources or split the doc;
- one whose parts did not all answer
  ([the parts' answers](parts-claims.md#asked-and-answered)).

A run asking parts asks nothing else. The docs read longest ago go first,
by the later of their `checked` and `judged-in-parts`: a doc read last
night waits behind the others, not first by name each night its sources
change ([#462](https://github.com/JN0V/workline/issues/462)). The cap per
run: [asked and answered](parts-claims.md#asked-and-answered).
