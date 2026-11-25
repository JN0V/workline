---
sources:
  - roles/documentalist/code/parts.go#partsFor
  - roles/documentalist/code/parts.go#readParts
  - roles/documentalist/code/parts.go#placeQuote
  - roles/documentalist/code/parts.go#partsVerdict.fixEntry
  - roles/documentalist/code/parts.go#sameWords
  - roles/documentalist/code/parts.go#claim.supports
  - roles/documentalist/code/parts.go#uncoveredSplit
checked: e81df7b34
verified: agent:claude-code
---
# Documentalist — the parts' answers

Part of [judged in parts](parts.md).

## Asked and answered

- At most `parts-max-per-run` parts (16) are asked in a run; a doc past
  it waits for the next round.
- A doc whose parts did not all answer is left to a person: nothing they
  found is kept, it is judged again later.

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
  the same, as is one whose fix changed nothing of what was found
  ([listed for a person](parts-fix.md#the-fix)) — asked again each
  night, its parts cost as much to the same end
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
  for a person, in two findings: the lines whose names its sources still
  hold, no part citing them; and those naming one none holds — gone or
  renamed in the code, or a source missing from its header.
- What is wrong, in conflict or to be read together goes to one last call
  (`fix`), with the doc and only the source lines the parts cited
  ([the fix after parts](parts-fix.md)).
