---
sources:
  - roles/documentalist/role.yaml#settings.parts-move-checked
  - internal/builtin/documentalist/parts.go#partsFor
  - internal/builtin/documentalist/parts.go#fixTaskHeader
  - internal/builtin/documentalist/parts.go#judgedInPartsPatch
  - internal/builtin/documentalist/parts.go#holdJudgedInParts
  - internal/builtin/documentalist/parts.go#held
  - internal/builtin/documentalist/parts.go#vouchFor
  - internal/builtin/documentalist/parts.go#unsupported
  - internal/builtin/documentalist/parts.go#vouched
  - internal/builtin/documentalist/parts.go#docSources
  - internal/builtin/documentalist/parts.go#partsHeader
  - internal/builtin/documentalist/parts.go#partsSupportedWhole
checked: e2e165c2d
verified: agent:claude-code
---
# Documentalist — the fix after parts

Part of [judged in parts](parts.md).

## The fix

- One last call (`fix`) patches what is wrong:
  [what it is given](parts-claims.md#reading-the-claims-no-ai).
- What the judge refuses of it:
  [moving `checked`](judge.md#moving-checked).
- A quote a part judged partial is no proof
  ([#467](https://github.com/JN0V/workline/issues/467)): words taken out
  on it alone are withheld for a person (`citation-unchecked`), the rest
  applied ([the judge's claims](judge-claims.md#removed-words-need-a-claim)).
  The task says so.
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
  it is open ([gardening](gardening-merge-requests.md#one-merge-request-per-task)).

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
patch is not applied: a set of proposals refused whole takes it out
([role contract](../../../docs/spec/role-contract.md#one-run), "Apply").
